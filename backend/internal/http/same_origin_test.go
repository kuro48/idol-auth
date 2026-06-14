package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// sameOriginTruthTable fixes the intended behaviour of sameOriginBrowserRequest
// and sameOriginAdminRequest so that a future refactor cannot silently change
// the access-control semantics.
//
// The key intentional difference between the two:
//   - sameOriginBrowserRequest: missing Sec-Fetch-Site/Origin/Referer → ALLOW
//     (non-browser API clients often omit these headers; CSRF requires a browser)
//   - sameOriginAdminRequest:   missing Sec-Fetch-Site/Origin/Referer → DENY
//     (admin endpoints require explicit proof of same-origin)
func TestSameOriginBrowserRequest(t *testing.T) {
	cases := []struct {
		name         string
		secFetchSite string
		origin       string
		referer      string
		host         string
		wantAllowed  bool
	}{
		{
			name:         "Sec-Fetch-Site same-origin → allowed",
			secFetchSite: "same-origin",
			wantAllowed:  true,
		},
		{
			name:         "Sec-Fetch-Site same-site → allowed",
			secFetchSite: "same-site",
			wantAllowed:  true,
		},
		{
			name:         "Sec-Fetch-Site none → allowed",
			secFetchSite: "none",
			wantAllowed:  true,
		},
		{
			name:         "Sec-Fetch-Site cross-site → denied",
			secFetchSite: "cross-site",
			wantAllowed:  false,
		},
		{
			name:        "matching Origin header → allowed",
			origin:      "http://localhost:8080",
			host:        "localhost:8080",
			wantAllowed: true,
		},
		{
			name:        "mismatched Origin header → denied",
			origin:      "http://evil.example.com",
			host:        "localhost:8080",
			wantAllowed: false,
		},
		{
			name:        "matching Referer header → allowed",
			referer:     "http://localhost:8080/some/path",
			host:        "localhost:8080",
			wantAllowed: true,
		},
		{
			name:        "mismatched Referer header → denied",
			referer:     "http://evil.example.com/path",
			host:        "localhost:8080",
			wantAllowed: false,
		},
		{
			// Non-browser API clients (curl, SDKs) omit all headers.
			// For browser endpoints this is allowed because CSRF requires a browser.
			name:        "no headers at all → allowed (non-browser client exemption)",
			wantAllowed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.host != "" {
				r.Host = tc.host
			}
			if tc.secFetchSite != "" {
				r.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				r.Header.Set("Referer", tc.referer)
			}
			got := sameOriginBrowserRequest(r)
			if got != tc.wantAllowed {
				t.Errorf("sameOriginBrowserRequest() = %v, want %v", got, tc.wantAllowed)
			}
		})
	}
}

func TestSameOriginAdminRequest(t *testing.T) {
	cases := []struct {
		name         string
		secFetchSite string
		origin       string
		referer      string
		host         string
		wantAllowed  bool
	}{
		{
			name:         "Sec-Fetch-Site same-origin → allowed",
			secFetchSite: "same-origin",
			wantAllowed:  true,
		},
		{
			name:         "Sec-Fetch-Site same-site → allowed",
			secFetchSite: "same-site",
			wantAllowed:  true,
		},
		{
			name:         "Sec-Fetch-Site none → allowed",
			secFetchSite: "none",
			wantAllowed:  true,
		},
		{
			name:         "Sec-Fetch-Site cross-site → denied",
			secFetchSite: "cross-site",
			wantAllowed:  false,
		},
		{
			name:        "matching Origin header → allowed",
			origin:      "http://localhost:8080",
			host:        "localhost:8080",
			wantAllowed: true,
		},
		{
			name:        "mismatched Origin header → denied",
			origin:      "http://evil.example.com",
			host:        "localhost:8080",
			wantAllowed: false,
		},
		{
			name:        "matching Referer header → allowed",
			referer:     "http://localhost:8080/admin",
			host:        "localhost:8080",
			wantAllowed: true,
		},
		{
			name:        "mismatched Referer header → denied",
			referer:     "http://evil.example.com/admin",
			host:        "localhost:8080",
			wantAllowed: false,
		},
		{
			// Admin endpoints require explicit proof of same-origin.
			// Missing headers means we cannot verify origin → deny.
			name:        "no headers at all → denied (admin requires proof)",
			wantAllowed: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.host != "" {
				r.Host = tc.host
			}
			if tc.secFetchSite != "" {
				r.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				r.Header.Set("Referer", tc.referer)
			}
			got := sameOriginAdminRequest(r)
			if got != tc.wantAllowed {
				t.Errorf("sameOriginAdminRequest() = %v, want %v", got, tc.wantAllowed)
			}
		})
	}
}
