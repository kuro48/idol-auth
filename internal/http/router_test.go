package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ryunosukekurokawa/idol-auth/internal/config"
	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
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

func TestHandleDocs(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/docs/index.html", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected Content-Type %q, got %q", "text/html; charset=utf-8", ct)
	}
	if csp := w.Header().Get("Content-Security-Policy"); csp == "" {
		t.Fatal("expected docs handler to set Content-Security-Policy")
	}
	body := w.Body.String()
	for _, fragment := range []string{
		`<title>Swagger UI</title>`,
		`swagger-ui.css`,
		`SwaggerUIBundle`,
		`doc.json`,
		`StandaloneLayout`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected body to contain %q", fragment)
		}
	}
}

func TestHandleDocsIndexRedirects(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, w.Code)
	}
	if got := w.Header().Get("Location"); got != "/docs/index.html" {
		t.Fatalf("expected redirect location %q, got %q", "/docs/index.html", got)
	}
}

func TestHandleSwaggerDocJSON(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/docs/doc.json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json; charset=utf-8", ct)
	}
	body := w.Body.String()
	for _, fragment := range []string{
		`"swagger": "2.0"`,
		`idol-auth API`,
		`"/v1/admin/apps"`,
		`"/v1/account"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected spec to contain %q", fragment)
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

type alwaysDenyLimiter struct{}

func (l *alwaysDenyLimiter) Allow(_ string) bool { return false }
