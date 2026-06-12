package demo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newSiteverifyServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("siteverify method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("siteverify parse form: %v", err)
		}
		if got := r.PostFormValue("secret"); got != "test-secret" {
			t.Errorf("siteverify secret = %q, want test-secret", got)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
}

func TestTurnstileVerifierVerify(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    bool
		wantErr bool
	}{
		{name: "returns true when siteverify succeeds", status: 200, body: `{"success":true}`, want: true},
		{name: "returns false when siteverify rejects token", status: 200, body: `{"success":false,"error-codes":["invalid-input-response"]}`, want: false},
		{name: "returns error on malformed response", status: 200, body: `not-json`, wantErr: true},
		{name: "returns error on server error", status: 500, body: ``, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newSiteverifyServer(t, tt.status, tt.body)
			defer srv.Close()

			v := NewTurnstileVerifier("test-secret")
			v.endpoint = srv.URL

			got, err := v.Verify(t.Context(), "some-token")
			if tt.wantErr {
				if err == nil {
					t.Fatal("Verify() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProtectRegistration(t *testing.T) {
	type nextCall struct {
		called bool
		body   string
	}

	newNext := func(call *nextCall) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			call.called = true
			b, _ := io.ReadAll(r.Body)
			call.body = string(b)
			w.WriteHeader(http.StatusOK)
		})
	}

	t.Run("passes through when verifier is nil", func(t *testing.T) {
		var call nextCall
		h := ProtectRegistration(newNext(&call), nil)

		req := httptest.NewRequest(http.MethodPost, "/self-service/registration?flow=abc", strings.NewReader("traits.email=a%40b.c"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if !call.called {
			t.Fatal("next handler not called")
		}
	})

	t.Run("passes through non-registration paths", func(t *testing.T) {
		srv := newSiteverifyServer(t, 200, `{"success":false}`)
		defer srv.Close()
		v := NewTurnstileVerifier("test-secret")
		v.endpoint = srv.URL

		var call nextCall
		h := ProtectRegistration(newNext(&call), v)

		req := httptest.NewRequest(http.MethodPost, "/self-service/login?flow=abc", strings.NewReader("identifier=a%40b.c"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if !call.called {
			t.Fatal("next handler not called")
		}
	})

	t.Run("passes through GET requests", func(t *testing.T) {
		v := NewTurnstileVerifier("test-secret")

		var call nextCall
		h := ProtectRegistration(newNext(&call), v)

		req := httptest.NewRequest(http.MethodGet, "/self-service/registration?flow=abc", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if !call.called {
			t.Fatal("next handler not called")
		}
	})

	t.Run("blocks registration without token", func(t *testing.T) {
		srv := newSiteverifyServer(t, 200, `{"success":true}`)
		defer srv.Close()
		v := NewTurnstileVerifier("test-secret")
		v.endpoint = srv.URL

		var call nextCall
		h := ProtectRegistration(newNext(&call), v)

		req := httptest.NewRequest(http.MethodPost, "/self-service/registration?flow=abc", strings.NewReader("traits.email=a%40b.c"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if call.called {
			t.Fatal("next handler called, want blocked")
		}
		if rec.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if loc := rec.Header().Get("Location"); loc != "/registration" {
			t.Errorf("redirect location = %q, want /registration", loc)
		}
	})

	t.Run("blocks registration with invalid token", func(t *testing.T) {
		srv := newSiteverifyServer(t, 200, `{"success":false}`)
		defer srv.Close()
		v := NewTurnstileVerifier("test-secret")
		v.endpoint = srv.URL

		var call nextCall
		h := ProtectRegistration(newNext(&call), v)

		form := url.Values{"traits.email": {"a@b.c"}, "cf-turnstile-response": {"bad-token"}}
		req := httptest.NewRequest(http.MethodPost, "/self-service/registration?flow=abc", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if call.called {
			t.Fatal("next handler called, want blocked")
		}
		if rec.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
	})

	t.Run("forwards registration with valid token and intact body", func(t *testing.T) {
		srv := newSiteverifyServer(t, 200, `{"success":true}`)
		defer srv.Close()
		v := NewTurnstileVerifier("test-secret")
		v.endpoint = srv.URL

		var call nextCall
		h := ProtectRegistration(newNext(&call), v)

		form := url.Values{"traits.email": {"a@b.c"}, "cf-turnstile-response": {"good-token"}}
		encoded := form.Encode()
		req := httptest.NewRequest(http.MethodPost, "/self-service/registration?flow=abc", strings.NewReader(encoded))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if !call.called {
			t.Fatal("next handler not called")
		}
		if call.body != encoded {
			t.Errorf("forwarded body = %q, want %q", call.body, encoded)
		}
	})

	t.Run("fails closed when siteverify is unreachable", func(t *testing.T) {
		v := NewTurnstileVerifier("test-secret")
		v.endpoint = "http://127.0.0.1:1" // closed port

		var call nextCall
		h := ProtectRegistration(newNext(&call), v)

		form := url.Values{"cf-turnstile-response": {"token"}}
		req := httptest.NewRequest(http.MethodPost, "/self-service/registration?flow=abc", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if call.called {
			t.Fatal("next handler called, want blocked")
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})
}
