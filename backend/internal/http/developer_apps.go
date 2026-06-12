package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/app"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

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
// self-service issuance. The client_secret inside the registration is returned
// once only and never stored in plain text.
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

// handleDeveloperCreateApp handles POST /v1/developer/apps: instant
// self-service registration. The request is auto-approved after provisioning;
// credentials are returned once in the response and never again.
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

// handleDeveloperAppsCreateHTML handles the browser form POST /developer/apps.
// Same instant issuance as the JSON endpoint, but renders the one-time
// credentials page on success.
func (s *server) handleDeveloperAppsCreateHTML(w http.ResponseWriter, r *http.Request) {
	if s.developerAppRegSvc == nil || s.adminSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if !validateDeveloperCSRF(r) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}

	input := parseSubmitInputFromForm(r)
	input.SelfService = true
	if input.ContactEmail == "" {
		input.ContactEmail = session.Email
	}

	renderFormError := func(message string) {
		csrfToken, _ := s.newDeveloperCSRFToken(w, r)
		setAccountCenterHeaders(w, "")
		_ = developerRequestFormTpl.Execute(w, developerRequestFormData{
			devPageBase: devPageBaseFromSession(s, session),
			CSRFToken:   csrfToken,
			Error:       message,
		})
	}

	overLimit, err := s.developerAppLimitReached(r.Context(), session.IdentityID)
	if err != nil {
		slog.ErrorContext(r.Context(), "instant app: count registrations", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if overLimit {
		renderFormError(fmt.Sprintf("登録できるアプリは最大 %d 件です。不要なアプリを取り下げてから再度お試しください", MaxSelfServiceAppsPerDeveloper))
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
		renderFormError(friendlyAppRegError(err))
		return
	}

	setAccountCenterHeaders(w, "")
	_ = developerCredentialsTpl.Execute(w, developerCredentialsData{
		devPageBase:     devPageBaseFromSession(s, session),
		AppName:         approved.Name,
		ClientID:        provisioned.Registration.Client.HydraClientID,
		ClientSecret:    provisioned.Registration.ClientSecret,
		ManagementToken: provisioned.ManagementToken,
		DetailURL:       "/developer/app-requests/" + approved.ID.String(),
	})
}

// developerAppLimitReached counts registrations that occupy quota: everything
// except rejected and withdrawn ones.
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
