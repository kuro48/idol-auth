package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Middleware behaviour
// ---------------------------------------------------------------------------

func TestRateLimitMiddlewareAllowsRequestUnderLimit(t *testing.T) {
	stub := &stubRateLimiter{allow: true}
	handler := rateLimitMiddleware(stub, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRateLimitMiddlewareReturns429WhenLimitExceeded(t *testing.T) {
	stub := &stubRateLimiter{allow: false}
	handler := rateLimitMiddleware(stub, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestRateLimitMiddlewareUsesRemoteAddrAsKey(t *testing.T) {
	stub := &stubRateLimiter{allow: true}
	handler := rateLimitMiddleware(stub, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if stub.lastKey != "203.0.113.5" {
		t.Fatalf("expected key %q, got %q", "203.0.113.5", stub.lastKey)
	}
}

func TestRateLimitMiddlewareUsesXRealIPWhenPresent(t *testing.T) {
	stub := &stubRateLimiter{allow: true}
	handler := rateLimitMiddleware(stub, []string{"10.0.0.1/32"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "203.0.113.99")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if stub.lastKey != "203.0.113.99" {
		t.Fatalf("expected X-Real-IP key %q, got %q", "203.0.113.99", stub.lastKey)
	}
}

// ---------------------------------------------------------------------------
// Sliding window limiter
// ---------------------------------------------------------------------------

func TestInMemoryRateLimiterAllowsUnderLimit(t *testing.T) {
	now := time.Unix(1000, 0)
	limiter := newSlidingWindowLimiter(3, time.Minute, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if !limiter.Allow("ip1") {
			t.Fatalf("request %d should be allowed (limit=3)", i+1)
		}
	}
}

func TestInMemoryRateLimiterBlocksAtLimit(t *testing.T) {
	now := time.Unix(1000, 0)
	limiter := newSlidingWindowLimiter(2, time.Minute, func() time.Time { return now })

	limiter.Allow("ip1")
	limiter.Allow("ip1")

	if limiter.Allow("ip1") {
		t.Fatal("third request should be blocked when limit=2")
	}
}

func TestInMemoryRateLimiterKeysAreIndependent(t *testing.T) {
	now := time.Unix(1000, 0)
	limiter := newSlidingWindowLimiter(1, time.Minute, func() time.Time { return now })

	limiter.Allow("ip1") // exhausts ip1's limit

	if !limiter.Allow("ip2") {
		t.Fatal("ip2 should have an independent counter")
	}
}

func TestInMemoryRateLimiterResetsAfterWindowExpires(t *testing.T) {
	current := time.Unix(1000, 0)
	limiter := newSlidingWindowLimiter(1, time.Minute, func() time.Time { return current })

	limiter.Allow("ip1") // use up limit at t=1000
	if limiter.Allow("ip1") {
		t.Fatal("second request at same time should be blocked")
	}

	current = current.Add(61 * time.Second) // advance past the 1-minute window

	if !limiter.Allow("ip1") {
		t.Fatal("request after window expiry should be allowed")
	}
}

func TestInMemoryRateLimiterCountsOnlyWindowRequests(t *testing.T) {
	current := time.Unix(1000, 0)
	limiter := newSlidingWindowLimiter(2, time.Minute, func() time.Time { return current })

	limiter.Allow("ip1")                    // t=1000
	current = current.Add(30 * time.Second) // t=1030
	limiter.Allow("ip1")                    // t=1030
	if limiter.Allow("ip1") {
		t.Fatal("third request within window should be blocked")
	}

	current = current.Add(31 * time.Second) // t=1061; t=1000 entry expired
	if !limiter.Allow("ip1") {
		t.Fatal("request should be allowed after first entry expires from window")
	}
}

func TestRateLimitMiddlewareUsesFirstXForwardedForIP(t *testing.T) {
	stub := &stubRateLimiter{allow: true}
	handler := rateLimitMiddleware(stub, []string{"10.0.0.1/32"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.2, 10.0.0.3")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if stub.lastKey != "198.51.100.1" {
		t.Fatalf("expected first X-Forwarded-For IP %q, got %q", "198.51.100.1", stub.lastKey)
	}
}

func TestRateLimitMiddlewareUsesSingleXForwardedForIP(t *testing.T) {
	stub := &stubRateLimiter{allow: true}
	handler := rateLimitMiddleware(stub, []string{"10.0.0.1/32"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.5")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if stub.lastKey != "198.51.100.5" {
		t.Fatalf("expected single X-Forwarded-For IP %q, got %q", "198.51.100.5", stub.lastKey)
	}
}

func TestNewInMemoryRateLimiterAllowsAndBlocks(t *testing.T) {
	limiter := NewInMemoryRateLimiter(2, time.Minute)

	if !limiter.Allow("ip1") {
		t.Fatal("first request should be allowed")
	}
	if !limiter.Allow("ip1") {
		t.Fatal("second request should be allowed")
	}
	if limiter.Allow("ip1") {
		t.Fatal("third request should be blocked (limit=2)")
	}
}

func TestRateLimitMiddlewareIgnoresForwardedHeadersFromUntrustedRemote(t *testing.T) {
	stub := &stubRateLimiter{allow: true}
	handler := rateLimitMiddleware(stub, []string{"10.0.0.1/32"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Real-IP", "198.51.100.99")
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if stub.lastKey != "203.0.113.5" {
		t.Fatalf("expected untrusted request to use remote addr, got %q", stub.lastKey)
	}
}

func TestRequestIsSecureUsesTrustedProxyHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-Proto", "https")

	if !requestIsSecure(req, []string{"10.0.0.1/32"}) {
		t.Fatal("expected trusted proxy https header to be accepted")
	}
}

func TestRequestIsSecureIgnoresUntrustedProxyHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-Proto", "https")

	if requestIsSecure(req, []string{"10.0.0.1/32"}) {
		t.Fatal("expected untrusted proxy https header to be ignored")
	}
}

// ---------------------------------------------------------------------------
// Stubs
// ---------------------------------------------------------------------------

type stubRateLimiter struct {
	allow   bool
	lastKey string
}

func (s *stubRateLimiter) Allow(key string) bool {
	s.lastKey = key
	return s.allow
}
