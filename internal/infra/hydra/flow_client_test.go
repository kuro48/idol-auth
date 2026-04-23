package hydra

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
)

// ---------------------------------------------------------------------------
// GetLoginRequest
// ---------------------------------------------------------------------------

func TestFlowClientGetLoginRequestParsesSkipAndSubject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/oauth2/auth/requests/login" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("login_challenge") != "challenge-abc" {
			t.Errorf("missing or wrong login_challenge query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"skip": true, "subject": "identity-123"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	got, err := client.GetLoginRequest(context.Background(), "challenge-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Skip {
		t.Error("expected Skip=true")
	}
	if got.Subject != "identity-123" {
		t.Errorf("Subject: got %q, want %q", got.Subject, "identity-123")
	}
}

func TestFlowClientGetLoginRequestReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	_, err := client.GetLoginRequest(context.Background(), "bad-challenge")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
}

// ---------------------------------------------------------------------------
// AcceptLoginRequest
// ---------------------------------------------------------------------------

func TestFlowClientAcceptLoginRequestSendsSubjectInBody(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://example.com/callback"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	_, err := client.AcceptLoginRequest(context.Background(), "challenge-xyz", "user-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody["subject"] != "user-456" {
		t.Errorf("expected subject %q in body, got %v", "user-456", gotBody["subject"])
	}
}

func TestFlowClientAcceptLoginRequestReturnsRedirectTo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://example.com/after-login"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	redirectTo, err := client.AcceptLoginRequest(context.Background(), "challenge-xyz", "user-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if redirectTo != "https://example.com/after-login" {
		t.Errorf("redirect_to: got %q, want %q", redirectTo, "https://example.com/after-login")
	}
}

// ---------------------------------------------------------------------------
// GetConsentRequest
// ---------------------------------------------------------------------------

func TestFlowClientGetConsentRequestParsesFullResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"subject":                         "user-789",
			"skip":                            true,
			"requested_scope":                 []string{"openid", "profile"},
			"requested_access_token_audience": []string{"api://default"},
			"client": map[string]any{
				"client_id":    "my-client",
				"client_name":  "My App",
				"skip_consent": true,
			},
		})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	got, err := client.GetConsentRequest(context.Background(), "consent-challenge-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subject != "user-789" {
		t.Errorf("Subject: got %q, want %q", got.Subject, "user-789")
	}
	if !got.Skip {
		t.Error("expected Skip=true")
	}
	if len(got.RequestedScope) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(got.RequestedScope))
	}
	if len(got.RequestedAccessTokenAudience) != 1 {
		t.Errorf("expected 1 audience, got %d", len(got.RequestedAccessTokenAudience))
	}
	if got.Client.ClientID != "my-client" {
		t.Errorf("Client.ClientID: got %q, want %q", got.Client.ClientID, "my-client")
	}
	if got.Client.ClientName != "My App" {
		t.Errorf("Client.ClientName: got %q, want %q", got.Client.ClientName, "My App")
	}
	if !got.Client.SkipConsent {
		t.Error("expected Client.SkipConsent=true")
	}
}

// ---------------------------------------------------------------------------
// AcceptConsentRequest
// ---------------------------------------------------------------------------

func TestFlowClientAcceptConsentRequestSendsGrantsWithoutSessionWhenClaimsEmpty(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://example.com/done"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	_, err := client.AcceptConsentRequest(
		context.Background(), "consent-ch",
		[]string{"openid"}, []string{"api://default"},
		apphttp.ConsentSessionClaims{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := gotBody["session"]; ok {
		t.Error("expected no 'session' key in body when claims are empty")
	}
	if gotBody["grant_scope"] == nil {
		t.Error("expected grant_scope in body")
	}
	if gotBody["grant_access_token_audience"] == nil {
		t.Error("expected grant_access_token_audience in body")
	}
}

func TestFlowClientAcceptConsentRequestIncludesSessionClaimsWhenSet(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://example.com/done"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	claims := apphttp.ConsentSessionClaims{
		AccessToken: map[string]any{"roles": []string{"admin", "editor"}},
		IDToken:     map[string]any{"roles": []string{"admin", "editor"}},
	}
	redirectTo, err := client.AcceptConsentRequest(
		context.Background(), "consent-ch",
		[]string{"openid"}, []string{},
		claims,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if redirectTo != "https://example.com/done" {
		t.Errorf("redirect_to: got %q, want %q", redirectTo, "https://example.com/done")
	}
	session, ok := gotBody["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'session' map in body, got %T: %v", gotBody["session"], gotBody["session"])
	}
	if _, ok := session["access_token"]; !ok {
		t.Error("expected session.access_token in body")
	}
	if _, ok := session["id_token"]; !ok {
		t.Error("expected session.id_token in body")
	}
}

// ---------------------------------------------------------------------------
// RejectConsentRequest
// ---------------------------------------------------------------------------

func TestFlowClientRejectConsentRequestSendsErrorFields(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://example.com/rejected"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	redirectTo, err := client.RejectConsentRequest(context.Background(), "consent-ch", "access_denied", "user denied")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody["error"] != "access_denied" {
		t.Errorf("error field: got %v, want %q", gotBody["error"], "access_denied")
	}
	if gotBody["error_description"] != "user denied" {
		t.Errorf("error_description field: got %v, want %q", gotBody["error_description"], "user denied")
	}
	if redirectTo != "https://example.com/rejected" {
		t.Errorf("redirect_to: got %q, want %q", redirectTo, "https://example.com/rejected")
	}
}

// ---------------------------------------------------------------------------
// GetLogoutRequest / AcceptLogoutRequest
// ---------------------------------------------------------------------------

func TestFlowClientGetLogoutRequestParsesSubject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("logout_challenge") != "logout-ch" {
			t.Errorf("missing or wrong logout_challenge query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"subject": "identity-logout"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	got, err := client.GetLogoutRequest(context.Background(), "logout-ch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subject != "identity-logout" {
		t.Errorf("Subject: got %q, want %q", got.Subject, "identity-logout")
	}
}

func TestFlowClientAcceptLogoutRequestReturnsRedirectTo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://example.com/post-logout"})
	}))
	defer srv.Close()

	client := NewFlowClient(srv.URL)
	redirectTo, err := client.AcceptLogoutRequest(context.Background(), "logout-ch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if redirectTo != "https://example.com/post-logout" {
		t.Errorf("redirect_to: got %q, want %q", redirectTo, "https://example.com/post-logout")
	}
}
