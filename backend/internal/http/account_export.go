package http

import (
	"net/http"
)

func (s *server) handleExportAccount(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	export, err := s.accountSvc.ExportAccountData(r.Context(), session.IdentityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to export account data")
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="account-export.json"`)
	writeJSON(w, http.StatusOK, export)
}
