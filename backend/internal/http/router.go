package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/config"
	"github.com/kuro48/idol-auth/internal/domain/account"
	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
	"github.com/kuro48/idol-auth/internal/domain/app"
	"github.com/kuro48/idol-auth/internal/domain/loginhistory"
	"github.com/kuro48/idol-auth/internal/domain/profile"
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
	App                config.AppConfig
	Admin              config.AdminConfig
	Ory                config.OryConfig
	Security           config.SecurityConfig
	Limiter            RateLimiter            // optional; nil disables rate limiting
	AccountSvc         AccountService         // optional; nil disables account endpoints
	ProfileSvc         ProfileService         // optional; nil disables profile endpoints
	PublicSvc          PublicAuthService      // optional; nil disables /v1/public endpoints
	AdminAppRegSvc     AdminAppRegService     // optional; nil disables /v1/admin/app-requests endpoints
	DeveloperAppRegSvc DeveloperAppRegService // optional; nil disables /developer/app-requests endpoints
	WebhookRepo        app.WebhookRepository  // optional; nil disables PATCH /v1/apps/self/webhook
	SessionMgr         SessionManager         // optional; nil disables session list/revoke endpoints
	LoginHistorySvc    LoginHistoryService    // optional; nil disables /v1/account/login-history
	EmailVerifSvc      EmailVerifChecker      // optional; nil disables /v1/account/email-status
	PasswordChangeSvc  PasswordChanger        // optional; nil disables /v1/account/password-change
	SocialProviderSvc  SocialProviderLister   // optional; nil disables /v1/account/social-providers
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
	SessionID                   string   `json:"session_id,omitempty"`
	IdentityID                  string   `json:"identity_id,omitempty"`
	Email                       string   `json:"email,omitempty"`
	Phone                       string   `json:"phone,omitempty"`
	DisplayName                 string   `json:"display_name,omitempty"`
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
	SetAppPartyType(ctx context.Context, appID uuid.UUID, partyType app.PartyType, actorID string) (app.App, error)
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
	LogoutSession(ctx context.Context, r *http.Request) error
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
	ExportAccountData(ctx context.Context, identityID string) (account.AccountExport, error)
	GetAppStats(ctx context.Context, appID uuid.UUID) (account.AppMembershipStats, error)
}

type ProfileService interface {
	GetProfile(ctx context.Context, identityID string) (profile.Profile, error)
	UpdateProfile(ctx context.Context, identityID string, input profile.UpdateInput) (profile.Profile, error)
}

// SessionManager lists and revokes individual Kratos sessions for an identity.
type SessionManager interface {
	ListSessionsForIdentity(ctx context.Context, identityID string) ([]account.SessionInfo, error)
	RevokeSession(ctx context.Context, sessionID string) error
}

// LoginHistoryService returns observed-login records for the current identity.
type LoginHistoryService interface {
	List(ctx context.Context, identityID string, limit int) ([]loginhistory.Event, error)
}

// EmailVerifChecker returns the email address and verification status for an identity.
type EmailVerifChecker interface {
	GetEmailVerificationStatus(ctx context.Context, identityID string) (string, bool, error)
}

// PasswordChanger changes the Kratos password for the current session's identity.
type PasswordChanger interface {
	ChangePassword(ctx context.Context, r *http.Request, newPassword string) error
}

// SocialProviderLister lists the OIDC social providers linked to an identity.
type SocialProviderLister interface {
	GetSocialProviders(ctx context.Context, identityID string) ([]account.SocialProvider, error)
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
	publicSvc          PublicAuthService
	adminAppRegSvc     AdminAppRegService
	developerAppRegSvc DeveloperAppRegService
	webhookRepo        app.WebhookRepository
	sessionMgr         SessionManager
	loginHistorySvc    LoginHistoryService
	emailVerifSvc      EmailVerifChecker
	passwordChangeSvc  PasswordChanger
	socialProviderSvc  SocialProviderLister
	readiness          readinessChecker
	authFailureLimiter RateLimiter // tight per-IP limiter for bootstrap token failures
	appTokenLimiter    RateLimiter // tight per-IP limiter for app management token failures
	credentialLimiter  RateLimiter // strict per-IP limiter for /login and /register
	themeLimiter       RateLimiter // moderate per-IP limiter for /v1/auth/theme
}

func NewRouter(cfg RouterConfig, adminSvc AdminService, readiness readinessChecker, authSvc AuthService) http.Handler {
	s := &server{
		config:             cfg,
		adminSvc:           adminSvc,
		authSvc:            authSvc,
		accountSvc:         cfg.AccountSvc,
		profileSvc:         cfg.ProfileSvc,
		publicSvc:          cfg.PublicSvc,
		adminAppRegSvc:     cfg.AdminAppRegSvc,
		developerAppRegSvc: cfg.DeveloperAppRegSvc,
		webhookRepo:        cfg.WebhookRepo,
		sessionMgr:         cfg.SessionMgr,
		loginHistorySvc:    cfg.LoginHistorySvc,
		emailVerifSvc:      cfg.EmailVerifSvc,
		passwordChangeSvc:  cfg.PasswordChangeSvc,
		socialProviderSvc:  cfg.SocialProviderSvc,
		readiness:          readiness,
		authFailureLimiter: NewInMemoryRateLimiter(5, 5*time.Minute),
		appTokenLimiter:    NewInMemoryRateLimiter(5, 5*time.Minute),
		credentialLimiter:  NewInMemoryRateLimiter(5, time.Minute),
		themeLimiter:       NewInMemoryRateLimiter(10, time.Minute),
	}

	r := chi.NewRouter()
	r.Use(securityHeaders)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(httpMetricsMiddleware)

	r.Handle("/metrics", metricsHandler())
	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)
	// /docs and all non-API paths are served by the React SPA (Caddy container).
	// Traefik routes those requests to the frontend at lower priority than /v1/*.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	r.Get("/login", s.handleLoginPage)
	r.Get("/register", s.handleRegistrationPage)
	r.Get("/uploads/avatars/{file}", s.handleAvatarAsset)
	r.Route("/legal", func(r chi.Router) {
		r.Get("/terms", s.handleLegalTerms)
		r.Get("/privacy", s.handleLegalPrivacy)
		r.Get("/contact", s.handleLegalContact)
		r.Get("/incident", s.handleLegalIncident)
	})
	r.Get("/account/restore", s.handleRestoreAccount)
	r.With(s.accountSessionCSRFMiddleware).Post("/account/restore", s.handleSubmitRestore)
	r.Route("/v1/auth", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Get("/providers", s.handleProviders)
		r.Get("/session", s.handleSession)
		r.Get("/csrf", s.handleGetCSRFToken)
		r.With(rateLimitMiddleware(s.themeLimiter, s.config.Security.TrustedProxies), s.accountSessionCSRFMiddleware).Post("/theme", s.handleThemePreference)
		r.Post("/logout", s.handleLogoutStart)
		r.Get("/logout", s.handleLogout)
		r.Get("/login", s.handleLogin)
		r.Get("/consent", s.handleConsent)
		r.Post("/consent", s.handleConsentSubmit)
	})

	r.Route("/v1/admin", func(r chi.Router) {
		if len(s.config.Security.CORSAllowedOrigins) > 0 {
			r.Use(corsMiddleware(s.config.Security.CORSAllowedOrigins))
		}
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.adminAuth)
		r.Use(s.adminSessionCSRFMiddleware)
		r.Get("/apps", s.handleListApps)
		r.Post("/apps", s.handleCreateApp)
		r.Post("/apps/{appID}/management-token", s.handleIssueManagementToken)
		r.Get("/apps/{appID}/clients", s.handleListOIDCClients)
		r.Post("/apps/{appID}/clients", s.handleCreateOIDCClient)
		r.Patch("/apps/{appID}/party-type", s.handleSetAppPartyType)
		r.Get("/users", s.handleSearchIdentities)
		r.Patch("/users/{userRef}", s.handlePatchUser)
		r.Patch("/users/{userRef}/profile-awards", s.handlePatchProfileAwards)
		r.Post("/users/{userRef}/revoke-sessions", s.handleRevokeIdentitySessions)
		r.Delete("/users/{userRef}", s.handleDeleteIdentity)
		r.Get("/audit-logs", s.handleListAuditLogs)
		r.Get("/app-requests", s.handleListAdminAppRequests)
		r.Get("/app-requests/{id}", s.handleGetAdminAppRequest)
		r.Post("/app-requests/{id}/approve", s.handleApproveAppRequest)
		r.Post("/app-requests/{id}/reject", s.handleRejectAppRequest)
		r.Post("/app-requests/{id}/request-changes", s.handleRequestChangesAppRequest)
	})

	r.Route("/v1/account", func(r chi.Router) {
		if len(s.config.Security.CORSAllowedOrigins) > 0 {
			r.Use(corsMiddleware(s.config.Security.CORSAllowedOrigins))
		}
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.accountAuth)
		r.Use(s.accountSessionCSRFMiddleware)
		r.Get("/", s.handleAccountOverview)
		r.Delete("/apps/{appID}", s.handleDisconnectAccountApp)
		r.Get("/deletion", s.handleGetDeletionRequest)
		r.Post("/deletion", s.handleScheduleDeletion)
		r.Delete("/deletion", s.handleCancelDeletion)
		r.Get("/profile", s.handleGetProfile)
		r.Patch("/profile", s.handlePatchProfile)
		r.Post("/profile/avatar", s.handleUploadAvatar)
		r.Patch("/profile/visibility", s.handlePatchProfileVisibility)
		r.Get("/data-preferences", s.handleGetDataPreferences)
		r.Patch("/data-preferences", s.handlePatchDataPreferences)
		r.Patch("/recovery-contacts", s.handlePatchRecoveryContacts)
		r.Get("/export", s.handleExportAccount)
		r.Post("/developer-registration", s.handleDeveloperRegistration)
		if s.sessionMgr != nil {
			r.Get("/sessions", s.handleListSessions)
			r.Delete("/sessions/{sessionId}", s.handleRevokeSession)
		}
		if s.loginHistorySvc != nil {
			r.Get("/login-history", s.handleListLoginHistory)
		}
		if s.emailVerifSvc != nil {
			r.Get("/email-status", s.handleGetEmailStatus)
		}
		if s.passwordChangeSvc != nil {
			r.Post("/password-change", s.handlePasswordChange)
		}
		if s.socialProviderSvc != nil {
			r.Get("/social-providers", s.handleGetSocialProviders)
		}
	})

	r.Route("/v1/users", func(r chi.Router) {
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.accountAuth)
		r.Get("/{user_id}/profile", s.handleGetPublicUserProfile)
	})

	r.Route("/v1/developer/apps", func(r chi.Router) {
		if len(s.config.Security.CORSAllowedOrigins) > 0 {
			r.Use(corsMiddleware(s.config.Security.CORSAllowedOrigins))
		}
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.developerAuth)
		r.Use(s.accountSessionCSRFMiddleware)
		r.Post("/", s.handleDeveloperCreateApp)
	})

	r.Route("/v1/developer/app-requests", func(r chi.Router) {
		if len(s.config.Security.CORSAllowedOrigins) > 0 {
			r.Use(corsMiddleware(s.config.Security.CORSAllowedOrigins))
		}
		if s.config.Limiter != nil {
			r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
		}
		r.Use(s.developerAuth)
		r.Use(s.accountSessionCSRFMiddleware)
		r.Get("/", s.handleListMyAppRequests)
		r.Post("/", s.handleSubmitAppRequest)
		r.Get("/{id}", s.handleGetMyAppRequest)
		r.Post("/{id}/withdraw", s.handleWithdrawMyAppRequest)
		r.Post("/{id}/resubmit", s.handleResubmitMyAppRequest)
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
		if s.webhookRepo != nil {
			r.Patch("/webhook", s.handlePatchAppWebhook)
		}
		r.Get("/stats", s.handleGetAppStats)
	})

	if s.publicSvc != nil {
		r.Route("/v1/public", func(r chi.Router) {
			if len(s.config.Security.CORSAllowedOrigins) > 0 {
				r.Use(corsMiddleware(s.config.Security.CORSAllowedOrigins))
			}
			if s.config.Limiter != nil {
				r.Use(rateLimitMiddleware(s.config.Limiter, s.config.Security.TrustedProxies))
			}
			r.Route("/browser", func(r chi.Router) {
				r.Get("/login", s.handlePublicBrowserLogin)
				r.Get("/registration", s.handlePublicBrowserRegistration)
				r.Get("/logout", s.handlePublicBrowserLogout)
			})
			r.Route("/api", func(r chi.Router) {
				r.Post("/token", s.handlePublicToken)
				r.Post("/token/revoke", s.handlePublicRevoke)
				r.Post("/token/introspect", s.handlePublicIntrospect)
				r.Get("/session", s.handlePublicSession)
				r.Group(func(r chi.Router) {
					r.Use(rateLimitMiddleware(s.credentialLimiter, s.config.Security.TrustedProxies))
					r.Post("/register", s.handlePublicRegister)
					r.Post("/login", s.handlePublicLogin)
				})
			})
		})
	}

	return r
}
