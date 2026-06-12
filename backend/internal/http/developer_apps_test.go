package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
	apphttp "github.com/kuro48/idol-auth/internal/http"
)

func instantCreateBody() string {
	return `{
		"name": "My Instant App",
		"type": "web",
		"description": "An instantly registered app",
		"redirect_uris": ["https://example.com/callback"],
		"scopes": ["openid", "email"]
	}`
}

func newInstantCreateRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/developer/apps", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return req
}

func routerForInstantApps(svc apphttp.DeveloperAppRegService, adminSvc *stubAdminService) http.Handler {
	cfg := testConfig()
	cfg.DeveloperAppRegSvc = svc
	return apphttp.NewRouter(cfg, adminSvc, nil, authenticatedAuthService())
}

func TestInstantCreateApp_RequiresAuth(t *testing.T) {
	cfg := testConfig()
	cfg.DeveloperAppRegSvc = &controlledDeveloperService{}
	router := apphttp.NewRouter(cfg, &stubAdminService{}, nil, &stubAuthService{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, newInstantCreateRequest(instantCreateBody()))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestInstantCreateApp_Success(t *testing.T) {
	svc := &controlledDeveloperService{
		autoApproveResult: appreg.Request{
			ID:     uuid.New(),
			Status: appreg.StatusApproved,
		},
	}
	router := routerForInstantApps(svc, &stubAdminService{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, newInstantCreateRequest(instantCreateBody()))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if _, ok := resp["management_token"]; !ok {
		t.Errorf("expected management_token in response, got %v", resp)
	}
	if _, ok := resp["client_secret"]; !ok {
		t.Errorf("expected client_secret in response, got %v", resp)
	}
	if !svc.autoApproveCalled {
		t.Error("expected SubmitAutoApproved to be called")
	}
	if svc.lastAutoApproveInput.ContactEmail != "dev@example.com" {
		t.Errorf("expected contact email defaulted to session email, got %q", svc.lastAutoApproveInput.ContactEmail)
	}
}

func TestInstantCreateApp_AppLimitReached(t *testing.T) {
	var active []appreg.Request
	for i := 0; i < apphttp.MaxSelfServiceAppsPerDeveloper; i++ {
		active = append(active, appreg.Request{ID: uuid.New(), Status: appreg.StatusApproved})
	}
	svc := &controlledDeveloperService{listResult: active}
	router := routerForInstantApps(svc, &stubAdminService{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, newInstantCreateRequest(instantCreateBody()))

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 when app limit reached, got %d", w.Code)
	}
	if svc.autoApproveCalled {
		t.Error("expected SubmitAutoApproved not to be called when over limit")
	}
}

func TestInstantCreateApp_WithdrawnAndRejectedDoNotCount(t *testing.T) {
	var inactive []appreg.Request
	for i := 0; i < apphttp.MaxSelfServiceAppsPerDeveloper; i++ {
		inactive = append(inactive,
			appreg.Request{ID: uuid.New(), Status: appreg.StatusRejected},
			appreg.Request{ID: uuid.New(), Status: appreg.StatusWithdrawn},
		)
	}
	svc := &controlledDeveloperService{
		listResult:        inactive,
		autoApproveResult: appreg.Request{ID: uuid.New(), Status: appreg.StatusApproved},
	}
	router := routerForInstantApps(svc, &stubAdminService{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, newInstantCreateRequest(instantCreateBody()))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInstantCreateApp_ScopeNotAllowed(t *testing.T) {
	svc := &controlledDeveloperService{autoApproveErr: appreg.ErrScopeNotAllowed}
	router := routerForInstantApps(svc, &stubAdminService{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, newInstantCreateRequest(instantCreateBody()))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for disallowed scope, got %d", w.Code)
	}
}

func TestInstantCreateApp_CSRFRejected(t *testing.T) {
	svc := &controlledDeveloperService{}
	router := routerForInstantApps(svc, &stubAdminService{})

	req := httptest.NewRequest(http.MethodPost, "/v1/developer/apps", strings.NewReader(instantCreateBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-site request, got %d", w.Code)
	}
}

func TestInstantCreateAppHTML_ShowsCredentialsOnce(t *testing.T) {
	svc := &controlledDeveloperService{
		autoApproveResult: appreg.Request{
			ID:     uuid.New(),
			Status: appreg.StatusApproved,
			Name:   "My Instant App",
		},
	}
	router := routerForInstantApps(svc, &stubAdminService{})

	// Obtain a CSRF cookie + token from the form page first.
	formReq := httptest.NewRequest(http.MethodGet, "/developer/app-requests/new", nil)
	formW := httptest.NewRecorder()
	router.ServeHTTP(formW, formReq)
	cookieVal, tokenVal := extractCSRF(formW)
	if cookieVal == "" || tokenVal == "" {
		t.Fatalf("failed to extract CSRF token from form page")
	}

	form := url.Values{
		"csrf_token":    {tokenVal},
		"name":          {"My Instant App"},
		"type":          {"web"},
		"description":   {"An instantly registered app"},
		"redirect_uris": {"https://example.com/callback"},
	}
	req := httptest.NewRequest(http.MethodPost, "/developer/apps", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "idol_auth_devreq_csrf", Value: cookieVal})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "mgmt-secret") {
		t.Error("expected management token displayed on credentials page")
	}
	if !strings.Contains(body, "一度しか表示されません") {
		t.Error("expected one-time display warning on credentials page")
	}
	if !svc.lastAutoApproveInput.SelfService {
		t.Error("expected SelfService flag set on HTML form submission")
	}
}

func TestInstantCreateAppHTML_MissingCSRFRejected(t *testing.T) {
	svc := &controlledDeveloperService{}
	router := routerForInstantApps(svc, &stubAdminService{})

	form := url.Values{"name": {"App"}}
	req := httptest.NewRequest(http.MethodPost, "/developer/apps", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without CSRF token, got %d", w.Code)
	}
}

// TestSubmitAppRequest_SelfServiceFieldIgnored ensures the review-flow JSON API
// cannot smuggle SelfService=true to bypass purpose validation.
func TestSubmitAppRequest_SelfServiceFieldIgnored(t *testing.T) {
	svc := &controlledDeveloperService{submitResult: appreg.Request{ID: uuid.New()}}
	router := routerForInstantApps(svc, &stubAdminService{})

	body := `{"Name": "App", "SelfService": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/developer/app-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if svc.lastSubmitInput.SelfService {
		t.Error("expected SelfService flag to be stripped from review-flow submissions")
	}
}
