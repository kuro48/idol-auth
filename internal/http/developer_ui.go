package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

const developerUISessionKey contextKey = "developer_ui_session"

type developerUIListData struct {
	Email     string
	LogoutURL string
	Requests  []appreg.Request
}

type developerUIDetailData struct {
	Email     string
	LogoutURL string
	Request   appreg.Request
}

func (s *server) developerUIAuth(next http.Handler) http.Handler {
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
		ctx := context.WithValue(r.Context(), developerUISessionKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func developerUISessionFromContext(ctx context.Context) (SessionView, bool) {
	session, ok := ctx.Value(developerUISessionKey).(SessionView)
	return session, ok
}

func setDeveloperUIHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; "+
			"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
}

func (s *server) developerLogoutURL() string {
	return strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/logout"
}

func (s *server) handleDeveloperUIList(w http.ResponseWriter, r *http.Request) {
	session, ok := developerUISessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	if s.developerSvc == nil {
		renderDeveloperUIError(w, http.StatusServiceUnavailable, "開発者サービスが利用できません")
		return
	}
	requests, err := s.developerSvc.ListMine(r.Context(), session.IdentityID)
	if err != nil {
		renderDeveloperUIError(w, http.StatusBadGateway, "申請一覧の取得に失敗しました")
		return
	}
	setDeveloperUIHeaders(w)
	_ = developerUITpl.ExecuteTemplate(w, "dev-list", developerUIListData{
		Email:     session.Email,
		LogoutURL: s.developerLogoutURL(),
		Requests:  requests,
	})
}

func (s *server) handleDeveloperUINew(w http.ResponseWriter, r *http.Request) {
	session, ok := developerUISessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	setDeveloperUIHeaders(w)
	_ = developerUITpl.ExecuteTemplate(w, "dev-new", developerUIListData{
		Email:     session.Email,
		LogoutURL: s.developerLogoutURL(),
	})
}

func (s *server) handleDeveloperUIDetail(w http.ResponseWriter, r *http.Request) {
	session, ok := developerUISessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
		return
	}
	if s.developerSvc == nil {
		renderDeveloperUIError(w, http.StatusServiceUnavailable, "開発者サービスが利用できません")
		return
	}
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, err := s.developerSvc.GetForOwner(r.Context(), id, session.IdentityID)
	if err != nil {
		renderDeveloperUIError(w, http.StatusNotFound, "申請が見つかりません")
		return
	}
	setDeveloperUIHeaders(w)
	_ = developerUITpl.ExecuteTemplate(w, "dev-detail", developerUIDetailData{
		Email:     session.Email,
		LogoutURL: s.developerLogoutURL(),
		Request:   req,
	})
}

func renderDeveloperUIError(w http.ResponseWriter, status int, message string) {
	setDeveloperUIHeaders(w)
	w.WriteHeader(status)
	_ = developerUITpl.ExecuteTemplate(w, "dev-error", map[string]string{
		"Message": message,
	})
}
