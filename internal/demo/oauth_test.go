package demo

import (
	"net/url"
	"testing"
)

func TestBuildAuthorizationURL(t *testing.T) {
	u, err := BuildAuthorizationURL(AuthorizationParams{
		HydraBrowserURL: "http://localhost:4444",
		ClientID:        "demo-client",
		RedirectURI:     "http://localhost:3001/oauth/callback",
		State:           "state-123",
		CodeChallenge:   "challenge-123",
		Scopes:          []string{"openid", "offline_access"},
	})
	if err != nil {
		t.Fatalf("BuildAuthorizationURL() error = %v", err)
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Path != "/oauth2/auth" {
		t.Fatalf("expected authorize path, got %q", parsed.Path)
	}
	if parsed.Query().Get("client_id") != "demo-client" {
		t.Fatalf("unexpected client_id: %q", parsed.Query().Get("client_id"))
	}
	if parsed.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected code_challenge_method: %q", parsed.Query().Get("code_challenge_method"))
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	if got := normalizeBaseURL("http://localhost:4444/"); got != "http://localhost:4444" {
		t.Fatalf("unexpected normalized URL: %q", got)
	}
}
