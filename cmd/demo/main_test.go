package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ryunosukekurokawa/idol-auth/internal/demo"
)

func TestSanitizeTokenResponse(t *testing.T) {
	input := map[string]any{
		"access_token":  "access-value",
		"refresh_token": "refresh-value",
		"id_token":      "id-value",
		"expires_in":    300,
	}

	sanitized := sanitizeTokenResponse(input)

	for _, key := range []string{"access_token", "refresh_token", "id_token"} {
		if sanitized[key] != "<redacted>" {
			t.Fatalf("expected %s to be redacted, got %#v", key, sanitized[key])
		}
	}
	if sanitized["expires_in"] != 300 {
		t.Fatalf("expected non-token fields to be preserved, got %#v", sanitized["expires_in"])
	}
}

func TestExchangeCodeUsesHydraPublicURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/token" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if got := r.FormValue("client_id"); got != "client-123" {
			t.Fatalf("unexpected client_id: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "x", "token_type": "bearer"})
	}))
	defer srv.Close()

	cfg := &demo.Config{
		AppURL:         "http://localhost:3002",
		HydraPublicURL: srv.URL,
	}

	tokenResp, err := exchangeCode(context.Background(), cfg, "client-123", "verifier-123", "code-123")
	if err != nil {
		t.Fatalf("exchangeCode() error = %v", err)
	}
	if tokenResp["access_token"] != "x" {
		t.Fatalf("unexpected token response: %#v", tokenResp)
	}
}

func TestRenderHomeRendersJapaneseLabels(t *testing.T) {
	rec := httptest.NewRecorder()

	renderHome(rec, &demo.Config{})

	body := rec.Body.String()
	for _, fragment := range []string{
		"Idol Auth",
		"ファーストパーティでログイン",
		"パートナーアプリでログイン",
		"アカウントを作成",
		"ログイン画面を開く",
		"セキュリティ設定",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected %q in body, got %s", fragment, body)
		}
	}
}

func TestRenderTokenRendersJapaneseLabels(t *testing.T) {
	rec := httptest.NewRecorder()

	renderToken(rec, map[string]any{"token_type": "bearer"})

	body := rec.Body.String()
	for _, fragment := range []string{
		"OIDC コールバック完了",
		"デモのホームに戻る",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected %q in body, got %s", fragment, body)
		}
	}
}
