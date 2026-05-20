package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kuro48/idol-auth/internal/demo"
)

func TestHandleLogoutRedirectsToKratosLogoutURL(t *testing.T) {
	kratos := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/self-service/logout/browser" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("return_to"); got != "https://accounts.example.com/login" {
			t.Fatalf("expected return_to login URL, got %q", got)
		}
		if got := r.Header.Get("Cookie"); got != "ory_session=test" {
			t.Fatalf("expected only Ory cookie to be forwarded, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"logout_url":"https://accounts.example.com/self-service/logout?token=abc"}`))
	}))
	defer kratos.Close()

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.Header.Set("Cookie", "foo=bar; ory_session=test")
	w := httptest.NewRecorder()

	handleLogout(w, req, &demo.PortalConfig{
		AppURL:          "https://accounts.example.com",
		KratosPublicURL: kratos.URL,
	}, kratos.Client())

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if got := w.Header().Get("Location"); got != "https://accounts.example.com/self-service/logout?token=abc" {
		t.Fatalf("expected Kratos logout redirect, got %q", got)
	}
}

func TestHandleLogoutRedirectsToLoginWhenNoActiveSession(t *testing.T) {
	kratos := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no session", http.StatusUnauthorized)
	}))
	defer kratos.Close()

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	w := httptest.NewRecorder()

	handleLogout(w, req, &demo.PortalConfig{
		AppURL:          "https://accounts.example.com/",
		KratosPublicURL: kratos.URL,
	}, kratos.Client())

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if got := w.Header().Get("Location"); got != "https://accounts.example.com/login" {
		t.Fatalf("expected login redirect, got %q", got)
	}
}

func TestFilterOryCookies(t *testing.T) {
	got := filterOryCookies("foo=bar; ory_session=test; csrf=x; ory_kratos_session=y")
	for _, want := range []string{"ory_session=test", "ory_kratos_session=y"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q to contain %q", got, want)
		}
	}
	if strings.Contains(got, "foo=bar") || strings.Contains(got, "csrf=x") {
		t.Fatalf("expected non-Ory cookies to be filtered, got %q", got)
	}
}

func TestRenderHomeUsesAccountCenterDesignSystem(t *testing.T) {
	rec := httptest.NewRecorder()

	renderHome(rec, "#ffb2d8")

	body := rec.Body.String()
	for _, fragment := range []string{
		"--oshi-weak:#fff1e8",
		"--surface-2:#fffaf6",
		"--radius-lg:8px",
		"brand-mark",
		"推し色を選ぶ",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected portal home to contain account center design fragment %q", fragment)
		}
	}
}

func TestRenderHomeUsesAccountCenterPortalLayout(t *testing.T) {
	rec := httptest.NewRecorder()

	renderHome(rec, "#ffb2d8")

	body := rec.Body.String()
	for _, fragment := range []string{
		`class="topbar"`,
		`profile-hero`,
		`class="service-list"`,
		`Account Portal`,
		`class="badge-oshi"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected portal home to contain account center layout fragment %q", fragment)
		}
	}
}

func TestRenderHomeAvoidsNeumorphicSurfaceTreatment(t *testing.T) {
	rec := httptest.NewRecorder()

	renderHome(rec, "#ffb2d8")

	body := rec.Body.String()
	for _, fragment := range []string{
		"backdrop-filter",
		"blur(",
		"0 16px 40px",
		"0 24px 64px",
		"rgba(255,255,255,0.9)",
		"linear-gradient(160deg",
		"radial-gradient",
		"border-radius: 28px",
	} {
		if strings.Contains(body, fragment) {
			t.Fatalf("expected portal home to avoid neumorphic fragment %q", fragment)
		}
	}
}
