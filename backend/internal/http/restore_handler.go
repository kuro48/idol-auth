package http

import (
	"crypto/subtle"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/kuro48/idol-auth/internal/domain/account"
	"github.com/kuro48/idol-auth/internal/oshi"
)

const restoreCSRFCookieName = "idol_auth_restore_csrf"

func (s *server) setRestoreCSRFCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := s.config.Security.CookieSecure && requestIsSecure(r, s.config.Security.TrustedProxies)
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- restore CSRF cookie is HttpOnly, SameSite=Strict, and Secure in production.
		Name:     restoreCSRFCookieName,
		Value:    token,
		Path:     "/account/restore",
		Domain:   s.config.Security.CookieDomain,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   1800,
	})
}

func validateRestoreCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(restoreCSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := strings.TrimSpace(r.FormValue("csrf_token"))
	return formToken != "" && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) == 1
}

type restorePageData struct {
	OshiColor    template.CSS
	DaysLeft     int
	ScheduledFor time.Time
	CSRFToken    string
	LogoutURL    string
}

func (s *server) handleRestoreAccount(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	session, err := s.authSvc.CurrentSession(r.Context(), r)
	if err != nil || !session.Authenticated || session.IdentityID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var req *account.DeletionRequest
	if s.accountSvc != nil {
		req, _ = s.accountSvc.GetDeletionRequest(r.Context(), session.IdentityID)
	}
	if req == nil || req.Status != account.DeletionStatusScheduled {
		http.Redirect(w, r, "/account/", http.StatusSeeOther)
		return
	}

	nonce, err := newCSRFToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.setRestoreCSRFCookie(w, r, nonce)

	oshiColor := template.CSS("#f472b6")
	if c := oshi.NormalizeColor(session.OshiColor); c != "" {
		oshiColor = template.CSS(c) // #nosec G203 -- NormalizeColor returns a fixed allowlist hex color.
	}
	daysLeft := int(time.Until(req.ScheduledFor).Hours()/24) + 1
	if daysLeft < 0 {
		daysLeft = 0
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = restoreAccountTpl.Execute(w, restorePageData{
		OshiColor:    oshiColor,
		DaysLeft:     daysLeft,
		ScheduledFor: req.ScheduledFor,
		CSRFToken:    nonce,
		LogoutURL:    strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/logout",
	})
}

func (s *server) handleSubmitRestore(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	session, err := s.authSvc.CurrentSession(r.Context(), r)
	if err != nil || !session.Authenticated || session.IdentityID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if !validateRestoreCSRF(r) {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}

	if s.accountSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := s.accountSvc.CancelDeletion(r.Context(), session.IdentityID, session.IdentityID); err != nil {
		http.Error(w, "復活に失敗しました。もう一度お試しください。", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/account/", http.StatusSeeOther)
}
