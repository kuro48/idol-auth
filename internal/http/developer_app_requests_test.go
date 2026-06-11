package http_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
	apphttp "github.com/kuro48/idol-auth/internal/http"
)

// controlledDeveloperService allows tests to inject specific responses.
type controlledDeveloperService struct {
	listResult   []appreg.Request
	listErr      error
	getResult    appreg.Request
	getErr       error
	submitResult appreg.Request
	submitErr    error
	withdrawErr  error
	resubmitErr  error

	autoApproveResult    appreg.Request
	autoApproveErr       error
	autoApproveCalled    bool
	lastAutoApproveInput appreg.SubmitInput
	lastSubmitInput      appreg.SubmitInput
}

func (s *controlledDeveloperService) ListMine(_ context.Context, _ string) ([]appreg.Request, error) {
	return s.listResult, s.listErr
}

func (s *controlledDeveloperService) GetForOwner(_ context.Context, id uuid.UUID, _ string) (appreg.Request, error) {
	if s.getErr != nil {
		return appreg.Request{}, s.getErr
	}
	r := s.getResult
	r.ID = id
	return r, nil
}

func (s *controlledDeveloperService) Submit(_ context.Context, _ string, input appreg.SubmitInput) (appreg.Request, error) {
	s.lastSubmitInput = input
	if s.submitErr != nil {
		return appreg.Request{}, s.submitErr
	}
	r := s.submitResult
	if r.ID == (uuid.UUID{}) {
		r.ID = uuid.New()
	}
	r.Name = input.Name
	return r, nil
}

func (s *controlledDeveloperService) SubmitAutoApproved(_ context.Context, _ string, input appreg.SubmitInput, provision appreg.ProvisionFunc) (appreg.Request, error) {
	s.autoApproveCalled = true
	s.lastAutoApproveInput = input
	if s.autoApproveErr != nil {
		return appreg.Request{}, s.autoApproveErr
	}
	if _, _, err := provision(s.autoApproveResult); err != nil {
		return appreg.Request{}, err
	}
	return s.autoApproveResult, nil
}

func (s *controlledDeveloperService) Withdraw(_ context.Context, id uuid.UUID, _ string) (appreg.Request, error) {
	if s.withdrawErr != nil {
		return appreg.Request{}, s.withdrawErr
	}
	return appreg.Request{ID: id}, nil
}

func (s *controlledDeveloperService) Resubmit(_ context.Context, id uuid.UUID, _ string, input appreg.SubmitInput) (appreg.Request, error) {
	if s.resubmitErr != nil {
		return appreg.Request{}, s.resubmitErr
	}
	return appreg.Request{ID: id, Name: input.Name}, nil
}

func authenticatedAuthService() *stubAuthService {
	return &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-abc",
			Email:         "dev@example.com",
			DisplayName:   "Dev User",
		},
	}
}

func routerWithDeveloperSvc(svc apphttp.DeveloperAppRegService) http.Handler {
	cfg := testConfig()
	cfg.DeveloperAppRegSvc = svc
	return apphttp.NewRouter(cfg, &stubAdminService{}, nil, authenticatedAuthService())
}

// extractCSRF reads the CSRF cookie value and form hidden-field value from a response.
func extractCSRF(w *httptest.ResponseRecorder) (cookieVal, tokenVal string) {
	for _, c := range w.Result().Cookies() {
		if c.Name == "idol_auth_devreq_csrf" {
			cookieVal = c.Value
		}
	}
	body := w.Body.String()
	const marker = `name="csrf_token" value="`
	if idx := strings.Index(body, marker); idx >= 0 {
		rest := body[idx+len(marker):]
		if end := strings.Index(rest, `"`); end >= 0 {
			tokenVal = rest[:end]
		}
	}
	return
}

// --- list page ---

func TestDeveloperListPageRequiresAuth(t *testing.T) {
	cfg := testConfig()
	cfg.DeveloperAppRegSvc = &controlledDeveloperService{}
	router := apphttp.NewRouter(cfg, nil, nil, &stubAuthService{})

	req := httptest.NewRequest(http.MethodGet, "/developer/app-requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect for unauthenticated, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "login") {
		t.Errorf("expected redirect to login, got %q", w.Header().Get("Location"))
	}
}

func TestDeveloperListPageRendersEmpty(t *testing.T) {
	router := routerWithDeveloperSvc(&controlledDeveloperService{listResult: []appreg.Request{}})

	req := httptest.NewRequest(http.MethodGet, "/developer/app-requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, fragment := range []string{"アプリ申請一覧", "/developer/app-requests/new", "brand-mark"} {
		if !strings.Contains(body, fragment) {
			t.Errorf("expected body to contain %q", fragment)
		}
	}
}

func TestDeveloperListPageShowsRequests(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	svc := &controlledDeveloperService{
		listResult: []appreg.Request{
			{ID: id, Name: "My App", Type: "web", Status: appreg.StatusPending},
		},
	}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodGet, "/developer/app-requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "My App") {
		t.Errorf("expected request name in body")
	}
	if !strings.Contains(body, id.String()) {
		t.Errorf("expected request ID link in body")
	}
}

// --- new form page ---

func TestDeveloperNewFormPageRendersCorrectly(t *testing.T) {
	router := routerWithDeveloperSvc(&controlledDeveloperService{})

	req := httptest.NewRequest(http.MethodGet, "/developer/app-requests/new", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, fragment := range []string{"新規アプリ登録", "csrf_token", `name="name"`, `name="type"`, `name="purpose"`} {
		if !strings.Contains(body, fragment) {
			t.Errorf("expected form to contain %q", fragment)
		}
	}
}

// --- create (POST /developer/app-requests) ---

func TestDeveloperCreateRejectsMissingCSRF(t *testing.T) {
	router := routerWithDeveloperSvc(&controlledDeveloperService{})

	form := url.Values{"name": {"My App"}, "type": {"web"}}
	req := httptest.NewRequest(http.MethodPost, "/developer/app-requests", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF, got %d", w.Code)
	}
}

func TestDeveloperCreateWithValidationErrorReRendersForm(t *testing.T) {
	svc := &controlledDeveloperService{submitErr: appreg.ErrInvalidName}
	router := routerWithDeveloperSvc(svc)

	getReq := httptest.NewRequest(http.MethodGet, "/developer/app-requests/new", nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	csrfCookieVal, csrfToken := extractCSRF(getW)

	form := url.Values{"csrf_token": {csrfToken}, "name": {"x"}}
	postReq := httptest.NewRequest(http.MethodPost, "/developer/app-requests", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "idol_auth_devreq_csrf", Value: csrfCookieVal})
	postW := httptest.NewRecorder()
	router.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("expected 200 re-render, got %d", postW.Code)
	}
	if !strings.Contains(postW.Body.String(), "アプリ名を確認してください") {
		t.Errorf("expected validation error in re-rendered form")
	}
}

func TestDeveloperCreateSuccessRedirectsToDetail(t *testing.T) {
	fixedID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	svc := &controlledDeveloperService{submitResult: appreg.Request{ID: fixedID}}
	router := routerWithDeveloperSvc(svc)

	getReq := httptest.NewRequest(http.MethodGet, "/developer/app-requests/new", nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	csrfCookieVal, csrfToken := extractCSRF(getW)

	form := url.Values{"csrf_token": {csrfToken}, "name": {"Demo"}, "type": {"web"}}
	postReq := httptest.NewRequest(http.MethodPost, "/developer/app-requests", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "idol_auth_devreq_csrf", Value: csrfCookieVal})
	postW := httptest.NewRecorder()
	router.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", postW.Code)
	}
	if loc := postW.Header().Get("Location"); !strings.Contains(loc, fixedID.String()) {
		t.Errorf("expected redirect to detail page, got %q", loc)
	}
}

// --- detail page ---

func TestDeveloperDetailPageReturns404ForMissingRequest(t *testing.T) {
	svc := &controlledDeveloperService{getErr: appreg.ErrRequestNotFound}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodGet, "/developer/app-requests/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeveloperDetailPageReturns403ForWrongOwner(t *testing.T) {
	svc := &controlledDeveloperService{getErr: appreg.ErrNotOwner}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodGet, "/developer/app-requests/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestDeveloperDetailPageRendersRequest(t *testing.T) {
	svc := &controlledDeveloperService{
		getResult: appreg.Request{Name: "My Test App", Type: "web", Status: appreg.StatusPending},
	}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodGet, "/developer/app-requests/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "My Test App") {
		t.Errorf("expected app name in body")
	}
	if !strings.Contains(body, "pending") {
		t.Errorf("expected status badge in body")
	}
}

// --- withdraw ---

func TestDeveloperWithdrawRejectsMissingCSRF(t *testing.T) {
	router := routerWithDeveloperSvc(&controlledDeveloperService{})

	req := httptest.NewRequest(http.MethodPost, "/developer/app-requests/"+uuid.New().String()+"/withdraw",
		bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF, got %d", w.Code)
	}
}

func TestDeveloperWithdrawSuccessRedirectsToList(t *testing.T) {
	svc := &controlledDeveloperService{getResult: appreg.Request{Status: appreg.StatusPending}}
	router := routerWithDeveloperSvc(svc)
	id := uuid.New()

	getReq := httptest.NewRequest(http.MethodGet, "/developer/app-requests/"+id.String(), nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	csrfCookieVal, csrfToken := extractCSRF(getW)

	form := url.Values{"csrf_token": {csrfToken}}
	postReq := httptest.NewRequest(http.MethodPost, "/developer/app-requests/"+id.String()+"/withdraw",
		strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "idol_auth_devreq_csrf", Value: csrfCookieVal})
	postW := httptest.NewRecorder()
	router.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", postW.Code)
	}
	if loc := postW.Header().Get("Location"); loc != "/developer/app-requests" {
		t.Errorf("expected redirect to list, got %q", loc)
	}
}

// --- JSON API ---

func TestDeveloperAPIListRequiresAuth(t *testing.T) {
	cfg := testConfig()
	cfg.DeveloperAppRegSvc = &controlledDeveloperService{}
	router := apphttp.NewRouter(cfg, nil, nil, &stubAuthService{})

	req := httptest.NewRequest(http.MethodGet, "/v1/developer/app-requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeveloperAPIListReturnsItems(t *testing.T) {
	svc := &controlledDeveloperService{
		listResult: []appreg.Request{{ID: uuid.New(), Name: "App One", Status: appreg.StatusPending}},
	}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/developer/app-requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "App One") {
		t.Errorf("expected app name in response")
	}
}

func TestDeveloperAPISubmitRequiresAuth(t *testing.T) {
	cfg := testConfig()
	cfg.DeveloperAppRegSvc = &controlledDeveloperService{}
	router := apphttp.NewRouter(cfg, nil, nil, &stubAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/v1/developer/app-requests",
		bytes.NewBufferString(`{"Name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeveloperAPISubmitRejectsUnknownFields(t *testing.T) {
	router := routerWithDeveloperSvc(&controlledDeveloperService{})

	req := httptest.NewRequest(http.MethodPost, "/v1/developer/app-requests",
		bytes.NewBufferString(`{"Name":"test","unknown_field":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown JSON fields, got %d", w.Code)
	}
}

func TestDeveloperAPIGetNotFound(t *testing.T) {
	svc := &controlledDeveloperService{getErr: appreg.ErrRequestNotFound}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/developer/app-requests/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeveloperAPIWithdrawForbiddenForWrongStatus(t *testing.T) {
	svc := &controlledDeveloperService{withdrawErr: appreg.ErrCannotWithdraw}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/developer/app-requests/"+uuid.New().String()+"/withdraw",
		bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeveloperAPIMutatingRequestRejectsCrossSiteOrigin(t *testing.T) {
	router := routerWithDeveloperSvc(&controlledDeveloperService{})

	req := httptest.NewRequest(http.MethodPost, "/v1/developer/app-requests",
		bytes.NewBufferString(`{"Name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestDeveloperAPIResubmitInvalidTransition(t *testing.T) {
	svc := &controlledDeveloperService{resubmitErr: appreg.ErrCannotResubmit}
	router := routerWithDeveloperSvc(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/developer/app-requests/"+uuid.New().String()+"/resubmit",
		bytes.NewBufferString(`{"Name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestFriendlyAppRegErrorHandlesAllDomainErrors(t *testing.T) {
	errs := []error{
		appreg.ErrInvalidName,
		appreg.ErrInvalidType,
		appreg.ErrInvalidDescription,
		appreg.ErrInvalidPurpose,
		appreg.ErrInvalidEmail,
		appreg.ErrInvalidRedirectURI,
		appreg.ErrInvalidURL,
		appreg.ErrRedirectURIRequired,
		appreg.ErrCannotResubmit,
		appreg.ErrCannotWithdraw,
		appreg.ErrDuplicatePending,
		errors.New("unexpected"),
	}
	svc := &controlledDeveloperService{}
	router := routerWithDeveloperSvc(svc)

	for _, e := range errs {
		svc.submitErr = e

		getReq := httptest.NewRequest(http.MethodGet, "/developer/app-requests/new", nil)
		getW := httptest.NewRecorder()
		router.ServeHTTP(getW, getReq)
		csrfCookieVal, csrfToken := extractCSRF(getW)

		form := url.Values{"csrf_token": {csrfToken}}
		postReq := httptest.NewRequest(http.MethodPost, "/developer/app-requests", strings.NewReader(form.Encode()))
		postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		postReq.AddCookie(&http.Cookie{Name: "idol_auth_devreq_csrf", Value: csrfCookieVal})
		postW := httptest.NewRecorder()
		router.ServeHTTP(postW, postReq)

		if postW.Code != http.StatusOK {
			t.Errorf("for error %v: expected 200 re-render, got %d", e, postW.Code)
		}
	}
}
