package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/app"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

// AdminAppRegService is the subset of appreg.Service used by admin review handlers.
type AdminAppRegService interface {
	ListForAdmin(ctx context.Context, statusFilter string) ([]appreg.Request, error)
	GetByID(ctx context.Context, id uuid.UUID) (appreg.Request, error)
	BeginReview(ctx context.Context, id uuid.UUID, reviewerID string) (appreg.Request, error)
	Approve(ctx context.Context, id uuid.UUID, reviewerID string, appID, clientID uuid.UUID) (appreg.Request, error)
	Reject(ctx context.Context, id uuid.UUID, reviewerID, note string) (appreg.Request, error)
	RequestChanges(ctx context.Context, id uuid.UUID, reviewerID, note string) (appreg.Request, error)
}

// handleListAdminAppRequests handles GET /v1/admin/app-requests.
func (s *server) handleListAdminAppRequests(w http.ResponseWriter, r *http.Request) {
	if s.adminAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	statusFilter := r.URL.Query().Get("status")
	reqs, err := s.adminAppRegSvc.ListForAdmin(r.Context(), statusFilter)
	if err != nil {
		slog.ErrorContext(r.Context(), "list admin app requests", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if reqs == nil {
		reqs = []appreg.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": reqs})
}

// handleGetAdminAppRequest handles GET /v1/admin/app-requests/{id}.
func (s *server) handleGetAdminAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.adminAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.adminAppRegSvc.GetByID(r.Context(), id)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// approveAppRegInput is the optional JSON body for POST .../approve.
type approveAppRegInput struct {
	PartyType string `json:"party_type"` // defaults to "third_party"
}

// handleApproveAppRequest handles POST /v1/admin/app-requests/{id}/approve.
//
// Approval sequence:
//  1. Create app record  (adminSvc.CreateApp)
//  2. Create OIDC client (adminSvc.CreateOIDCClient)
//  3. Issue management token (adminSvc.IssueManagementToken)
//  4. Mark request approved (adminAppRegSvc.Approve)
//
// client_secret is returned once only and never stored in plain text.
func (s *server) handleApproveAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.adminAppRegSvc == nil || s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var body approveAppRegInput
	if !decodeOptionalJSON(w, r, &body) {
		return
	}

	partyType := app.PartyType(body.PartyType)
	if partyType == "" {
		partyType = app.PartyTypeThird
	}
	if partyType != app.PartyTypeFirst && partyType != app.PartyTypeThird {
		writeError(w, http.StatusBadRequest, "party_type must be first_party or third_party")
		return
	}

	actorID := adminActorIDFromContext(r.Context())
	regReq, err := s.adminAppRegSvc.BeginReview(r.Context(), id, actorID)
	if err != nil {
		writeAppRegError(w, err)
		return
	}

	createdApp, err := s.adminSvc.CreateApp(r.Context(), app.CreateAppInput{
		Name:        regReq.Name,
		Slug:        regReq.Slug,
		Type:        app.AppType(regReq.Type),
		PartyType:   partyType,
		Description: regReq.Description,
		ActorID:     actorID,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "approve: create app", "request_id", id, "error", err)
		writeDomainError(w, err)
		return
	}

	reg, err := s.adminSvc.CreateOIDCClient(r.Context(), createdApp.ID, app.CreateOIDCClientInput{
		RedirectURIs:           regReq.RedirectURIs,
		PostLogoutRedirectURIs: regReq.PostLogoutRedirectURIs,
		Scopes:                 regReq.Scopes,
		ActorID:                actorID,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "approve: create oidc client", "request_id", id, "app_id", createdApp.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create OIDC client")
		return
	}

	managementToken, err := s.adminSvc.IssueManagementToken(r.Context(), createdApp.ID, actorID)
	if err != nil {
		slog.ErrorContext(r.Context(), "approve: issue management token", "request_id", id, "app_id", createdApp.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to issue management token")
		return
	}

	approved, err := s.adminAppRegSvc.Approve(r.Context(), id, actorID, createdApp.ID, reg.Client.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "approve: mark approved", "request_id", id, "error", err)
		writeAppRegError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"request":          approved,
		"app":              createdApp,
		"client":           reg.Client,
		"client_secret":    reg.ClientSecret,
		"management_token": managementToken,
	})
}

// handleRejectAppRequest handles POST /v1/admin/app-requests/{id}/reject.
func (s *server) handleRejectAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.adminAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var body struct {
		Note string `json:"note"`
	}
	if !decodeOptionalJSON(w, r, &body) {
		return
	}

	actorID := adminActorIDFromContext(r.Context())
	rejected, err := s.adminAppRegSvc.Reject(r.Context(), id, actorID, body.Note)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rejected)
}

// handleRequestChangesAppRequest handles POST /v1/admin/app-requests/{id}/request-changes.
func (s *server) handleRequestChangesAppRequest(w http.ResponseWriter, r *http.Request) {
	if s.adminAppRegSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "app registration service unavailable")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var body struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Note == "" {
		writeError(w, http.StatusBadRequest, "note is required when requesting changes")
		return
	}

	actorID := adminActorIDFromContext(r.Context())
	updated, err := s.adminAppRegSvc.RequestChanges(r.Context(), id, actorID, body.Note)
	if err != nil {
		writeAppRegError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil || r.ContentLength == 0 {
		return true
	}
	if err := decodeJSON(w, r, dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
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
