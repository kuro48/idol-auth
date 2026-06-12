package http

import (
	"net/http"
	"strings"
)

type loginPageData struct {
	KratosFlowURL string
	RecoveryURL   string
	AltPageURL    string
	AltPageLabel  string
	Nonce         string
}

func loginPageCSP(nonce string) string {
	return "default-src 'self'; script-src 'nonce-" + nonce + "'; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'"
}

func (s *server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.config.Ory.KratosBrowserURL, "/")
	nonce, err := newCSRFToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", loginPageCSP(nonce))
	_ = loginPageTpl.Execute(w, loginPageData{
		KratosFlowURL: base + "/self-service/login/browser",
		RecoveryURL:   base + "/self-service/recovery/browser",
		AltPageURL:    "/register",
		AltPageLabel:  "アカウント登録",
		Nonce:         nonce,
	})
}

func (s *server) handleRegistrationPage(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.config.Ory.KratosBrowserURL, "/")
	nonce, err := newCSRFToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", loginPageCSP(nonce))
	_ = registrationPageTpl.Execute(w, loginPageData{
		KratosFlowURL: base + "/self-service/registration/browser",
		AltPageURL:    "/login",
		AltPageLabel:  "ログイン",
		Nonce:         nonce,
	})
}
