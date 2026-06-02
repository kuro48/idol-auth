package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	sessions, err := s.sessionMgr.ListSessionsForIdentity(r.Context(), session.IdentityID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to list sessions")
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	_, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "sessionId is required")
		return
	}
	if err := s.sessionMgr.RevokeSession(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusBadGateway, "failed to revoke session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
