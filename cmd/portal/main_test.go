package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kuro48/idol-auth/internal/demo"
)

func TestPortalRootRedirectsToAccountCenter(t *testing.T) {
	handler, err := newPortalHandler(&demo.PortalConfig{
		AppURL:           "https://accounts.example.com",
		AccountCenterURL: "https://auth.example.com/account/",
		KratosPublicURL:  "http://kratos:4433",
		KratosAdminURL:   "http://kratos:4434",
		KratosBrowserURL: "https://accounts.example.com",
	}, http.DefaultClient)
	if err != nil {
		t.Fatalf("newPortalHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if got := w.Header().Get("Location"); got != "https://auth.example.com/account/" {
		t.Fatalf("expected account center redirect, got %q", got)
	}
	if strings.Contains(w.Body.String(), "Idol Auth Portal") {
		t.Fatalf("expected portal summary page to be removed, got %q", w.Body.String())
	}
}

func TestPortalRootRedirectFallbackUsesAppURL(t *testing.T) {
	got := accountCenterURL(&demo.PortalConfig{AppURL: "https://auth.example.com/"})
	if got != "https://auth.example.com/account/" {
		t.Fatalf("expected fallback account center URL, got %q", got)
	}
}

func TestLegalBaseURLUsesAccountCenterOrigin(t *testing.T) {
	got := legalBaseURL(&demo.PortalConfig{
		AppURL:           "https://accounts.example.com",
		AccountCenterURL: "https://auth.example.com/account/",
	})
	if got != "https://auth.example.com" {
		t.Fatalf("expected account center origin, got %q", got)
	}
}

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
