package http

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/account"
	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/profile"
)

var (
	ErrChallengeRequired      = errors.New("challenge is required")
	ErrConsentSessionMismatch = errors.New("active session does not match consent subject")
)

var (
	errUserNotFound     = errors.New("user not found")
	errAmbiguousUserRef = errors.New("multiple users found for identifier")
)

type AuthAction string

const AuthActionRedirect AuthAction = "redirect"

type RouterConfig struct {
	App        config.AppConfig
	Admin      config.AdminConfig
	Ory        config.OryConfig
	Security   config.SecurityConfig
	Limiter    RateLimiter    // optional; nil disables rate limiting
	ProfileSvc ProfileService // optional; nil disables profile endpoints
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
	OshiColor                    string
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
	OshiColor                   string   `json:"oshi_color,omitempty"`
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
	IssueManagementToken(ctx context.Context, appID uuid.UUID, actorID string) (string, error)
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

type AccountService interface {
	ListMembershipsForIdentity(ctx context.Context, identityID string) ([]account.AppMembership, error)
	ListMembershipsForApp(ctx context.Context, appID uuid.UUID) ([]account.AppMembership, error)
	DisconnectIdentityFromApp(ctx context.Context, identityID string, appID uuid.UUID, actorID string) error
	RevokeAppUser(ctx context.Context, appID uuid.UUID, identityID, actorID string) error
	ScheduleDeletion(ctx context.Context, identityID, actorID, reason string) (account.DeletionRequest, error)
	CancelDeletion(ctx context.Context, identityID, actorID string) error
	GetDeletionRequest(ctx context.Context, identityID string) (*account.DeletionRequest, error)
	ResolveAppByToken(ctx context.Context, rawToken string) (app.App, error)
	RegisterIdentityForApp(ctx context.Context, appEntity app.App, input account.RegisterIdentityInput, actorID string) (account.RegisterForAppResult, error)
	GetMembershipForApp(ctx context.Context, appID uuid.UUID, identityID string) (account.AppMembership, error)
}

type ProfileService interface {
	GetProfile(ctx context.Context, identityID string) (profile.Profile, error)
	UpdateProfile(ctx context.Context, identityID string, input profile.UpdateInput) (profile.Profile, error)
}

type themePreferenceService interface {
	UpdateThemePreference(ctx context.Context, r *http.Request, color string) (SessionView, error)
}

type readinessChecker interface {
	Ready(ctx context.Context) error
}

type server struct {
	config             RouterConfig
	adminSvc           AdminService
	authSvc            AuthService
	accountSvc         AccountService
	profileSvc         ProfileService
	readiness          readinessChecker
	authFailureLimiter RateLimiter // tight per-IP limiter for bootstrap token failures
}

type contextKey string

const adminActorIDKey contextKey = "admin_actor_id"
const appActorKey contextKey = "app_actor"
const accountIdentityIDKey contextKey = "account_identity_id"

const consentCSRFCookieName = "idol_auth_consent_csrf"

func NewRouter(cfg RouterConfig, adminSvc AdminService, readiness readinessChecker, authSvc AuthService, accountSvcs ...AccountService) http.Handler {
	var accountSvc AccountService
	if len(accountSvcs) > 0 {
		accountSvc = accountSvcs[0]
	}
	s := &server{
		config:             cfg,
		adminSvc:           adminSvc,
		authSvc:            authSvc,
		accountSvc:         accountSvc,
		profileSvc:         cfg.ProfileSvc,
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
	r.Get("/docs", s.handleDocsIndex)
	r.Get("/docs/*", s.handleDocs)
	r.Route("/account", func(r chi.Router) {
		r.Use(s.accountUIAuth)
		r.Get("/", s.handleAccountCenter)
	})

	r.Route("/v1/auth", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Get("/providers", s.handleProviders)
		r.Get("/session", s.handleSession)
		r.Post("/theme", s.handleThemePreference)
		r.Post("/logout", s.handleLogoutStart)
		r.Get("/logout", s.handleLogout)
		r.Get("/login", s.handleLogin)
		r.Get("/consent", s.handleConsent)
		r.Post("/consent", s.handleConsentSubmit)
	})

	r.Route("/v1/admin", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.adminAuth)
		r.Get("/apps", s.handleListApps)
		r.Post("/apps", s.handleCreateApp)
		r.Post("/apps/{appID}/management-token", s.handleIssueManagementToken)
		r.Get("/apps/{appID}/clients", s.handleListOIDCClients)
		r.Post("/apps/{appID}/clients", s.handleCreateOIDCClient)
		r.Get("/users", s.handleSearchIdentities)
		r.Patch("/users/{userRef}", s.handlePatchUser)
		r.Post("/users/{userRef}/revoke-sessions", s.handleRevokeIdentitySessions)
		r.Delete("/users/{userRef}", s.handleDeleteIdentity)
		r.Get("/audit-logs", s.handleListAuditLogs)
	})

	r.Route("/v1/account", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.accountAuth)
		r.Get("/", s.handleAccountOverview)
		r.Delete("/apps/{appID}", s.handleDisconnectAccountApp)
		r.Get("/deletion", s.handleGetDeletionRequest)
		r.Post("/deletion", s.handleScheduleDeletion)
		r.Delete("/deletion", s.handleCancelDeletion)
		r.Get("/profile", s.handleGetProfile)
		r.Patch("/profile", s.handlePatchProfile)
	})

	r.Route("/v1/apps/self", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.appTokenAuth)
		r.Get("/users", s.handleListAppUsers)
		r.Post("/users", s.handleRegisterAppUser)
		r.Delete("/users/{identityID}", s.handleRevokeAppUser)
		r.Get("/users/{identityID}/profile", s.handleGetAppUserProfile)
	})

	r.Route("/admin-ui", func(r chi.Router) {
		r.Use(s.adminUIAuth)
		r.Get("/", s.handleAdminUIOverview)
		r.Get("/apps", s.handleAdminUIApps)
		r.Get("/users", s.handleAdminUIUsers)
		r.Get("/audit-logs", s.handleAdminUIAuditLogs)
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
	logoutURL := strings.TrimRight(s.config.Ory.HydraBrowserURL, "/") + "/oauth2/sessions/logout"
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]string{"logout_url": logoutURL})
		return
	}
	http.Redirect(w, r, logoutURL, http.StatusSeeOther)
}

// wantsJSON returns true when the caller expects a JSON response.
// Defaults to true when Accept is absent or wildcard to preserve
// backwards compatibility for programmatic clients that do not set Accept.
func wantsJSON(r *http.Request) bool {
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		return true
	}
	accept := r.Header.Get("Accept")
	if accept == "" || accept == "*/*" {
		return true
	}
	return strings.Contains(accept, "application/json")
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
		// Top-level shorthand: provide redirect_uris here to create an OIDC
		// client inline without an explicit "client" block. client_type and
		// name are inferred automatically from the app. Takes effect only when
		// the "client" field is absent.
		RedirectURIs           []string `json:"redirect_uris"`
		PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris"`
		Scopes                 []string `json:"scopes"`
		Client                 *struct {
			Name                    string   `json:"name"`
			ClientType              string   `json:"client_type"`
			TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
			RedirectURIs            []string `json:"redirect_uris"`
			PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris"`
			Scopes                  []string `json:"scopes"`
		} `json:"client"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	actorID := adminActorIDFromContext(r.Context())
	created, err := s.adminSvc.CreateApp(r.Context(), app.CreateAppInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Type:        app.AppType(req.Type),
		PartyType:   app.PartyType(req.PartyType),
		Description: req.Description,
		ActorID:     actorID,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// Determine whether to create an inline OIDC client.
	// Explicit "client" block takes precedence over top-level shorthand.
	var clientInput *app.CreateOIDCClientInput
	if req.Client != nil {
		clientInput = &app.CreateOIDCClientInput{
			Name:                    req.Client.Name,
			ClientType:              app.ClientType(req.Client.ClientType),
			TokenEndpointAuthMethod: req.Client.TokenEndpointAuthMethod,
			RedirectURIs:            req.Client.RedirectURIs,
			PostLogoutRedirectURIs:  req.Client.PostLogoutRedirectURIs,
			Scopes:                  req.Client.Scopes,
			ActorID:                 actorID,
		}
	} else if len(req.RedirectURIs) > 0 {
		clientInput = &app.CreateOIDCClientInput{
			RedirectURIs:           req.RedirectURIs,
			PostLogoutRedirectURIs: req.PostLogoutRedirectURIs,
			Scopes:                 req.Scopes,
			ActorID:                actorID,
		}
	}

	if clientInput == nil {
		w.Header().Set("Location", "/v1/admin/apps/"+created.ID.String())
		token, err := s.adminSvc.IssueManagementToken(r.Context(), created.ID, actorID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"app":              created,
			"management_token": token,
		})
		return
	}

	reg, err := s.adminSvc.CreateOIDCClient(r.Context(), created.ID, *clientInput)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	token, err := s.adminSvc.IssueManagementToken(r.Context(), created.ID, actorID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("Location", "/v1/admin/apps/"+created.ID.String())
	writeJSON(w, http.StatusCreated, map[string]any{
		"app":              created,
		"client":           reg.Client,
		"client_secret":    reg.ClientSecret,
		"management_token": token,
	})
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

func (s *server) handleIssueManagementToken(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}
	token, err := s.adminSvc.IssueManagementToken(r.Context(), appID, adminActorIDFromContext(r.Context()))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":           appID.String(),
		"management_token": token,
	})
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
	if err := decodeJSON(w, r, &req); err != nil {
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

	w.Header().Set("Location", "/v1/admin/apps/"+appID.String()+"/clients/"+created.Client.ID.String())
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

func (s *server) handleDeleteIdentity(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID, err := s.resolveUserRef(r.Context(), chi.URLParam(r, "userRef"))
	if err != nil {
		writeUserRefError(w, err)
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

func (s *server) handleRevokeIdentitySessions(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID, err := s.resolveUserRef(r.Context(), chi.URLParam(r, "userRef"))
	if err != nil {
		writeUserRefError(w, err)
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

func (s *server) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	if s.adminSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "admin service unavailable")
		return
	}
	identityID, err := s.resolveUserRef(r.Context(), chi.URLParam(r, "userRef"))
	if err != nil {
		writeUserRefError(w, err)
		return
	}

	var req struct {
		State string    `json:"state"`
		Roles *[]string `json:"roles"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if req.State == "" && req.Roles == nil {
		writeError(w, http.StatusBadRequest, "at least one of 'state' or 'roles' is required")
		return
	}

	actorID := adminActorIDFromContext(r.Context())

	var identity admindomain.Identity
	var hasIdentity bool

	if req.State != "" {
		switch req.State {
		case string(admindomain.IdentityStateActive):
			identity, err = s.adminSvc.EnableIdentity(r.Context(), admindomain.EnableIdentityInput{
				IdentityID: identityID,
				ActorID:    actorID,
			})
			if err != nil {
				writeError(w, http.StatusBadGateway, "failed to enable identity")
				return
			}
			hasIdentity = true
		case string(admindomain.IdentityStateInactive):
			identity, err = s.adminSvc.DisableIdentity(r.Context(), admindomain.DisableIdentityInput{
				IdentityID: identityID,
				ActorID:    actorID,
			})
			if err != nil {
				writeError(w, http.StatusBadGateway, "failed to disable identity")
				return
			}
			hasIdentity = true
		default:
			writeError(w, http.StatusBadRequest, "state must be 'active' or 'inactive'")
			return
		}
	}

	if req.Roles != nil {
		roles, err := s.adminSvc.SetIdentityRoles(r.Context(), admindomain.SetIdentityRolesInput{
			IdentityID: identityID,
			Roles:      *req.Roles,
			ActorID:    actorID,
		})
		if err != nil {
			writeError(w, http.StatusBadGateway, "failed to update identity roles")
			return
		}
		if hasIdentity {
			identity.Roles = roles
			writeJSON(w, http.StatusOK, identity)
		} else {
			writeJSON(w, http.StatusOK, map[string]any{
				"identity_id": identityID,
				"roles":       roles,
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, identity)
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

func (s *server) handleAccountOverview(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	memberships, err := s.accountSvc.ListMembershipsForIdentity(r.Context(), session.IdentityID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to load account memberships")
		return
	}
	deletionRequest, err := s.accountSvc.GetDeletionRequest(r.Context(), session.IdentityID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"identity_id":      session.IdentityID,
		"email":            session.Email,
		"memberships":      memberships,
		"deletion_request": deletionRequest,
		"authenticated":    true,
		"subject":          session.Subject,
	})
}

func (s *server) handleDisconnectAccountApp(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}
	if err := s.accountSvc.DisconnectIdentityFromApp(r.Context(), session.IdentityID, appID, session.IdentityID); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleGetDeletionRequest(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	request, err := s.accountSvc.GetDeletionRequest(r.Context(), session.IdentityID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"request": request})
}

func (s *server) handleScheduleDeletion(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	request, err := s.accountSvc.ScheduleDeletion(r.Context(), session.IdentityID, session.IdentityID, req.Reason)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"request": request})
}

func (s *server) handleCancelDeletion(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	session, ok := accountSessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err := s.accountSvc.CancelDeletion(r.Context(), session.IdentityID, session.IdentityID); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleListAppUsers(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	items, err := s.accountSvc.ListMembershipsForApp(r.Context(), appActor.ID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app":   appActor,
		"items": items,
	})
}

func (s *server) handleRevokeAppUser(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	identityID := strings.TrimSpace(chi.URLParam(r, "identityID"))
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "identity id is required")
		return
	}
	if err := s.accountSvc.RevokeAppUser(r.Context(), appActor.ID, identityID, appActor.ID.String()); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleRegisterAppUser(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "account service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	result, err := s.accountSvc.RegisterIdentityForApp(r.Context(), appActor, account.RegisterIdentityInput{
		Email:       req.Email,
		DisplayName: req.DisplayName,
	}, appActor.ID.String())
	if err != nil {
		if errors.Is(err, account.ErrSharedAccountAlreadyExists) {
			writeError(w, http.StatusConflict, "shared account already exists; use the shared account sign-in flow")
			return
		}
		writeError(w, http.StatusBadGateway, "failed to register user")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *server) handleGetAppUserProfile(w http.ResponseWriter, r *http.Request) {
	if s.accountSvc == nil || s.profileSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	appActor, ok := appActorFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "app authorization required")
		return
	}
	identityID := strings.TrimSpace(chi.URLParam(r, "identityID"))
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "identity id is required")
		return
	}
	if _, err := s.accountSvc.GetMembershipForApp(r.Context(), appActor.ID, identityID); err != nil {
		if errors.Is(err, account.ErrMembershipNotFound) {
			writeError(w, http.StatusNotFound, "user not found in this app")
			return
		}
		writeError(w, http.StatusBadGateway, "failed to verify membership")
		return
	}
	p, err := s.profileSvc.GetProfile(r.Context(), identityID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to load profile")
		return
	}
	writeJSON(w, http.StatusOK, p.PublicView())
}

func (s *server) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token != "" {
			ip := clientIP(r, s.config.Security.TrustedProxies)
			if s.config.Admin.BootstrapToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.config.Admin.BootstrapToken)) == 1 {
				// Valid bootstrap token — do not consume the failure rate-limit budget.
				ctx := context.WithValue(r.Context(), adminActorIDKey, "bootstrap-admin")
				ctx = admindomain.WithRequestMetadata(ctx, admindomain.RequestMetadata{
					IPAddress: ip,
					UserAgent: r.UserAgent(),
					RequestID: middleware.GetReqID(r.Context()),
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Wrong or absent bootstrap token — consume from the per-IP failure budget
			// to prevent brute-force guessing.
			if !s.authFailureLimiter.Allow(ip) {
				writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
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

func (s *server) accountAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil {
			writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		session, err := s.authSvc.CurrentSession(r.Context(), r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to resolve account session")
			return
		}
		if !session.Authenticated || strings.TrimSpace(session.IdentityID) == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), accountIdentityIDKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) appTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.accountSvc == nil {
			writeError(w, http.StatusServiceUnavailable, "account service unavailable")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeError(w, http.StatusUnauthorized, "app authorization required")
			return
		}
		appActor, err := s.accountSvc.ResolveAppByToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, app.ErrAppNotFound) {
				writeError(w, http.StatusUnauthorized, "app authorization required")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to resolve app authorization")
			return
		}
		ctx := context.WithValue(r.Context(), appActorKey, appActor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func adminActorIDFromContext(ctx context.Context) string {
	if actorID, ok := ctx.Value(adminActorIDKey).(string); ok && actorID != "" {
		return actorID
	}
	return "bootstrap-admin"
}

func accountSessionFromContext(ctx context.Context) (SessionView, bool) {
	session, ok := ctx.Value(accountIdentityIDKey).(SessionView)
	return session, ok
}

func appActorFromContext(ctx context.Context) (app.App, bool) {
	appActor, ok := ctx.Value(appActorKey).(app.App)
	return appActor, ok
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

const maxRequestBodyBytes = 1 << 20 // 1 MiB

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
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

func writeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, account.ErrDeletionRequestNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, app.ErrAppNotFound):
		writeError(w, http.StatusNotFound, err.Error())
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

func writeConsentPage(w http.ResponseWriter, prompt *ConsentPrompt, secureCookies bool, cookieDomain string) error {
	const tpl = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>アプリ連携の確認</title>
  <style>
    :root {
      --oshi: #b2b2ff;
      --oshi-deep: #4c4cc6;
      --oshi-soft: rgba(178,178,255,0.18);
      --oshi-line: rgba(178,178,255,0.42);
    }
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background:
        radial-gradient(circle at 16% 20%, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0) 24%),
        radial-gradient(circle at 82% 10%, var(--oshi-soft) 0%, rgba(255,255,255,0) 32%),
        linear-gradient(160deg, #fff8fb 0%, #f4f6ff 48%, #edfaff 100%);
      color: #1d2040;
      font-family: "Avenir Next", "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
      padding: 28px 18px 88px;
    }
    main {
      max-width: 760px;
      margin: 0 auto;
      padding: 28px;
      background: rgba(255,255,255,0.82);
      border-radius: 34px;
      border: 1px solid rgba(255,255,255,0.84);
      box-shadow: 0 26px 70px rgba(59,61,109,0.13);
      backdrop-filter: blur(24px);
      position: relative;
      overflow: hidden;
    }
    main::before {
      content: "";
      position: absolute;
      inset: -10% auto auto -10%;
      width: 260px;
      height: 260px;
      border-radius: 999px;
      background: radial-gradient(circle, var(--oshi-soft) 0%, rgba(255,255,255,0) 70%);
      pointer-events: none;
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 7px 14px;
      border-radius: 999px;
      background: rgba(255,255,255,0.72);
      border: 1px solid var(--oshi-line);
      color: var(--oshi-deep);
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      margin-bottom: 18px;
      position: relative;
      z-index: 1;
    }
    .app-row {
      display: flex;
      align-items: center;
      gap: 14px;
      margin-bottom: 18px;
      position: relative;
      z-index: 1;
    }
    .app-icon {
      width: 54px;
      height: 54px;
      border-radius: 18px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(180deg, rgba(255,255,255,0.96), var(--oshi-soft));
      border: 1px solid var(--oshi-line);
      color: var(--oshi-deep);
      font-size: 22px;
    }
    .app-name {
      font-size: 28px;
      font-weight: 800;
      letter-spacing: -0.05em;
      line-height: 1.02;
    }
    .app-id {
      color: #8a8faa;
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      margin-top: 4px;
    }
    p {
      margin: 0 0 20px;
      color: #6f7394;
      line-height: 1.8;
      position: relative;
      z-index: 1;
    }
    .section-label {
      margin: 0 0 10px;
      color: var(--oshi-deep);
      font-size: 12px;
      font-weight: 700;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      position: relative;
      z-index: 1;
    }
    ul {
      list-style: none;
      padding: 0;
      margin: 0 0 20px;
      display: grid;
      gap: 8px;
      position: relative;
      z-index: 1;
    }
    li {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 12px 14px;
      border-radius: 18px;
      background: rgba(255,255,255,0.72);
      border: 1px solid rgba(29,32,64,0.08);
      color: #4f5477;
      font-size: 14px;
    }
    li::before {
      content: "◉";
      color: var(--oshi-deep);
      font-size: 9px;
    }
    .divider {
      border: none;
      border-top: 1px solid rgba(29,32,64,0.08);
      margin: 6px 0 24px;
      position: relative;
      z-index: 1;
    }
    .actions {
      display: flex;
      gap: 12px;
      position: relative;
      z-index: 1;
    }
    button {
      flex: 1;
      border-radius: 18px;
      padding: 15px 18px;
      font-size: 15px;
      font-weight: 700;
      cursor: pointer;
      transition: transform 0.14s ease, box-shadow 0.14s ease, opacity 0.14s ease;
    }
    button:active { transform: scale(0.99); }
    .allow {
      border: none;
      background: linear-gradient(180deg, var(--oshi), rgba(255,255,255,0.68));
      color: #1d2040;
      box-shadow: 0 18px 36px rgba(59,61,109,0.14);
    }
    .allow:hover { opacity: 0.86; }
    .deny {
      background: rgba(255,255,255,0.72);
      color: #5d6382;
      border: 1px solid rgba(29,32,64,0.08);
    }
    .deny:hover { box-shadow: 0 14px 28px rgba(59,61,109,0.1); }
    #oshi-picker { position: fixed; right: 18px; bottom: 18px; z-index: 20; }
    #oshi-toggle {
      width: 58px;
      height: 58px;
      border-radius: 20px;
      border: 1px solid rgba(255,255,255,0.84);
      background: linear-gradient(180deg, rgba(255,255,255,0.96), var(--oshi-soft));
      color: var(--oshi-deep);
      font-size: 24px;
      cursor: pointer;
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      backdrop-filter: blur(24px);
    }
    #oshi-swatches {
      display: none;
      grid-template-columns: repeat(4, 1fr);
      gap: 10px;
      width: 188px;
      margin-bottom: 12px;
      padding: 14px;
      border-radius: 22px;
      background: rgba(255,255,255,0.86);
      border: 1px solid rgba(255,255,255,0.84);
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      backdrop-filter: blur(24px);
    }
    .swatch {
      width: 100%;
      aspect-ratio: 1;
      border-radius: 999px;
      border: 2px solid transparent;
      cursor: pointer;
    }
    .swatch.active { border-color: #1d2040; }
    @media (max-width: 640px) {
      main { padding: 22px; border-radius: 26px; }
      .app-name { font-size: 24px; }
      .actions { flex-direction: column; }
      #oshi-toggle { width: 52px; height: 52px; border-radius: 18px; }
      #oshi-swatches { width: 168px; }
    }
  </style>
  <script>
    var OSHI=['#ffb2b2','#ffb2d8','#ffb2ff','#d8b2ff','#b2b2ff','#b2d8ff','#b2ffff','#b2ffd8','#b2ffb2','#d8ffb2','#ffffb2','#ffd8b2'];
    function normalizeOshi(raw){
      raw=(raw||'').trim().toLowerCase();
      return OSHI.indexOf(raw)>=0?raw:'';
    }
    function oshiRgb(hex){return[parseInt(hex.slice(1,3),16),parseInt(hex.slice(3,5),16),parseInt(hex.slice(5,7),16)];}
    function oshiHex(r,g,b){return'#'+[r,g,b].map(function(v){return Math.min(255,Math.max(0,v)).toString(16).padStart(2,'0');}).join('');}
    function applyOshi(color){
      var c=oshiRgb(color), root=document.documentElement;
      root.style.setProperty('--oshi', color);
      root.style.setProperty('--oshi-deep', oshiHex(c[0]-90, c[1]-90, c[2]-40));
      root.style.setProperty('--oshi-soft', 'rgba('+c[0]+','+c[1]+','+c[2]+',0.18)');
      root.style.setProperty('--oshi-line', 'rgba('+c[0]+','+c[1]+','+c[2]+',0.42)');
    }
    function persistOshi(color){
      fetch('/v1/auth/theme',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        credentials:'same-origin',
        body:JSON.stringify({oshi_color:color})
      }).catch(function(){});
    }
    var _oshi=normalizeOshi({{ printf "%q" .OshiColor }})||OSHI[4];
    applyOshi(_oshi);
  </script>
</head>
<body>
  <main>
    <div class="eyebrow">✦ OAuth Consent</div>
    <div class="app-row">
      <div class="app-icon">◈</div>
      <div>
        <div class="app-name">{{ .DisplayName }}</div>
        <div class="app-id">{{ .ClientID }}</div>
      </div>
    </div>
    <p>このアプリが共有アカウントへのアクセスを求めています。スコープと利用先を確認して、連携を許可するか選んでください。</p>
    <div class="section-label">Requested scopes</div>
    <ul>{{ range .RequestedScope }}<li>{{ . }}</li>{{ end }}</ul>
    {{ if .RequestedAudience }}
    <div class="section-label">Requested audiences</div>
    <ul>{{ range .RequestedAudience }}<li>{{ . }}</li>{{ end }}</ul>
    {{ end }}
    <hr class="divider">
    <form method="post" action="/v1/auth/consent">
      <input type="hidden" name="consent_challenge" value="{{ .Challenge }}">
      <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
      <div class="actions">
        <button class="allow" type="submit" name="action" value="accept">アクセスを許可</button>
        <button class="deny" type="submit" name="action" value="deny">キャンセル</button>
      </div>
    </form>
  </main>
  <div id="oshi-picker">
    <div id="oshi-swatches" aria-label="推しメンカラーパレット"></div>
    <button id="oshi-toggle" type="button" title="推しメンカラー">✦</button>
  </div>
  <script>
    (function(){
      var sw=document.getElementById('oshi-swatches');
      var toggle=document.getElementById('oshi-toggle');
      var current=normalizeOshi({{ printf "%q" .OshiColor }})||OSHI[4];
      OSHI.forEach(function(color){
        var btn=document.createElement('button');
        btn.type='button';
        btn.className='swatch'+(color===current?' active':'');
        btn.style.background=color;
        btn.title='推しメンカラー '+(OSHI.indexOf(color)+1);
        btn.addEventListener('click', function(){
          applyOshi(color);
          persistOshi(color);
          document.querySelectorAll('.swatch').forEach(function(node){
            node.classList.toggle('active', node===btn);
          });
        });
        sw.appendChild(btn);
      });
      toggle.addEventListener('click', function(){
        sw.style.display = sw.style.display === 'grid' ? 'none' : 'grid';
      });
    })();
  </script>
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
	setConsentCSRFCookie(w, csrfToken, secureCookies, cookieDomain)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'")
	view := struct {
		Challenge         string
		ClientID          string
		DisplayName       string
		OshiColor         string
		CSRFToken         string
		RequestedScope    []string
		RequestedAudience []string
	}{
		Challenge:         prompt.Challenge,
		ClientID:          prompt.ClientID,
		DisplayName:       displayName,
		OshiColor:         prompt.OshiColor,
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
		w.Header().Set("Cache-Control", "no-store")
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

func setConsentCSRFCookie(w http.ResponseWriter, token string, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     consentCSRFCookieName,
		Value:    token,
		Path:     "/v1/auth/consent",
		Domain:   domain,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   600, // 10 minutes — matches Hydra consent flow TTL
	})
}

func clearConsentCSRFCookie(w http.ResponseWriter, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     consentCSRFCookieName,
		Value:    "",
		Path:     "/v1/auth/consent",
		Domain:   domain,
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

// resolveUserRef accepts either a UUID or an email/identifier string.
// UUIDs are passed through directly; other values trigger a Kratos identity search.
func (s *server) resolveUserRef(ctx context.Context, ref string) (string, error) {
	if _, err := uuid.Parse(ref); err == nil {
		return ref, nil
	}
	identities, err := s.adminSvc.SearchIdentities(ctx, admindomain.SearchIdentitiesInput{
		CredentialsIdentifier: ref,
	})
	if err != nil {
		return "", fmt.Errorf("search identity: %w", err)
	}
	switch len(identities) {
	case 0:
		return "", errUserNotFound
	case 1:
		return identities[0].ID, nil
	default:
		return "", errAmbiguousUserRef
	}
}

func writeUserRefError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, errAmbiguousUserRef):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadGateway, "failed to resolve user")
	}
}
