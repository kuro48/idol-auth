package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (s *server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if s.readiness != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.readiness.Ready(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, ProviderView{
		LoginURL:        strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/login/browser",
		RegistrationURL: strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/registration/browser",
		RecoveryURL:     strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/recovery/browser",
		VerificationURL: strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/verification/browser",
		SettingsURL:     strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/settings/browser",
		LogoutURL:       strings.TrimRight(s.config.Ory.HydraBrowserURL, "/") + "/oauth2/sessions/logout",
	})
}

func (s *server) handleSession(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
		return
	}
	session, err := s.authSvc.CurrentSession(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve session")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *server) handleThemePreference(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
		return
	}
	themeSvc, ok := s.authSvc.(themePreferenceService)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "theme preference unavailable")
		return
	}
	var req struct {
		OshiColor string `json:"oshi_color"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	session, err := themeSvc.UpdateThemePreference(r.Context(), r, req.OshiColor)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOshiColor):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoActiveSession):
			writeError(w, http.StatusUnauthorized, "authentication required")
		case errors.Is(err, ErrThemePreferenceUnavailable):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		default:
			writeError(w, http.StatusBadGateway, "failed to persist theme preference")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"oshi_color": session.OshiColor,
	})
}

func (s *server) handleLogoutStart(w http.ResponseWriter, r *http.Request) {
	baseURL := strings.TrimRight(s.config.App.BaseURL, "/") + "/"
	if s.authSvc != nil {
		// Best-effort: invalidate the Kratos session server-side so the browser
		// never navigates to the Kratos domain.
		_ = s.authSvc.LogoutSession(r.Context(), r)
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]string{"logout_url": baseURL})
		return
	}
	http.Redirect(w, r, baseURL, http.StatusSeeOther)
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
		return
	}
	result, err := s.authSvc.HandleLogin(r.Context(), r, r.URL.Query().Get("login_challenge"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if !validateRedirectURL(result.RedirectTo) {
		writeError(w, http.StatusBadGateway, "invalid redirect from upstream")
		return
	}
	http.Redirect(w, r, result.RedirectTo, http.StatusFound)
}

func (s *server) handleConsent(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
		return
	}
	result, err := s.authSvc.HandleConsent(r.Context(), r, r.URL.Query().Get("consent_challenge"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if result.Prompt != nil {
		secureCookies := s.config.Security.CookieSecure && requestIsSecure(r, s.config.Security.TrustedProxies)
		if err := writeConsentPage(w, result.Prompt, secureCookies, s.config.Security.CookieDomain); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to render consent page")
		}
		return
	}
	if !validateRedirectURL(result.RedirectTo) {
		writeError(w, http.StatusBadGateway, "invalid redirect from upstream")
		return
	}
	http.Redirect(w, r, result.RedirectTo, http.StatusFound)
}

func (s *server) handleConsentSubmit(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	if !validateConsentCSRF(r) {
		writeError(w, http.StatusForbidden, "csrf validation failed")
		return
	}
	action := r.Form.Get("action")
	if action != "accept" && action != "deny" {
		writeError(w, http.StatusBadRequest, "invalid consent action")
		return
	}
	result, err := s.authSvc.SubmitConsent(r.Context(), r, r.Form.Get("consent_challenge"), ConsentDecisionInput{
		Accept: action == "accept",
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if !validateRedirectURL(result.RedirectTo) {
		writeError(w, http.StatusBadGateway, "invalid redirect from upstream")
		return
	}
	secureCookies := s.config.Security.CookieSecure && requestIsSecure(r, s.config.Security.TrustedProxies)
	clearConsentCSRFCookie(w, secureCookies, s.config.Security.CookieDomain)
	http.Redirect(w, r, result.RedirectTo, http.StatusFound)
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.URL.Query().Get("logout_challenge")) == "" {
		s.handleLogoutStart(w, r)
		return
	}
	if s.authSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
		return
	}
	result, err := s.authSvc.HandleLogout(r.Context(), r.URL.Query().Get("logout_challenge"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if !validateRedirectURL(result.RedirectTo) {
		writeError(w, http.StatusBadGateway, "invalid redirect from upstream")
		return
	}
	http.Redirect(w, r, result.RedirectTo, http.StatusFound)
}
