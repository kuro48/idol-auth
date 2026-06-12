package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kuro48/idol-auth/internal/config"
	apphttp "github.com/kuro48/idol-auth/internal/http"
)

func TestLegalPages(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)

	cases := []struct {
		path        string
		wantTitle   string
		wantEyebrow string
	}{
		{"/legal/terms", "利用規約", "Legal"},
		{"/legal/privacy", "プライバシーポリシー", "Legal"},
		{"/legal/contact", "問い合わせ先", "Support"},
		{"/legal/incident", "障害時連絡先", "Support"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
			ct := w.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, "text/html") {
				t.Errorf("expected Content-Type text/html, got %q", ct)
			}
			body := w.Body.String()
			if !strings.Contains(body, tc.wantTitle) {
				t.Errorf("body missing title %q", tc.wantTitle)
			}
			if !strings.Contains(body, "OshiLink") {
				t.Error("body missing brand name OshiLink")
			}
			for _, link := range []string{"/legal/terms", "/legal/privacy", "/legal/contact", "/legal/incident"} {
				if !strings.Contains(body, link) {
					t.Errorf("body missing cross-link %q", link)
				}
			}
			if !strings.Contains(body, "最終更新日") {
				t.Error("body missing 最終更新日")
			}
		})
	}
}

func TestLegalPagesNoAuth(t *testing.T) {
	// Legal pages must be accessible without any auth service wired up.
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, nil, nil)

	paths := []string{"/legal/terms", "/legal/privacy", "/legal/contact", "/legal/incident"}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200 without auth, got %d", path, w.Code)
		}
	}
}
