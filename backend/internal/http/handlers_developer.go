package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/app"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

// DeveloperAppRegService is the subset of appreg.Service used by developer-facing handlers.
type DeveloperAppRegService interface {
	Submit(ctx context.Context, ownerID string, input appreg.SubmitInput) (appreg.Request, error)
	SubmitAutoApproved(ctx context.Context, ownerID string, input appreg.SubmitInput, provision appreg.ProvisionFunc) (appreg.Request, error)
	GetForOwner(ctx context.Context, id uuid.UUID, ownerID string) (appreg.Request, error)
	ListMine(ctx context.Context, ownerID string) ([]appreg.Request, error)
	Withdraw(ctx context.Context, id uuid.UUID, ownerID string) (appreg.Request, error)
	Resubmit(ctx context.Context, id uuid.UUID, ownerID string, input appreg.SubmitInput) (appreg.Request, error)
}

// MaxSelfServiceAppsPerDeveloper caps how many live registrations a single
// developer account can hold via instant (auto-approved) issuance.
const MaxSelfServiceAppsPerDeveloper = 5

// developerCreateAppInput is the JSON body for POST /v1/developer/apps.
type developerCreateAppInput struct {
	Name                   string   `json:"name"`
	Type                   string   `json:"type"`
	Description            string   `json:"description"`
	HomepageURL            string   `json:"homepage_url"`
	PrivacyPolicyURL       string   `json:"privacy_policy_url"`
	TermsURL               string   `json:"terms_url"`
	ContactEmail           string   `json:"contact_email"`
	Organization           string   `json:"organization"`
	Purpose                string   `json:"purpose"`
	RedirectURIs           []string `json:"redirect_uris"`
	PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris"`
	Scopes                 []string `json:"scopes"`
}

// provisionedApp bundles everything created for an approved registration.
type provisionedApp struct {
	App             app.App
	Registration    app.ClientRegistration
	ManagementToken string
}

// provisionAppFromRequest creates the app record, OIDC client, and management
// token for a registration request. Shared by admin approval and instant
// self-service issuance.
func (s *server) provisionAppFromRequest(ctx context.Context, regReq appreg.Request, partyType app.PartyType, actorID string) (provisionedApp, error) {
	createdApp, err := s.adminSvc.CreateApp(ctx, app.CreateAppInput{
		Name:        regReq.Name,
		Slug:        regReq.Slug,
		Type:        app.AppType(regReq.Type),
		PartyType:   partyType,
		Description: regReq.Description,
		ActorID:     actorID,
	})
	if err != nil {
		return provisionedApp{}, fmt.Errorf("create app: %w", err)
	}

	reg, err := s.adminSvc.CreateOIDCClient(ctx, createdApp.ID, app.CreateOIDCClientInput{
		RedirectURIs:           regReq.RedirectURIs,
		PostLogoutRedirectURIs: regReq.PostLogoutRedirectURIs,
		Scopes:                 regReq.Scopes,
		ActorID:                actorID,
	})
	if err != nil {
		return provisionedApp{}, fmt.Errorf("create oidc client: %w", err)
	}

	managementToken, err := s.adminSvc.IssueManagementToken(ctx, createdApp.ID, actorID)
	if err != nil {
		return provisionedApp{}, fmt.Errorf("issue management token: %w", err)
	}

	return provisionedApp{App: createdApp, Registration: reg, ManagementToken: managementToken}, nil
}

func (s *server) developerAppLimitReached(ctx context.Context, identityID string) (bool, error) {
	mine, err := s.developerAppRegSvc.ListMine(ctx, identityID)
	if err != nil {
		return false, err
	}
	active := 0
	for _, req := range mine {
		switch req.Status {
		case appreg.StatusRejected, appreg.StatusWithdrawn:
		default:
			active++
		}
	}
	return active >= MaxSelfServiceAppsPerDeveloper, nil
}

// handleDeveloperCreateApp handles POST /v1/developer/apps: instant
// self-service registration with auto-approval.
func (s *server) handleDeveloperCreateApp(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil || s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body developerCreateAppInput
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	contactEmail := body.ContactEmail
	if contactEmail == "" {
		contactEmail = session.Email
	}
	input := appreg.SubmitInput{
		Name:                   body.Name,
		Type:                   body.Type,
		Description:            body.Description,
		HomepageURL:            body.HomepageURL,
		PrivacyPolicyURL:       body.PrivacyPolicyURL,
		TermsURL:               body.TermsURL,
		ContactEmail:           contactEmail,
		Organization:           body.Organization,
		Purpose:                body.Purpose,
		RedirectURIs:           body.RedirectURIs,
		PostLogoutRedirectURIs: body.PostLogoutRedirectURIs,
		Scopes:                 body.Scopes,
		SelfService:            true,
	}

	overLimit, err := s.developerAppLimitReached(r.Context(), session.IdentityID)
	if err != nil {
		slog.ErrorContext(r.Context(), "instant app: count registrations", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if overLimit {
		writeError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("app limit reached: a developer account can hold at most %d registrations", MaxSelfServiceAppsPerDeveloper))
		return
	}

	var provisioned provisionedApp
	approved, err := s.developerAppRegSvc.SubmitAutoApproved(r.Context(), session.IdentityID, input,
		func(regReq appreg.Request) (uuid.UUID, uuid.UUID, error) {
			p, perr := s.provisionAppFromRequest(r.Context(), regReq, app.PartyTypeThird, session.IdentityID)
			if perr != nil {
				return uuid.Nil, uuid.Nil, perr
			}
			provisioned = p
			return p.App.ID, p.Registration.Client.ID, nil
		})
	if err != nil {
		slog.ErrorContext(r.Context(), "instant app registration failed",
			"identity_id", session.IdentityID, "error", err)
		writeInstantRegError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"request":          approved,
		"app":              provisioned.App,
		"client":           provisioned.Registration.Client,
		"client_secret":    provisioned.Registration.ClientSecret,
		"management_token": provisioned.ManagementToken,
	})
}

func (s *server) handleListMyAppRequests(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	reqs, err := s.developerAppRegSvc.ListMine(r.Context(), session.IdentityID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list my app requests", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if reqs == nil {
		reqs = []appreg.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": reqs})
}

func (s *server) handleSubmitAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var input appreg.SubmitInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	input.SelfService = false
	req, err := s.developerAppRegSvc.Submit(r.Context(), session.IdentityID, input)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (s *server) handleGetMyAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.developerAppRegSvc.GetForOwner(r.Context(), id, session.IdentityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *server) handleWithdrawMyAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.developerAppRegSvc.Withdraw(r.Context(), id, session.IdentityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *server) handleResubmitMyAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var input appreg.SubmitInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req, err := s.developerAppRegSvc.Resubmit(r.Context(), id, session.IdentityID, input)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// writeInstantRegError maps errors from both the appreg and app domains.
func writeInstantRegError(w http.ResponseWriter, err error) {
	if isAppRegError(err) {
		writeAppRegError(w, err)
		return
	}
	writeDomainError(w, err)
}

func isAppRegError(err error) bool {
	for _, target := range []error{
		appreg.ErrInvalidName, appreg.ErrInvalidType, appreg.ErrInvalidDescription,
		appreg.ErrInvalidPurpose, appreg.ErrInvalidEmail, appreg.ErrInvalidRedirectURI,
		appreg.ErrInvalidURL, appreg.ErrRedirectURIRequired, appreg.ErrScopeNotAllowed,
		appreg.ErrDuplicatePending, appreg.ErrRequestNotFound, appreg.ErrNotOwner,
		appreg.ErrAlreadyDecided, appreg.ErrInvalidTransition,
		appreg.ErrCannotResubmit, appreg.ErrCannotWithdraw,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// sameOrigin reports whether u and allowedRaw have the same scheme and host.
// Used by public_browser.go to validate Kratos redirect URLs.
func sameOrigin(u *url.URL, allowedRaw string) bool {
	allowed, err := url.Parse(strings.TrimSpace(allowedRaw))
	if err != nil || allowed.Host == "" {
		return false
	}
	return strings.EqualFold(u.Scheme, allowed.Scheme) && strings.EqualFold(u.Host, allowed.Host)
}
