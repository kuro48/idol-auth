package http

import (
	"net/http"
	"net/url"
	"strings"

	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
)

func (s *server) adminUIAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		session, err := s.authSvc.CurrentSession(r.Context(), r)
		if err != nil || !session.Authenticated {
			http.Redirect(w, r, s.kratosLoginURL(r.RequestURI), http.StatusSeeOther)
			return
		}
		if !adminSessionMFASatisfied(session) {
			http.Redirect(w, r, s.kratosSettingsURL(r.RequestURI), http.StatusSeeOther)
			return
		}
		if !emailAllowed(s.config.Admin.AllowedEmails, session.Email) && !roleAllowed(s.config.Admin.AllowedRoles, session.Roles) {
			setAdminUIHeaders(w)
			w.WriteHeader(http.StatusForbidden)
			_ = adminUITpl.ExecuteTemplate(w, "error", map[string]string{
				"Title": "アクセス権限がありません",
				"Msg":   "管理者メールアドレスまたはロールが必要です。",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) kratosLoginURL(returnPath string) string {
	u, _ := url.Parse(strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/login/browser")
	q := u.Query()
	q.Set("return_to", s.config.App.BaseURL+returnPath)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *server) kratosSettingsURL(returnPath string) string {
	u, _ := url.Parse(strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/settings/browser")
	q := u.Query()
	q.Set("return_to", s.config.App.BaseURL+returnPath)
	u.RawQuery = q.Encode()
	return u.String()
}

func setAdminUIHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; "+
			"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
}

type adminUIBase struct {
	Email string
	Nav   string
}

type adminUIOverviewData struct {
	adminUIBase
	AppCount   int
	RecentLogs []admindomain.AuditLog
}

type adminUIAppsData struct {
	adminUIBase
	Apps []app.App
}

type adminUIUsersData struct {
	adminUIBase
	Query string
	State string
	Users []admindomain.Identity
}

type adminUIAuditData struct {
	adminUIBase
	EventType  string
	ActorID    string
	TargetType string
	Logs       []admindomain.AuditLog
	HasMore    bool
}

func (s *server) handleAdminUIOverview(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		http.Error(w, "admin service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, _ := s.authSvc.CurrentSession(r.Context(), r)
	apps, _ := s.adminSvc.ListApps(r.Context())
	logs, _ := s.adminSvc.ListAuditLogs(r.Context(), admindomain.ListAuditLogsInput{Limit: 20})
	setAdminUIHeaders(w)
	_ = adminUITpl.ExecuteTemplate(w, "overview", adminUIOverviewData{
		adminUIBase: adminUIBase{Email: session.Email, Nav: "overview"},
		AppCount:    len(apps),
		RecentLogs:  logs,
	})
}

func (s *server) handleAdminUIApps(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		http.Error(w, "admin service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, _ := s.authSvc.CurrentSession(r.Context(), r)
	apps, _ := s.adminSvc.ListApps(r.Context())
	setAdminUIHeaders(w)
	_ = adminUITpl.ExecuteTemplate(w, "apps", adminUIAppsData{
		adminUIBase: adminUIBase{Email: session.Email, Nav: "apps"},
		Apps:        apps,
	})
}

func (s *server) handleAdminUIUsers(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		http.Error(w, "admin service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, _ := s.authSvc.CurrentSession(r.Context(), r)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	filter := admindomain.SearchIdentitiesInput{CredentialsIdentifier: query}
	switch state {
	case "active":
		filter.State = admindomain.IdentityStateActive
	case "inactive":
		filter.State = admindomain.IdentityStateInactive
	}
	users, _ := s.adminSvc.SearchIdentities(r.Context(), filter)
	setAdminUIHeaders(w)
	_ = adminUITpl.ExecuteTemplate(w, "users", adminUIUsersData{
		adminUIBase: adminUIBase{Email: session.Email, Nav: "users"},
		Query:       query,
		State:       state,
		Users:       users,
	})
}

func (s *server) handleAdminUIAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		http.Error(w, "admin service unavailable", http.StatusServiceUnavailable)
		return
	}
	session, _ := s.authSvc.CurrentSession(r.Context(), r)
	eventType := strings.TrimSpace(r.URL.Query().Get("event_type"))
	actorID := strings.TrimSpace(r.URL.Query().Get("actor_id"))
	targetType := strings.TrimSpace(r.URL.Query().Get("target_type"))
	const pageSize = 50
	logs, _ := s.adminSvc.ListAuditLogs(r.Context(), admindomain.ListAuditLogsInput{
		EventType:  eventType,
		ActorID:    actorID,
		TargetType: targetType,
		Limit:      pageSize + 1,
	})
	hasMore := len(logs) > pageSize
	if hasMore {
		logs = logs[:pageSize]
	}
	setAdminUIHeaders(w)
	_ = adminUITpl.ExecuteTemplate(w, "audit-logs", adminUIAuditData{
		adminUIBase: adminUIBase{Email: session.Email, Nav: "audit-logs"},
		EventType:   eventType,
		ActorID:     actorID,
		TargetType:  targetType,
		Logs:        logs,
		HasMore:     hasMore,
	})
}
