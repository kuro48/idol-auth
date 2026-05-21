package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	apphttp "github.com/kuro48/idol-auth/internal/http"
)

type stubSettingsAuthService struct {
	stubAuthService
	flow         *apphttp.KratosSettingsFlow
	submitResult apphttp.KratosSettingsSubmitResult
	submittedID  string
	submitted    url.Values
}

func (s *stubSettingsAuthService) GetSettingsFlow(_ context.Context, _ *http.Request, _ string) (*apphttp.KratosSettingsFlow, error) {
	return s.flow, nil
}

func (s *stubSettingsAuthService) SubmitSettingsFlow(_ context.Context, _ *http.Request, flowID string, form url.Values) (apphttp.KratosSettingsSubmitResult, error) {
	s.submittedID = flowID
	s.submitted = form
	return s.submitResult, nil
}

func TestHandleAccountCenterLinksToLocalSettingsPage(t *testing.T) {
	authn := &stubSettingsAuthService{
		stubAuthService: stubAuthService{session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
			Email:         "fan@example.com",
		}},
	}
	router := apphttp.NewRouter(testConfig(), nil, nil, authn)

	req := httptest.NewRequest(http.MethodGet, "/account/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), `href="/settings"`) {
		t.Fatalf("expected account center to link to local settings page, got body %q", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "セキュリティ設定") {
		t.Fatalf("expected account center to label settings as security settings, got body %q", w.Body.String())
	}
}

func TestHandleSettingsRendersFlowWithProxySubmitAction(t *testing.T) {
	authn := &stubSettingsAuthService{
		stubAuthService: stubAuthService{session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
			Email:         "fan@example.com",
		}},
		flow: &apphttp.KratosSettingsFlow{
			ID:     "flow-123",
			Action: "http://kratos/self-service/settings?flow=flow-123",
			Method: "POST",
			Nodes: []apphttp.KratosSettingsNode{
				{Group: "default", Name: "csrf_token", InputType: "hidden", Value: "csrf"},
				{Group: "profile", Name: "traits.email", InputType: "email", Value: "fan@example.com", Label: "Email"},
				{Group: "profile", Name: "method", InputType: "submit", Value: "profile", Label: "保存"},
			},
		},
	}
	router := apphttp.NewRouter(testConfig(), nil, nil, authn)

	req := httptest.NewRequest(http.MethodGet, "/settings?flow=flow-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `action="/v1/settings/flow?flow=flow-123"`) {
		t.Fatalf("expected settings form to submit to proxy action, got body %q", body)
	}
	if !strings.Contains(body, `name="csrf_token" value="csrf"`) {
		t.Fatalf("expected CSRF token to be rendered, got body %q", body)
	}
}

func TestHandleSettingsFlowGetReturnsFlowJSON(t *testing.T) {
	authn := &stubSettingsAuthService{
		stubAuthService: stubAuthService{session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
		}},
		flow: &apphttp.KratosSettingsFlow{ID: "flow-123"},
	}
	router := apphttp.NewRouter(testConfig(), nil, nil, authn)

	req := httptest.NewRequest(http.MethodGet, "/v1/settings/flow?id=flow-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), `"id":"flow-123"`) {
		t.Fatalf("expected settings flow JSON, got %q", w.Body.String())
	}
}

func TestHandleSettingsFlowSubmitForwardsFormAndRedirects(t *testing.T) {
	authn := &stubSettingsAuthService{
		stubAuthService: stubAuthService{session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
		}},
		submitResult: apphttp.KratosSettingsSubmitResult{
			RedirectTo: "/account",
			SetCookies: []string{"ory_session=next; Path=/; HttpOnly"},
		},
	}
	router := apphttp.NewRouter(testConfig(), nil, nil, authn)

	req := httptest.NewRequest(http.MethodPost, "/v1/settings/flow?flow=flow-123", strings.NewReader("csrf_token=csrf&traits.email=next%40example.com&method=profile"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if authn.submittedID != "flow-123" {
		t.Fatalf("expected submitted flow id flow-123, got %q", authn.submittedID)
	}
	if got := authn.submitted.Get("traits.email"); got != "next@example.com" {
		t.Fatalf("expected submitted email, got %q", got)
	}
	if got := w.Header().Get("Set-Cookie"); !strings.Contains(got, "ory_session=next") {
		t.Fatalf("expected Ory cookie to be relayed, got %q", got)
	}
	if got := w.Header().Get("Location"); got != "/account" {
		t.Fatalf("expected redirect to /account, got %q", got)
	}
}

func TestHandleSettingsFlowSubmitRejectsExternalRedirect(t *testing.T) {
	authn := &stubSettingsAuthService{
		stubAuthService: stubAuthService{session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
		}},
		submitResult: apphttp.KratosSettingsSubmitResult{
			RedirectTo: "https://evil.example.com/phishing",
		},
	}
	router := apphttp.NewRouter(testConfig(), nil, nil, authn)

	req := httptest.NewRequest(http.MethodPost, "/v1/settings/flow?flow=flow-123", strings.NewReader("csrf_token=csrf&method=profile"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if got := w.Header().Get("Location"); got != "/account" {
		t.Fatalf("expected external redirect to be replaced with /account, got %q", got)
	}
}

func TestHandleSettingsFlowSubmitRejectsSchemeRelativeRedirect(t *testing.T) {
	authn := &stubSettingsAuthService{
		stubAuthService: stubAuthService{session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
		}},
		submitResult: apphttp.KratosSettingsSubmitResult{
			RedirectTo: "//evil.example.com/phishing",
		},
	}
	router := apphttp.NewRouter(testConfig(), nil, nil, authn)

	req := httptest.NewRequest(http.MethodPost, "/v1/settings/flow?flow=flow-123", strings.NewReader("csrf_token=csrf&method=profile"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if got := w.Header().Get("Location"); got != "/account" {
		t.Fatalf("expected scheme-relative redirect to be replaced with /account, got %q", got)
	}
}

func TestHandleSettingsFlowSubmitAllowsConfiguredKratosRedirect(t *testing.T) {
	authn := &stubSettingsAuthService{
		stubAuthService: stubAuthService{session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
		}},
		submitResult: apphttp.KratosSettingsSubmitResult{
			RedirectTo: "http://localhost:4433/self-service/login/browser?aal=aal2",
		},
	}
	router := apphttp.NewRouter(testConfig(), nil, nil, authn)

	req := httptest.NewRequest(http.MethodPost, "/v1/settings/flow?flow=flow-123", strings.NewReader("csrf_token=csrf&method=profile"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if got := w.Header().Get("Location"); got != "http://localhost:4433/self-service/login/browser?aal=aal2" {
		t.Fatalf("expected configured Kratos redirect to pass, got %q", got)
	}
}
