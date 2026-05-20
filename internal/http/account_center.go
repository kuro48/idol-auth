package http

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

type accountCenterData struct {
	Email             string
	Phone             string
	IdentityID        string
	LogoutURL         string
	KratosSettingsURL string
	DisplayName       string
	Initials          string
	OshiColor         template.CSS
}

func (s *server) accountUIAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		session, err := s.authSvc.CurrentSession(r.Context(), r)
		if err != nil || !session.Authenticated || session.IdentityID == "" {
			http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), accountIdentityIDKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) handleAccountCenter(w http.ResponseWriter, r *http.Request) {
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	oshiColor := template.CSS("#1740c9")
	if c := session.OshiColor; c != "" {
		oshiColor = template.CSS(c)
	}
	setAccountCenterHeaders(w)
	_ = accountCenterTpl.Execute(w, accountCenterData{
		Email:             session.Email,
		Phone:             session.Phone,
		IdentityID:        session.IdentityID,
		LogoutURL:         strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/logout",
		KratosSettingsURL: "/settings",
		DisplayName:       session.DisplayName,
		Initials:          initials(session.DisplayName, session.Email),
		OshiColor:         oshiColor,
	})
}

func setAccountCenterHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; "+
			"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
}

func initials(displayName, email string) string {
	s := displayName
	if s == "" && email != "" {
		s = strings.SplitN(email, "@", 2)[0]
	}
	if s == "" {
		return "?"
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return "?"
	}
	var sb strings.Builder
	for i, w := range words {
		if i >= 2 {
			break
		}
		runes := []rune(w)
		if len(runes) > 0 {
			sb.WriteRune(runes[0])
		}
	}
	result := strings.ToUpper(sb.String())
	if result == "" {
		return "?"
	}
	return result
}

func (s *server) kratosLoginURL(returnPath string) string {
	u, _ := url.Parse(strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/login/browser")
	q := u.Query()
	q.Set("return_to", s.config.App.BaseURL+returnPath)
	u.RawQuery = q.Encode()
	return u.String()
}
