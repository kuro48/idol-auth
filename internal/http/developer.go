package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

// DeveloperService is the subset of appreg.Service used by developer API handlers.
type DeveloperService interface {
	Submit(ctx context.Context, identityID string, input appreg.SubmitInput) (appreg.Request, error)
	GetForOwner(ctx context.Context, id uuid.UUID, identityID string) (appreg.Request, error)
	ListMine(ctx context.Context, identityID string) ([]appreg.Request, error)
	Withdraw(ctx context.Context, id uuid.UUID, identityID string) (appreg.Request, error)
	Resubmit(ctx context.Context, id uuid.UUID, identityID string, input appreg.SubmitInput) (appreg.Request, error)
}

// submitAppRegInput is the JSON body for POST /v1/developer/applications.
type submitAppRegInput struct {
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

func (in submitAppRegInput) toSubmitInput() appreg.SubmitInput {
	return appreg.SubmitInput{
		Name:                   in.Name,
		Type:                   in.Type,
		Description:            in.Description,
		HomepageURL:            in.HomepageURL,
		PrivacyPolicyURL:       in.PrivacyPolicyURL,
		TermsURL:               in.TermsURL,
		ContactEmail:           in.ContactEmail,
		Organization:           in.Organization,
		Purpose:                in.Purpose,
		RedirectURIs:           in.RedirectURIs,
		PostLogoutRedirectURIs: in.PostLogoutRedirectURIs,
		Scopes:                 in.Scopes,
	}
}

// handleSubmitAppReg handles POST /v1/developer/applications.
func (s *server) handleSubmitAppReg(w http.ResponseWriter, r *http.Request) {
	identityID := developerIdentityFromContext(r.Context())

	var body submitAppRegInput
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req, err := s.developerSvc.Submit(r.Context(), identityID, body.toSubmitInput())
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

// handleListAppRegs handles GET /v1/developer/applications.
func (s *server) handleListAppRegs(w http.ResponseWriter, r *http.Request) {
	identityID := developerIdentityFromContext(r.Context())

	reqs, err := s.developerSvc.ListMine(r.Context(), identityID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list app registrations", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if reqs == nil {
		reqs = []appreg.Request{}
	}
	writeJSON(w, http.StatusOK, reqs)
}

// handleGetAppReg handles GET /v1/developer/applications/{id}.
func (s *server) handleGetAppReg(w http.ResponseWriter, r *http.Request) {
	identityID := developerIdentityFromContext(r.Context())

	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, err := s.developerSvc.GetForOwner(r.Context(), id, identityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// handleWithdrawAppReg handles DELETE /v1/developer/applications/{id}.
func (s *server) handleWithdrawAppReg(w http.ResponseWriter, r *http.Request) {
	identityID := developerIdentityFromContext(r.Context())

	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, err := s.developerSvc.Withdraw(r.Context(), id, identityID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// handleResubmitAppReg handles PATCH /v1/developer/applications/{id}.
func (s *server) handleResubmitAppReg(w http.ResponseWriter, r *http.Request) {
	identityID := developerIdentityFromContext(r.Context())

	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var body submitAppRegInput
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req, err := s.developerSvc.Resubmit(r.Context(), id, identityID, body.toSubmitInput())
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// developerAuth requires a valid Kratos session and injects the identity ID into context.
func (s *server) developerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil {
			writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		session, err := s.authSvc.CurrentSession(r.Context(), r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to resolve session")
			return
		}
		if !session.Authenticated || session.IdentityID == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), developerIdentityIDKey, session.IdentityID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func developerIdentityFromContext(ctx context.Context) string {
	id, _ := ctx.Value(developerIdentityIDKey).(string)
	return id
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return uuid.UUID{}, false
	}
	return id, true
}

func writeAppRegError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, appreg.ErrInvalidName),
		errors.Is(err, appreg.ErrInvalidType),
		errors.Is(err, appreg.ErrInvalidDescription),
		errors.Is(err, appreg.ErrInvalidPurpose),
		errors.Is(err, appreg.ErrInvalidEmail),
		errors.Is(err, appreg.ErrInvalidRedirectURI),
		errors.Is(err, appreg.ErrInvalidURL),
		errors.Is(err, appreg.ErrRedirectURIRequired),
		errors.Is(err, appreg.ErrCannotResubmit),
		errors.Is(err, appreg.ErrCannotWithdraw):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, appreg.ErrDuplicatePending):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, appreg.ErrRequestNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, appreg.ErrNotOwner):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, appreg.ErrAlreadyDecided):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, appreg.ErrInvalidTransition):
		writeError(w, http.StatusConflict, err.Error())
	default:
		slog.Error("app registration error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
