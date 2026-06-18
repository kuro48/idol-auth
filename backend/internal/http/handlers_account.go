package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/account"
	"github.com/kuro48/idol-auth/internal/domain/app"
	"github.com/kuro48/idol-auth/internal/domain/profile"
)

func (s *server) handleAccountOverview(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	memberships, err := s.accountSvc.ListMembershipsForIdentity(r.Context(), session.IdentityID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to load account memberships")
		return
	}
	deletionRequest, err := s.accountSvc.GetDeletionRequest(r.Context(), session.IdentityID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"identity_id":      session.IdentityID,
		"email":            session.Email,
		"memberships":      memberships,
		"deletion_request": deletionRequest,
		"authenticated":    true,
		"subject":          session.Subject,
	})
}

func (s *server) handleDisconnectAccountApp(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}
	if err := s.accountSvc.DisconnectIdentityFromApp(r.Context(), session.IdentityID, appID, session.IdentityID); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleGetDeletionRequest(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	request, err := s.accountSvc.GetDeletionRequest(r.Context(), session.IdentityID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"request": request})
}

func (s *server) handleScheduleDeletion(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	request, err := s.accountSvc.ScheduleDeletion(r.Context(), session.IdentityID, session.IdentityID, req.Reason)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"request": request})
}

func (s *server) handleCancelDeletion(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err := s.accountSvc.CancelDeletion(r.Context(), session.IdentityID, session.IdentityID); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleListAppUsers(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	items, err := s.accountSvc.ListMembershipsForApp(r.Context(), appActor.ID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app":   appActor,
		"items": items,
	})
}

func (s *server) handleRevokeAppUser(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	identityID := strings.TrimSpace(chi.URLParam(r, "identityID"))
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "identity id is required")
		return
	}
	if err := s.accountSvc.RevokeAppUser(r.Context(), appActor.ID, identityID, appActor.ID.String()); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleRegisterAppUser(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	if appActor.PartyType != app.PartyTypeFirst {
		writeError(w, http.StatusForbidden, "shared account registration is restricted to first-party apps")
		return
	}
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	result, err := s.accountSvc.RegisterIdentityForApp(r.Context(), appActor, account.RegisterIdentityInput{
		Email:       req.Email,
		DisplayName: req.DisplayName,
	}, appActor.ID.String())
	if err != nil {
		if errors.Is(err, account.ErrSharedAccountAlreadyExists) {
			writeError(w, http.StatusConflict, "shared account already exists; use the shared account sign-in flow")
			return
		}
		writeError(w, http.StatusBadGateway, "failed to register user")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *server) handleGetAppUserProfile(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil || s.profileSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	identityID := strings.TrimSpace(chi.URLParam(r, "identityID"))
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "identity id is required")
		return
	}
	if _, err := s.accountSvc.GetMembershipForApp(r.Context(), appActor.ID, identityID); err != nil {
		if errors.Is(err, account.ErrMembershipNotFound) {
			writeError(w, http.StatusNotFound, "user not found in this app")
			return
		}
		writeError(w, http.StatusBadGateway, "failed to verify membership")
		return
	}
	p, err := s.profileSvc.GetProfile(r.Context(), identityID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to load profile")
		return
	}
	writeJSON(w, http.StatusOK, p.PublicView())
}

func (s *server) handlePatchRecoveryContacts(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "profile service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(session.AuthenticatorAssuranceLevel), "aal2") {
		writeError(w, http.StatusForbidden, "mfa required for this action")
		return
	}

	var req struct {
		RecoveryEmail *string `json:"recovery_email"`
		RecoveryPhone *string `json:"recovery_phone"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.RecoveryEmail == nil && req.RecoveryPhone == nil {
		writeError(w, http.StatusBadRequest, "recovery_email or recovery_phone is required")
		return
	}
	trimStringPtr(req.RecoveryEmail)
	trimStringPtr(req.RecoveryPhone)

	input := profile.UpdateInput{
		RecoveryEmail: req.RecoveryEmail,
		RecoveryPhone: req.RecoveryPhone,
	}
	updated, err := s.profileSvc.UpdateProfile(r.Context(), session.IdentityID, input)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to update recovery contacts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recovery_email": updated.RecoveryEmail,
		"recovery_phone": updated.RecoveryPhone,
	})
}

func (s *server) handlePasswordChange(w http.ResponseWriter, r *http.Request) {
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	_ = session // identity bound via Kratos session cookie

	var req struct {
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		writeError(w, http.StatusBadRequest, "new_password is required")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		writeError(w, http.StatusBadRequest, "passwords do not match")
		return
	}

	if err := s.passwordChangeSvc.ChangePassword(r.Context(), r, req.NewPassword); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleGetEmailStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	email, verified, err := s.emailVerifSvc.GetEmailVerificationStatus(r.Context(), session.IdentityID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to check email status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email":    email,
		"verified": verified,
	})
}
