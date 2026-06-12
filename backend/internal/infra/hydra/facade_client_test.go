package hydra

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFacadeClientIntrospectUsesPublicEndpoint(t *testing.T) {
	var gotPath string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"active":true}`))
	}))
	defer srv.Close()

	client := NewFacadeClient(srv.URL, "http://admin.invalid")
	out, status, err := client.Introspect(context.Background(), []byte("token=t&client_id=c&client_secret=s"))
	if err != nil {
		t.Fatalf("Introspect() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if gotPath != "/oauth2/introspect" {
		t.Fatalf("expected public introspection endpoint, got %q", gotPath)
	}
	if gotBody != "token=t&client_id=c&client_secret=s" {
		t.Fatalf("unexpected body %q", gotBody)
	}
	if !strings.Contains(string(out), `"active":true`) {
		t.Fatalf("unexpected response %s", string(out))
	}
}
