package http

import (
	"net/http"
	"strings"
)

type loginPageData struct {
	LoginURL        string
	RegistrationURL string
	LogoutURL       string
}

func (s *server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.config.Ory.KratosBrowserURL, "/")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; "+
			"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
	_ = loginPageTpl.Execute(w, loginPageData{
		LoginURL:        base + "/self-service/login/browser",
		RegistrationURL: base + "/self-service/registration/browser",
		LogoutURL:       base + "/logout",
	})
}
