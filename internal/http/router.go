package http

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/config"
	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
)

var (
	ErrChallengeRequired      = errors.New("challenge is required")
	ErrConsentSessionMismatch = errors.New("active session does not match consent subject")
)

type AuthAction string

const AuthActionRedirect AuthAction = "redirect"

type RouterConfig struct {
	App      config.AppConfig
	Admin    config.AdminConfig
	Ory      config.OryConfig
	Security config.SecurityConfig
	Limiter  RateLimiter // optional; nil disables rate limiting
}

type LoginFlowResult struct {
	Action     AuthAction `json:"action"`
	RedirectTo string     `json:"redirect_to"`
}

type AuthFlowResult struct {
	RedirectTo string `json:"redirect_to"`
}

type ConsentDecisionInput struct {
	Accept bool
}

type ConsentPrompt struct {
	Challenge                    string
	ClientID                     string
	ClientName                   string
	RequestedScope               []string
	RequestedAccessTokenAudience []string
}

type ConsentFlowResult struct {
	RedirectTo string         `json:"redirect_to,omitempty"`
	Prompt     *ConsentPrompt `json:"prompt,omitempty"`
}

type SessionView struct {
	Authenticated               bool     `json:"authenticated"`
	Subject                     string   `json:"subject,omitempty"`
	IdentityID                  string   `json:"identity_id,omitempty"`
	Email                       string   `json:"email,omitempty"`
	Roles                       []string `json:"roles,omitempty"`
	Methods                     []string `json:"methods,omitempty"`
	AuthenticatorAssuranceLevel string   `json:"authenticator_assurance_level,omitempty"`
}

type ProviderView struct {
	LoginURL        string `json:"login_url"`
	RegistrationURL string `json:"registration_url"`
	RecoveryURL     string `json:"recovery_url"`
	VerificationURL string `json:"verification_url"`
	SettingsURL     string `json:"settings_url"`
	LogoutURL       string `json:"logout_url"`
}

type AdminService interface {
	CreateApp(ctx context.Context, input app.CreateAppInput) (app.App, error)
	ListApps(ctx context.Context) ([]app.App, error)
	CreateOIDCClient(ctx context.Context, appID uuid.UUID, input app.CreateOIDCClientInput) (app.ClientRegistration, error)
	ListOIDCClients(ctx context.Context, appID uuid.UUID) ([]app.OIDCClient, error)
	SetIdentityRoles(ctx context.Context, input admindomain.SetIdentityRolesInput) ([]string, error)
	SearchIdentities(ctx context.Context, input admindomain.SearchIdentitiesInput) ([]admindomain.Identity, error)
	DisableIdentity(ctx context.Context, input admindomain.DisableIdentityInput) (admindomain.Identity, error)
	EnableIdentity(ctx context.Context, input admindomain.EnableIdentityInput) (admindomain.Identity, error)
	RevokeIdentitySessions(ctx context.Context, input admindomain.RevokeIdentitySessionsInput) error
	DeleteIdentity(ctx context.Context, input admindomain.DeleteIdentityInput) error
	ListAuditLogs(ctx context.Context, input admindomain.ListAuditLogsInput) ([]admindomain.AuditLog, error)
}

type AuthService interface {
	HandleLogin(ctx context.Context, r *http.Request, loginChallenge string) (LoginFlowResult, error)
	HandleConsent(ctx context.Context, r *http.Request, consentChallenge string) (ConsentFlowResult, error)
	SubmitConsent(ctx context.Context, r *http.Request, consentChallenge string, input ConsentDecisionInput) (AuthFlowResult, error)
	HandleLogout(ctx context.Context, logoutChallenge string) (AuthFlowResult, error)
	CurrentSession(ctx context.Context, r *http.Request) (SessionView, error)
}

type readinessChecker interface {
	Ready(ctx context.Context) error
}

type server struct {
	config             RouterConfig
	adminSvc           AdminService
	authSvc            AuthService
	readiness          readinessChecker
	authFailureLimiter RateLimiter // tight per-IP limiter for bootstrap token failures
}

type contextKey string

const adminActorIDKey contextKey = "admin_actor_id"

const consentCSRFCookieName = "idol_auth_consent_csrf"

func NewRouter(cfg RouterConfig, adminSvc AdminService, readiness readinessChecker, authSvc AuthService) http.Handler {
	s := &server{
		config:             cfg,
		adminSvc:           adminSvc,
		authSvc:            authSvc,
		readiness:          readiness,
		authFailureLimiter: NewInMemoryRateLimiter(5, 5*time.Minute),
	}

	r := chi.NewRouter()
	r.Use(securityHeaders)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Route("/v1/auth", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Get("/providers", s.handleProviders)
		r.Get("/session", s.handleSession)
		r.Post("/logout", s.handleLogoutStart)
		r.Get("/login", s.handleLogin)
		r.Get("/consent", s.handleConsent)
		r.Post("/consent", s.handleConsentSubmit)
		r.Get("/logout", s.handleLogout)
	})

	r.Route("/v1/admin", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.adminAuth)
		r.Get("/apps", s.handleListApps)
		r.Post("/apps", s.handleCreateApp)
		r.Get("/apps/{appID}/clients", s.handleListOIDCClients)
		r.Post("/apps/{appID}/clients", s.handleCreateOIDCClient)
		r.Get("/users", s.handleSearchIdentities)
		r.Post("/users/{identityID}/disable", s.handleDisableIdentity)
		r.Post("/users/{identityID}/enable", s.handleEnableIdentity)
		r.Post("/users/{identityID}/revoke-sessions", s.handleRevokeIdentitySessions)
		r.Delete("/users/{identityID}", s.handleDeleteIdentity)
		r.Put("/identities/{identityID}/roles", s.handleSetIdentityRoles)
		r.Get("/audit-logs", s.handleListAuditLogs)
	})

	return r
}

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

func (s *server) handleLogoutStart(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"logout_url": strings.TrimRight(s.config.Ory.HydraBrowserURL, "/") + "/oauth2/sessions/logout",
	})
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
		if err := writeConsentPage(w, result.Prompt, secureCookies); err != nil {
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
	clearConsentCSRFCookie(w, secureCookies)
	http.Redirect(w, r, result.RedirectTo, http.StatusFound)
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
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

func (s *server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Type        string `json:"type"`
		PartyType   string `json:"party_type"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	created, err := s.adminSvc.CreateApp(r.Context(), app.CreateAppInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Type:        app.AppType(req.Type),
		PartyType:   app.PartyType(req.PartyType),
		Description: req.Description,
		ActorID:     adminActorIDFromContext(r.Context()),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *server) handleListApps(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	apps, err := s.adminSvc.ListApps(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list apps")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": apps})
}

func (s *server) handleCreateOIDCClient(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}

	var req struct {
		Name                    string   `json:"name"`
		ClientType              string   `json:"client_type"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		RedirectURIs            []string `json:"redirect_uris"`
		PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris"`
		Scopes                  []string `json:"scopes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	created, err := s.adminSvc.CreateOIDCClient(r.Context(), appID, app.CreateOIDCClientInput{
		Name:                    req.Name,
		ClientType:              app.ClientType(req.ClientType),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		RedirectURIs:            req.RedirectURIs,
		PostLogoutRedirectURIs:  req.PostLogoutRedirectURIs,
		Scopes:                  req.Scopes,
		ActorID:                 adminActorIDFromContext(r.Context()),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"client":        created.Client,
		"client_secret": created.ClientSecret,
	})
}

func (s *server) handleListOIDCClients(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}

	clients, err := s.adminSvc.ListOIDCClients(r.Context(), appID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": clients})
}

func (s *server) handleSetIdentityRoles(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	identityID := chi.URLParam(r, "identityID")
	if _, err := uuid.Parse(identityID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid identity id")
		return
	}

	var req struct {
		Roles []string `json:"roles"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	roles, err := s.adminSvc.SetIdentityRoles(r.Context(), admindomain.SetIdentityRolesInput{
		IdentityID: identityID,
		Roles:      req.Roles,
		ActorID:    adminActorIDFromContext(r.Context()),
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to update identity roles")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"identity_id": identityID,
		"roles":       roles,
	})
}

func (s *server) handleSearchIdentities(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}

	filter := admindomain.SearchIdentitiesInput{
		CredentialsIdentifier: strings.TrimSpace(r.URL.Query().Get("identifier")),
	}
	switch state := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("state"))); state {
	case "":
	case string(admindomain.IdentityStateActive):
		filter.State = admindomain.IdentityStateActive
	case string(admindomain.IdentityStateInactive):
		filter.State = admindomain.IdentityStateInactive
	default:
		writeError(w, http.StatusBadRequest, "invalid identity state")
		return
	}

	identities, err := s.adminSvc.SearchIdentities(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to search identities")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": identities})
}

func (s *server) handleDisableIdentity(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID := chi.URLParam(r, "identityID")
	if _, err := uuid.Parse(identityID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid identity id")
		return
	}
	identity, err := s.adminSvc.DisableIdentity(r.Context(), admindomain.DisableIdentityInput{
		IdentityID: identityID,
		ActorID:    adminActorIDFromContext(r.Context()),
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to disable identity")
		return
	}
	writeJSON(w, http.StatusOK, identity)
}

func (s *server) handleDeleteIdentity(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID := chi.URLParam(r, "identityID")
	if _, err := uuid.Parse(identityID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid identity id")
		return
	}
	if err := s.adminSvc.DeleteIdentity(r.Context(), admindomain.DeleteIdentityInput{
		IdentityID: identityID,
		ActorID:    adminActorIDFromContext(r.Context()),
	}); err != nil {
		writeError(w, http.StatusBadGateway, "failed to delete identity")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleEnableIdentity(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID := chi.URLParam(r, "identityID")
	if _, err := uuid.Parse(identityID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid identity id")
		return
	}
	identity, err := s.adminSvc.EnableIdentity(r.Context(), admindomain.EnableIdentityInput{
		IdentityID: identityID,
		ActorID:    adminActorIDFromContext(r.Context()),
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to enable identity")
		return
	}
	writeJSON(w, http.StatusOK, identity)
}

func (s *server) handleRevokeIdentitySessions(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID := chi.URLParam(r, "identityID")
	if _, err := uuid.Parse(identityID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid identity id")
		return
	}
	if err := s.adminSvc.RevokeIdentitySessions(r.Context(), admindomain.RevokeIdentitySessionsInput{
		IdentityID: identityID,
		ActorID:    adminActorIDFromContext(r.Context()),
	}); err != nil {
		writeError(w, http.StatusBadGateway, "failed to revoke identity sessions")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	limit := 0
	offset := 0
	var err error
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}
	items, err := s.adminSvc.ListAuditLogs(r.Context(), admindomain.ListAuditLogsInput{
		ActorType:  strings.TrimSpace(r.URL.Query().Get("actor_type")),
		ActorID:    strings.TrimSpace(r.URL.Query().Get("actor_id")),
		TargetType: strings.TrimSpace(r.URL.Query().Get("target_type")),
		TargetID:   strings.TrimSpace(r.URL.Query().Get("target_id")),
		EventType:  strings.TrimSpace(r.URL.Query().Get("event_type")),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to list audit logs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token != "" {
			// Apply a tight per-IP limiter to all Bearer token attempts to prevent
			// brute-force guessing of the bootstrap token.
			ip := clientIP(r, s.config.Security.TrustedProxies)
			if !s.authFailureLimiter.Allow(ip) {
				writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
				return
			}
			if s.config.Admin.BootstrapToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.config.Admin.BootstrapToken)) == 1 {
				ctx := context.WithValue(r.Context(), adminActorIDKey, "bootstrap-admin")
				ctx = admindomain.WithRequestMetadata(ctx, admindomain.RequestMetadata{
					IPAddress: ip,
					UserAgent: r.UserAgent(),
					RequestID: middleware.GetReqID(r.Context()),
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if s.authSvc != nil {
			session, err := s.authSvc.CurrentSession(r.Context(), r)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to resolve admin session")
				return
			}
			if session.Authenticated {
				if !adminSessionMFASatisfied(session) {
					writeError(w, http.StatusForbidden, "admin mfa required")
					return
				}
				if emailAllowed(s.config.Admin.AllowedEmails, session.Email) || roleAllowed(s.config.Admin.AllowedRoles, session.Roles) {
					if adminMutationRequiresBootstrapToken(r.Method) {
						writeError(w, http.StatusForbidden, "admin bootstrap token required for mutating requests")
						return
					}
					actorID := session.Email
					if actorID == "" {
						actorID = session.IdentityID
					}
					ctx := context.WithValue(r.Context(), adminActorIDKey, actorID)
					ctx = admindomain.WithRequestMetadata(ctx, admindomain.RequestMetadata{
						IPAddress: clientIP(r, s.config.Security.TrustedProxies),
						UserAgent: r.UserAgent(),
						RequestID: middleware.GetReqID(r.Context()),
					})
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				writeError(w, http.StatusForbidden, "admin access denied")
				return
			}
		}

		writeError(w, http.StatusUnauthorized, "admin authorization required")
	})
}

func adminActorIDFromContext(ctx context.Context) string {
	if actorID, ok := ctx.Value(adminActorIDKey).(string); ok && actorID != "" {
		return actorID
	}
	return "bootstrap-admin"
}

func emailAllowed(allowed []string, email string) bool {
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	if normalizedEmail == "" {
		return false
	}
	for _, candidate := range allowed {
		if normalizedEmail == strings.TrimSpace(strings.ToLower(candidate)) {
			return true
		}
	}
	return false
}

func roleAllowed(allowed []string, roles []string) bool {
	if len(allowed) == 0 || len(roles) == 0 {
		return false
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		normalized := strings.TrimSpace(strings.ToLower(role))
		if normalized == "" {
			continue
		}
		allowedSet[normalized] = struct{}{}
	}
	for _, role := range roles {
		if _, ok := allowedSet[strings.TrimSpace(strings.ToLower(role))]; ok {
			return true
		}
	}
	return false
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrChallengeRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrConsentSessionMismatch):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		writeError(w, http.StatusBadGateway, "auth upstream error")
	}
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrInvalidAppName),
		errors.Is(err, app.ErrInvalidAppSlug),
		errors.Is(err, app.ErrInvalidAppType),
		errors.Is(err, app.ErrInvalidPartyType),
		errors.Is(err, app.ErrInvalidClientName),
		errors.Is(err, app.ErrInvalidClientType),
		errors.Is(err, app.ErrInvalidTokenEndpointAuthMethod),
		errors.Is(err, app.ErrInvalidRedirectURI),
		errors.Is(err, app.ErrRedirectURIsRequired),
		errors.Is(err, app.ErrRedirectURIsNotAllowed),
		errors.Is(err, app.ErrOpenIDScopeRequired),
		errors.Is(err, app.ErrConfidentialClientRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, app.ErrAppNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, app.ErrAppDisabled):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeConsentPage(w http.ResponseWriter, prompt *ConsentPrompt, secureCookies bool) error {
	const tpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Authorize Application</title>
  <style>
    body { margin: 0; font-family: Georgia, serif; background: linear-gradient(160deg, #f7efe6, #dbe8ef); color: #1b2c36; }
    main { max-width: 760px; margin: 48px auto; padding: 32px; background: rgba(255,255,255,0.9); border-radius: 24px; box-shadow: 0 24px 60px rgba(27,44,54,0.14); }
    h1 { margin-top: 0; }
    ul { padding-left: 20px; }
    .meta { color: #49606c; font-size: 14px; }
    .actions { display: flex; gap: 12px; margin-top: 28px; }
    button { border: 0; border-radius: 12px; padding: 14px 18px; font-size: 16px; cursor: pointer; }
    .allow { background: #155e75; color: #fff; }
    .deny { background: #e2e8f0; color: #1b2c36; }
  </style>
</head>
<body>
  <main>
    <h1>Authorize {{ .DisplayName }}</h1>
    <p class="meta">Client ID: {{ .ClientID }}</p>
    <p>This application is requesting access to your shared account.</p>
    <h2>Requested scopes</h2>
    <ul>{{ range .RequestedScope }}<li>{{ . }}</li>{{ end }}</ul>
    {{ if .RequestedAudience }}
    <h2>Requested API audiences</h2>
	    <ul>{{ range .RequestedAudience }}<li>{{ . }}</li>{{ end }}</ul>
	    {{ end }}
	    <form method="post" action="/v1/auth/consent">
	      <input type="hidden" name="consent_challenge" value="{{ .Challenge }}">
	      <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
	      <div class="actions">
	        <button class="allow" type="submit" name="action" value="accept">Allow access</button>
	        <button class="deny" type="submit" name="action" value="deny">Deny</button>
	      </div>
	    </form>
  </main>
</body>
</html>`
	displayName := prompt.ClientName
	if strings.TrimSpace(displayName) == "" {
		displayName = prompt.ClientID
	}
	csrfToken, err := newCSRFToken()
	if err != nil {
		return err
	}
	setConsentCSRFCookie(w, csrfToken, secureCookies)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; style-src 'unsafe-inline'")
	view := struct {
		Challenge         string
		ClientID          string
		DisplayName       string
		CSRFToken         string
		RequestedScope    []string
		RequestedAudience []string
	}{
		Challenge:         prompt.Challenge,
		ClientID:          prompt.ClientID,
		DisplayName:       displayName,
		CSRFToken:         csrfToken,
		RequestedScope:    prompt.RequestedScope,
		RequestedAudience: prompt.RequestedAccessTokenAudience,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	return template.Must(template.New("consent").Parse(tpl)).Execute(w, view)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'self'")
		next.ServeHTTP(w, r)
	})
}

func newCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func validateConsentCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(consentCSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := strings.TrimSpace(r.Form.Get("csrf_token"))
	if formToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) == 1
}

func setConsentCSRFCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     consentCSRFCookieName,
		Value:    token,
		Path:     "/v1/auth/consent",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   600, // 10 minutes — matches Hydra consent flow TTL
	})
}

func clearConsentCSRFCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     consentCSRFCookieName,
		Value:    "",
		Path:     "/v1/auth/consent",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   -1,
	})
}

// validateRedirectURL returns true when raw is an absolute http or https URL
// with a non-empty host. Used to guard against open-redirect payloads that
// could appear in Hydra admin API responses if the upstream is misconfigured.
func validateRedirectURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return scheme == "http" || scheme == "https"
}

func requestIsSecure(r *http.Request, trustedProxies []string) bool {
	if r.TLS != nil {
		return true
	}
	if !requestViaTrustedProxy(r, trustedProxies) {
		return false
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func adminSessionMFASatisfied(session SessionView) bool {
	return strings.EqualFold(strings.TrimSpace(session.AuthenticatorAssuranceLevel), "aal2")
}

func adminMutationRequiresBootstrapToken(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}
