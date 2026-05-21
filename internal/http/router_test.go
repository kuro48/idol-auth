package http_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/config"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
	apphttp "github.com/kuro48/idol-auth/internal/http"
)

func TestHandleHealthz(t *testing.T) {
	// Arrange
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", ct)
	}
	const want = "{\"status\":\"ok\"}\n"
	if got := w.Body.String(); got != want {
		t.Errorf("expected body %q, got %q", want, got)
	}
}

func TestHandleReadyz(t *testing.T) {
	// Arrange
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", ct)
	}
	const want = "{\"status\":\"ok\"}\n"
	if got := w.Body.String(); got != want {
		t.Errorf("expected body %q, got %q", want, got)
	}
}

func TestSwaggerDocsAreServed(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), nil, nil, nil)

	t.Run("ui", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs/index.html", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		if !strings.Contains(w.Body.String(), "Swagger UI") {
			t.Fatalf("expected Swagger UI response")
		}
	})

	t.Run("short path redirects to ui", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, w.Code)
		}
		if location := w.Header().Get("Location"); location != "/docs/index.html" {
			t.Fatalf("expected redirect to /docs/index.html, got %q", location)
		}
	})

	t.Run("spec", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs/doc.json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("expected JSON Content-Type, got %q", ct)
		}
		body := w.Body.String()
		for _, fragment := range []string{`"swagger": "2.0"`, `"title": "idol-auth API"`} {
			if !strings.Contains(body, fragment) {
				t.Fatalf("expected spec to contain %q", fragment)
			}
		}
	})
}

func TestLoginPageUsesAccountCenterDesignSystem(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	for _, fragment := range []string{
		"--oshi-weak:#fff1e8",
		"--surface-2:#fffaf6",
		"--radius-lg:28px",
		"brand-mark",
		"/register",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected login page to contain fragment %q", fragment)
		}
	}
}

func TestRegistrationPageRendersCorrectly(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	for _, fragment := range []string{
		"--oshi-weak:#fff1e8",
		"brand-mark",
		"/login",
		"self-service/registration/browser",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected registration page to contain fragment %q", fragment)
		}
	}
}

func TestNewRouter_UnknownRoute_Returns404(t *testing.T) {
	// Arrange
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleHealthz_MethodNotAllowed(t *testing.T) {
	// Arrange
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestRateLimitAppliedToAuthRoutes(t *testing.T) {
	cfg := testConfig()
	cfg.Limiter = &alwaysDenyLimiter{}
	router := apphttp.NewRouter(cfg, &stubAdminService{}, nil, &stubAuthService{})

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 from rate-limited router, got %d", w.Code)
	}
}

func TestRateLimitNotAppliedWhenNil(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{})

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when no rate limiter configured, got %d", w.Code)
	}
}

func TestRateLimitNotAppliedToHealthCheck(t *testing.T) {
	cfg := testConfig()
	cfg.Limiter = &alwaysDenyLimiter{}
	router := apphttp.NewRouter(cfg, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected healthz to bypass rate limit, got %d", w.Code)
	}
}

func TestPublicLoginDoesNotExposeUpstreamError(t *testing.T) {
	cfg := testConfig()
	cfg.PublicSvc = &stubPublicService{
		loginErr: errors.New("kratos returned 400: password policy details"),
	}
	router := apphttp.NewRouter(cfg, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/public/api/login", bytes.NewBufferString(`{"identifier":"user@example.com","password":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "invalid credentials") {
		t.Fatalf("expected generic error, got %s", body)
	}
	if strings.Contains(body, "kratos") || strings.Contains(body, "password policy") {
		t.Fatalf("expected upstream details to be hidden, got %s", body)
	}
}

type stubDeveloperService struct{}

func (s *stubDeveloperService) Submit(_ context.Context, _ string, input appreg.SubmitInput) (appreg.Request, error) {
	return appreg.Request{ID: uuid.New(), Name: input.Name}, nil
}

func (s *stubDeveloperService) GetForOwner(_ context.Context, id uuid.UUID, _ string) (appreg.Request, error) {
	return appreg.Request{ID: id, Name: "Demo App"}, nil
}

func (s *stubDeveloperService) ListMine(_ context.Context, _ string) ([]appreg.Request, error) {
	return []appreg.Request{}, nil
}

func (s *stubDeveloperService) Withdraw(_ context.Context, id uuid.UUID, _ string) (appreg.Request, error) {
	return appreg.Request{ID: id, Name: "Demo App"}, nil
}

func (s *stubDeveloperService) Resubmit(_ context.Context, id uuid.UUID, _ string, input appreg.SubmitInput) (appreg.Request, error) {
	return appreg.Request{ID: id, Name: input.Name}, nil
}

func TestPublicRegisterRejectsUnknownJSONFields(t *testing.T) {
	cfg := testConfig()
	cfg.PublicSvc = &stubPublicService{}
	router := apphttp.NewRouter(cfg, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/public/api/register", bytes.NewBufferString(`{"email":"user@example.com","password":"secret","role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %s", w.Body.String())
	}
}

type alwaysDenyLimiter struct{}

func (l *alwaysDenyLimiter) Allow(_ string) bool { return false }

type stubPublicService struct {
	registerResult apphttp.AuthResult
	registerErr    error
	loginResult    apphttp.AuthResult
	loginErr       error
}

func (s *stubPublicService) LoginURL(_ map[string]string) string { return "" }

func (s *stubPublicService) RegistrationURL(_ string) string { return "" }

func (s *stubPublicService) LogoutURL(_ map[string]string) string { return "" }

func (s *stubPublicService) Token(_ context.Context, _ []byte) ([]byte, int, error) {
	return nil, http.StatusOK, nil
}

func (s *stubPublicService) Revoke(_ context.Context, _ []byte) ([]byte, int, error) {
	return nil, http.StatusOK, nil
}

func (s *stubPublicService) Introspect(_ context.Context, _ []byte) ([]byte, int, error) {
	return nil, http.StatusOK, nil
}

func (s *stubPublicService) GetSession(_ context.Context, _ string) (apphttp.PublicSessionView, error) {
	return apphttp.PublicSessionView{}, nil
}

func (s *stubPublicService) Register(_ context.Context, _ apphttp.RegisterInput) (apphttp.AuthResult, error) {
	if s.registerErr != nil {
		return apphttp.AuthResult{}, s.registerErr
	}
	return s.registerResult, nil
}

func (s *stubPublicService) Login(_ context.Context, _ apphttp.LoginInput) (apphttp.AuthResult, error) {
	if s.loginErr != nil {
		return apphttp.AuthResult{}, s.loginErr
	}
	return s.loginResult, nil
}
