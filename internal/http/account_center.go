package http

import (
	"context"
	"net/http"
)

type accountCenterData struct {
	Email      string
	IdentityID string
	LogoutURL  string
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
	setAccountCenterHeaders(w)
	_ = accountCenterTpl.Execute(w, accountCenterData{
		Email:      session.Email,
		IdentityID: session.IdentityID,
		LogoutURL:  "/v1/auth/logout",
	})
}

func setAccountCenterHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; "+
			"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
}
