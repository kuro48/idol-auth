package demo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchKratosErrorFiltersCookiesAndEscapesID(t *testing.T) {
	var gotCookie string
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":{"message":"invalid flow"}}`))
	}))
	defer srv.Close()

	client := NewKratosFlowClient(srv.URL, "https://accounts.example.com")
	req := httptest.NewRequest(http.MethodGet, "/error?id=flow-1", nil)
	req.Header.Set("Cookie", "app_session=secret; ory_session=test")

	msg, code := fetchKratosError(context.Background(), client, req, "flow-1&admin=true")

	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "invalid flow" {
		t.Fatalf("expected kratos message, got %q", msg)
	}
	if gotCookie != "ory_session=test" {
		t.Fatalf("expected only Ory cookie to be forwarded, got %q", gotCookie)
	}
	if gotQuery != "id=flow-1%26admin%3Dtrue" {
		t.Fatalf("expected escaped id query, got %q", gotQuery)
	}
}

func TestFetchKratosErrorDoesNotExposeRawUpstreamBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "postgres://idol:secret@postgres:5432/idol_auth", http.StatusBadGateway)
	}))
	defer srv.Close()

	client := NewKratosFlowClient(srv.URL, "https://accounts.example.com")
	req := httptest.NewRequest(http.MethodGet, "/error?id=flow-1", nil)

	msg, code := fetchKratosError(context.Background(), client, req, "flow-1")

	if code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, code)
	}
	if strings.Contains(msg, "postgres") || strings.Contains(msg, "secret") {
		t.Fatalf("expected generic error message, got %q", msg)
	}
}
