package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kuro48/idol-auth/internal/domain/loginhistory"
)

// handleListLoginHistory returns recent observed Kratos sessions for the
// authenticated identity. Optional ?limit query (1-200) controls page size.
func (s *server) handleListLoginHistory(w http.ResponseWriter, r *http.Request) {
	if s.loginHistorySvc == nil {
		writeError(w, http.StatusServiceUnavailable, "login history service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	events, err := s.loginHistorySvc.List(r.Context(), session.IdentityID, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to load login history")
		return
	}
	if events == nil {
		events = []loginhistory.Event{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}
