package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ryunosukekurokawa/idol-auth/internal/config"
	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
)

func TestLoginChallengeRedirectsToKratosWhenSessionMissing(t *testing.T) {
	authn := &stubAuthService{
		loginResult: apphttp.LoginFlowResult{
			Action:     apphttp.AuthActionRedirect,
			RedirectTo: "http://kratos:4433/self-service/login/browser?return_to=http://localhost:8080/v1/auth/login?login_challenge=login123",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/login?login_challenge=login123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}
	if got := w.Header().Get("Location"); !strings.Contains(got, "self-service/login/browser") {
		t.Fatalf("expected Kratos login redirect, got %q", got)
	}
}

func TestLoginChallengeAcceptsWhenSessionAvailable(t *testing.T) {
	authn := &stubAuthService{
		loginResult: apphttp.LoginFlowResult{
			Action:     apphttp.AuthActionRedirect,
			RedirectTo: "http://hydra:4444/oauth2/auth?login_verifier=accepted",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/login?login_challenge=login123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}
	if got := w.Header().Get("Location"); !strings.Contains(got, "login_verifier=accepted") {
		t.Fatalf("expected Hydra redirect, got %q", got)
	}
}

func TestConsentChallengeAcceptsAndRedirects(t *testing.T) {
	authn := &stubAuthService{
		consentResult: apphttp.ConsentFlowResult{
			RedirectTo: "http://hydra:4444/oauth2/auth?consent_verifier=accepted",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/consent?consent_challenge=consent123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}
	if got := w.Header().Get("Location"); !strings.Contains(got, "consent_verifier=accepted") {
		t.Fatalf("expected Hydra redirect, got %q", got)
	}
}

func TestConsentChallengeRendersInteractivePrompt(t *testing.T) {
	authn := &stubAuthService{
		consentResult: apphttp.ConsentFlowResult{
			Prompt: &apphttp.ConsentPrompt{
				Challenge:      "consent123",
				ClientID:       "third-party-client",
				ClientName:     "Third Party App",
				RequestedScope: []string{"openid", "profile"},
			},
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/consent?consent_challenge=consent123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Third Party App") || !strings.Contains(body, "profile") {
		t.Fatalf("unexpected body: %s", body)
	}
	if !strings.Contains(body, `name="csrf_token"`) {
		t.Fatalf("expected csrf token in consent form, got %s", body)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected clickjacking protection header, got %q", got)
	}
	if got := w.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") {
		t.Fatalf("expected consent CSP header, got %q", got)
	}
	foundCSRFCookie := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "idol_auth_consent_csrf" && cookie.Value != "" {
			foundCSRFCookie = true
		}
	}
	if !foundCSRFCookie {
		t.Fatal("expected consent csrf cookie to be set")
	}
}

func TestConsentPostRedirectsAfterDecision(t *testing.T) {
	authn := &stubAuthService{
		submitConsentResult: apphttp.AuthFlowResult{
			RedirectTo: "http://hydra:4444/oauth2/auth?consent_verifier=accepted",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/consent", strings.NewReader("consent_challenge=consent123&action=accept&csrf_token=csrf123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "idol_auth_consent_csrf", Value: "csrf123", Path: "/v1/auth/consent"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}
	if got := w.Header().Get("Location"); !strings.Contains(got, "consent_verifier=accepted") {
		t.Fatalf("expected Hydra redirect, got %q", got)
	}
	if !authn.submitConsentInput.Accept {
		t.Fatal("expected accept decision to be forwarded")
	}
}

func TestConsentPostRejectsMissingCSRF(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/consent", strings.NewReader("consent_challenge=consent123&action=accept"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestConsentPostRejectsUnknownAction(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/consent", strings.NewReader("consent_challenge=consent123&action=&csrf_token=csrf123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "idol_auth_consent_csrf", Value: "csrf123", Path: "/v1/auth/consent"})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionReturnsAuthenticatedPrincipal(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			Subject:       "user-123",
			IdentityID:    "identity-123",
			Email:         "user@example.com",
			Roles:         []string{"admin"},
			Methods:       []string{"password", "totp"},
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"authenticated":true`) || !strings.Contains(body, `"subject":"user-123"`) || !strings.Contains(body, `"email":"user@example.com"`) || !strings.Contains(body, `"roles":["admin"]`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestProvidersReturnsPublicEndpoints(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/providers", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "/self-service/login/browser") {
		t.Fatalf("expected providers response to include Kratos login URL, got %s", body)
	}
}

func TestLogoutStartReturnsHydraLogoutURL(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "/oauth2/sessions/logout") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestLogoutChallengeRedirects(t *testing.T) {
	authn := &stubAuthService{
		logoutResult: apphttp.AuthFlowResult{
			RedirectTo: "http://localhost:3000/logged-out",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/logout?logout_challenge=logout123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}
	if got := w.Header().Get("Location"); got != "http://localhost:3000/logged-out" {
		t.Fatalf("expected logout redirect, got %q", got)
	}
}

func TestSessionReturnsServiceUnavailableWhenAuthSvcNil(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleConsentSessionMismatchReturnsForbidden(t *testing.T) {
	authn := &stubAuthService{consentErr: apphttp.ErrConsentSessionMismatch}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/consent?consent_challenge=consent123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestHandleLogoutReturnsBadGatewayOnUpstreamError(t *testing.T) {
	authn := &stubAuthService{logoutErr: errors.New("hydra down")}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/logout?logout_challenge=logout123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
}

func TestHandleLoginReturnsServiceUnavailableWhenAuthSvcNil(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/login?login_challenge=login123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleConsentReturnsServiceUnavailableWhenAuthSvcNil(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/consent?consent_challenge=consent123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleLogoutReturnsServiceUnavailableWhenAuthSvcNil(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/logout?logout_challenge=logout123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

type stubAuthService struct {
	loginResult         apphttp.LoginFlowResult
	loginErr            error
	consentResult       apphttp.ConsentFlowResult
	consentErr          error
	submitConsentResult apphttp.AuthFlowResult
	submitConsentInput  apphttp.ConsentDecisionInput
	logoutResult        apphttp.AuthFlowResult
	logoutErr           error
	session             apphttp.SessionView
	sessionErr          error
}

func (s *stubAuthService) HandleLogin(_ context.Context, _ *http.Request, _ string) (apphttp.LoginFlowResult, error) {
	return s.loginResult, s.loginErr
}

func (s *stubAuthService) HandleConsent(_ context.Context, _ *http.Request, _ string) (apphttp.ConsentFlowResult, error) {
	return s.consentResult, s.consentErr
}

func (s *stubAuthService) SubmitConsent(_ context.Context, _ *http.Request, _ string, input apphttp.ConsentDecisionInput) (apphttp.AuthFlowResult, error) {
	s.submitConsentInput = input
	return s.submitConsentResult, nil
}

func (s *stubAuthService) HandleLogout(_ context.Context, _ string) (apphttp.AuthFlowResult, error) {
	return s.logoutResult, s.logoutErr
}

func (s *stubAuthService) CurrentSession(_ context.Context, _ *http.Request) (apphttp.SessionView, error) {
	if s.sessionErr != nil {
		return apphttp.SessionView{}, s.sessionErr
	}
	return s.session, nil
}

func testConfig() apphttp.RouterConfig {
	return apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
		App:   config.AppConfig{BaseURL: "http://localhost:8080"},
		Security: config.SecurityConfig{
			CookieSecure: true,
		},
		Ory: config.OryConfig{
			KratosPublicURL:  "http://kratos:4433",
			HydraPublicURL:   "http://hydra:4444",
			KratosBrowserURL: "http://localhost:4433",
			HydraBrowserURL:  "http://localhost:4444",
		},
	}
}
