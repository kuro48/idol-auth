package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeRoles(t *testing.T) {
	got := normalizeRoles([]string{" Admin ", "platform-operator", "admin", ""})
	want := []string{"admin", "platform-operator"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestRunSetRoles(t *testing.T) {
	var authHeader string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v1/admin/identities/identity-123/roles" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"identity_id":"identity-123","roles":["admin","platform-operator"]}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"set-roles",
		"--base-url", server.URL,
		"--token", "secret-token",
		"--identity-id", "identity-123",
		"--roles", "platform-operator,admin,admin",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if authHeader != "Bearer secret-token" {
		t.Fatalf("unexpected auth header: %q", authHeader)
	}
	roles, ok := body["roles"].([]any)
	if !ok || len(roles) != 2 || roles[0] != "admin" || roles[1] != "platform-operator" {
		t.Fatalf("unexpected roles payload: %#v", body["roles"])
	}
	if !strings.Contains(stdout.String(), `"identity_id":"identity-123"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestSetRolesReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer server.Close()

	_, err := setRoles(context.Background(), server.URL, "secret-token", "identity-123", []string{"admin"})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 error, got %v", err)
	}
}
