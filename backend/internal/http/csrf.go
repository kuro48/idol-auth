package http

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const spaCsrfCookieName = "idol_auth_spa_csrf"

// handleGetCSRFToken issues a double-submit CSRF token for SPA clients.
// GET /v1/auth/csrf — no authentication required.
// The token is set in a non-HttpOnly cookie so JavaScript can read it,
// and is returned as JSON. Clients send the token as X-CSRF-Token on mutations.
func (s *server) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	token, err := newCSRFToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "csrf token generation failed")
		return
	}
	secure := s.config.Security.CookieSecure && requestIsSecure(r, s.config.Security.TrustedProxies)
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- double-submit CSRF cookie must be JS-readable; Secure/SameSite are set.
		Name:     spaCsrfCookieName,
		Value:    token,
		Path:     "/",
		Domain:   s.config.Security.CookieDomain,
		HttpOnly: false, // JS must read this for the double-submit pattern
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   3600,
	})
	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
}

// validateSPACSRFToken checks the double-submit cookie pattern.
// The cookie value must match the X-CSRF-Token header (constant-time compare).
func validateSPACSRFToken(r *http.Request) bool {
	cookie, err := r.Cookie(spaCsrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	return headerToken != "" && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) == 1
}
