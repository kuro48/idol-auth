package http

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
	"github.com/kuro48/idol-auth/internal/domain/app"
)

type contextKey string

const adminActorIDKey contextKey = "admin_actor_id"
const adminAuthMethodKey contextKey = "admin_auth_method"
const appActorKey contextKey = "app_actor"
const accountIdentityIDKey contextKey = "account_identity_id"

type adminAuthMethod string

const (
	adminAuthMethodBootstrap  adminAuthMethod = "bootstrap"
	adminAuthMethodCloudflare adminAuthMethod = "cloudflare"
)

func (s *server) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, s.config.Security.TrustedProxies)

		// 1. Bootstrap token (emergency / local dev access).
		if token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")); token != "" {
			if s.config.Admin.BootstrapToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.config.Admin.BootstrapToken)) == 1 {
				ctx := context.WithValue(r.Context(), adminActorIDKey, "bootstrap-admin")
				ctx = context.WithValue(ctx, adminAuthMethodKey, adminAuthMethodBootstrap)
				ctx = admindomain.WithRequestMetadata(ctx, admindomain.RequestMetadata{
					IPAddress: ip,
					UserAgent: r.UserAgent(),
					RequestID: middleware.GetReqID(r.Context()),
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if !s.authFailureLimiter.Allow(ip) {
				writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
				return
			}
		}

		// 2. Cloudflare Access JWT — added by CF proxy for all requests through the access policy.
		if s.cfAccessVerifier != nil {
			if cfToken := strings.TrimSpace(r.Header.Get("Cf-Access-Jwt-Assertion")); cfToken != "" {
				email, err := s.cfAccessVerifier.Verify(cfToken)
				if err != nil {
					if !s.authFailureLimiter.Allow(ip) {
						writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
						return
					}
					writeError(w, http.StatusUnauthorized, "admin authorization required")
					return
				}
				ctx := context.WithValue(r.Context(), adminActorIDKey, email)
				ctx = context.WithValue(ctx, adminAuthMethodKey, adminAuthMethodCloudflare)
				ctx = admindomain.WithRequestMetadata(ctx, admindomain.RequestMetadata{
					IPAddress: ip,
					UserAgent: r.UserAgent(),
					RequestID: middleware.GetReqID(r.Context()),
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if !s.authFailureLimiter.Allow(ip) {
			writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
			return
		}
		writeError(w, http.StatusUnauthorized, "admin authorization required")
	})
}

func (s *server) adminCSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !unsafeHTTPMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		method, _ := r.Context().Value(adminAuthMethodKey).(adminAuthMethod)
		if method == adminAuthMethodBootstrap {
			// Bootstrap token requests are not browser-originated; CSRF does not apply.
			next.ServeHTTP(w, r)
			return
		}
		if sameOriginAdminRequest(r) || validateSPACSRFToken(r) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusForbidden, "admin csrf validation failed")
	})
}

func (s *server) accountSessionCSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !unsafeHTTPMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if sameOriginBrowserRequest(r) || validateSPACSRFToken(r) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusForbidden, "csrf validation failed")
	})
}

func unsafeHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

// sameOriginRequest checks whether r appears to originate from the same origin.
// allowMissingHeaders controls what happens when Sec-Fetch-Site, Origin, and
// Referer are all absent:
//   - true  (browser endpoints): allow, because non-browser API clients omit
//     these headers and CSRF attacks require a browser
//   - false (admin endpoints): deny, because admin access requires explicit proof
func sameOriginRequest(r *http.Request, allowMissingHeaders bool) bool {
	switch strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")) {
	case "same-origin", "same-site", "none":
		return true
	case "cross-site":
		return false
	}

	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return requestOriginMatchesHost(r, origin)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return requestOriginMatchesHost(r, referer)
	}

	return allowMissingHeaders
}

func sameOriginBrowserRequest(r *http.Request) bool {
	return sameOriginRequest(r, true)
}

func sameOriginAdminRequest(r *http.Request) bool {
	return sameOriginRequest(r, false)
}

func requestOriginMatchesHost(r *http.Request, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	return strings.EqualFold(u.Host, host)
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

func (s *server) developerAuth(next http.Handler) http.Handler {
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
		if !roleAllowed([]string{"developer", "admin"}, session.Roles) {
			writeError(w, http.StatusForbidden, "developer access required")
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
		ip := clientIP(r, s.config.Security.TrustedProxies)
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			if !s.appTokenLimiter.Allow(ip) {
				writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
				return
			}
			writeError(w, http.StatusUnauthorized, "app authorization required")
			return
		}
		appActor, err := s.accountSvc.ResolveAppByToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, app.ErrAppNotFound) {
				if !s.appTokenLimiter.Allow(ip) {
					writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
					return
				}
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

// corsMiddleware sets CORS headers for requests whose Origin matches the allowlist.
// Preflight OPTIONS requests are answered immediately with 204.
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[strings.TrimRight(origin, "/")]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
				if r.Method == http.MethodOptions {
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token, X-Requested-With")
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
