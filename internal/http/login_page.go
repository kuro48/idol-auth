package http

import (
	"net/http"
	"strings"
)

type loginPageData struct {
	KratosFlowURL string
	AltPageURL    string
	AltPageLabel  string
}

const loginPageCSP = "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; " +
	"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'"

func (s *server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.config.Ory.KratosBrowserURL, "/")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", loginPageCSP)
	_ = loginPageTpl.Execute(w, loginPageData{
		KratosFlowURL: base + "/self-service/login/browser",
		AltPageURL:    "/register",
		AltPageLabel:  "アカウント登録",
	})
}

func (s *server) handleRegistrationPage(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.config.Ory.KratosBrowserURL, "/")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", loginPageCSP)
	_ = registrationPageTpl.Execute(w, loginPageData{
		KratosFlowURL: base + "/self-service/registration/browser",
		AltPageURL:    "/login",
		AltPageLabel:  "ログイン",
	})
}
