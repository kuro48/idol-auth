package http

import (
	"net/http"
)

func (s *server) handleGetAppStats(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	stats, err := s.accountSvc.GetAppStats(r.Context(), appActor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve app stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
