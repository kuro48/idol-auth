package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kuro48/idol-auth/internal/domain/account"
	"github.com/kuro48/idol-auth/internal/domain/app"
)

// wantsJSON returns true when the caller expects a JSON response.
// Defaults to true when Accept is absent or wildcard to preserve
// backwards compatibility for programmatic clients that do not set Accept.
func wantsJSON(r *http.Request) bool {
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		return true
	}
	accept := r.Header.Get("Accept")
	if accept == "" || accept == "*/*" {
		return true
	}
	return strings.Contains(accept, "application/json")
}

const maxRequestBodyBytes = 1 << 20 // 1 MiB

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrChallengeRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrConsentSessionMismatch):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		writeError(w, http.StatusBadGateway, "auth upstream error")
	}
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrInvalidAppName),
		errors.Is(err, app.ErrInvalidAppSlug),
		errors.Is(err, app.ErrInvalidAppType),
		errors.Is(err, app.ErrInvalidPartyType),
		errors.Is(err, app.ErrInvalidClientName),
		errors.Is(err, app.ErrInvalidClientType),
		errors.Is(err, app.ErrInvalidTokenEndpointAuthMethod),
		errors.Is(err, app.ErrInvalidRedirectURI),
		errors.Is(err, app.ErrRedirectURIsRequired),
		errors.Is(err, app.ErrRedirectURIsNotAllowed),
		errors.Is(err, app.ErrOpenIDScopeRequired),
		errors.Is(err, app.ErrConfidentialClientRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, app.ErrAppNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, app.ErrAppDisabled):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, account.ErrDeletionRequestNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, app.ErrAppNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeUserRefError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, errAmbiguousUserRef):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadGateway, "failed to resolve user")
	}
}
