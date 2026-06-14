package http

import (
	"net/http"
	"slices"

	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
)

func (s *server) handleDeveloperRegistration(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if slices.Contains(session.Roles, "developer") || slices.Contains(session.Roles, "admin") {
		writeError(w, http.StatusConflict, "already a developer")
		return
	}
	roles := append(slices.Clone(session.Roles), "developer")
	updated, err := s.adminSvc.SetIdentityRoles(r.Context(), admindomain.SetIdentityRolesInput{
		IdentityID: session.IdentityID,
		Roles:      roles,
		ActorID:    session.IdentityID,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to register as developer")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": updated})
}
