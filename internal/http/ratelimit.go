package http

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// RateLimiter decides whether a request identified by key should be allowed.
type RateLimiter interface {
	Allow(key string) bool
}

// NewInMemoryRateLimiter returns a per-key sliding-window rate limiter backed
// by an in-memory map. It is safe for concurrent use but does not share state
// across multiple process instances or survive process restarts.
//
// For production deployments with multiple replicas, replace this with a
// Redis-backed implementation using atomic INCR+EXPIRE to share state across
// instances. The RateLimiter interface is designed to make this substitution
// straightforward — pass the alternative via RouterConfig.Limiter.
func NewInMemoryRateLimiter(limit int, window time.Duration) RateLimiter {
	return newSlidingWindowLimiter(limit, window, time.Now)
}

func newSlidingWindowLimiter(limit int, window time.Duration, clock func() time.Time) *slidingWindowLimiter {
	return &slidingWindowLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		clock:    clock,
	}
}

// maxRateLimitKeys caps the number of distinct client IPs tracked to bound
// memory usage. When the map is full, expired entries are evicted first; if
// it is still full for a completely new key, that request is denied.
const maxRateLimitKeys = 10_000

type slidingWindowLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	clock    func() time.Time
}

func (l *slidingWindowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock()
	cutoff := now.Add(-l.window)

	existing := l.requests[key]
	active := make([]time.Time, 0, len(existing))
	for _, t := range existing {
		if t.After(cutoff) {
			active = append(active, t)
		}
	}

	if len(active) >= l.limit {
		l.requests[key] = active
		return false
	}

	_, known := l.requests[key]
	if !known && len(l.requests) >= maxRateLimitKeys {
		l.evictExpired(cutoff)
		if len(l.requests) >= maxRateLimitKeys {
			return false
		}
	}

	l.requests[key] = append(active, now)
	return true
}

// evictExpired removes all map entries whose timestamps are entirely outside
// the current window. Must be called with l.mu held.
func (l *slidingWindowLimiter) evictExpired(cutoff time.Time) {
	for k, timestamps := range l.requests {
		alive := false
		for _, t := range timestamps {
			if t.After(cutoff) {
				alive = true
				break
			}
		}
		if !alive {
			delete(l.requests, k)
		}
	}
}

// rateLimitMiddleware rejects requests whose client IP has exceeded the limit.
func rateLimitMiddleware(limiter RateLimiter, trustedProxies []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r, trustedProxies)
			if !limiter.Allow(key) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP returns the best-effort real client IP, preferring X-Real-IP over
// X-Forwarded-For over RemoteAddr. Header values are validated with
// netip.ParseAddr; malformed values are skipped.
func clientIP(r *http.Request, trustedProxies []string) string {
	remoteIP := remoteAddrIP(r.RemoteAddr)
	if !requestViaTrustedProxy(r, trustedProxies) {
		return remoteIP
	}
	if raw := strings.TrimSpace(r.Header.Get("X-Real-IP")); raw != "" {
		if addr, err := netip.ParseAddr(raw); err == nil {
			return addr.String()
		}
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first := forwarded
		if idx := strings.Index(forwarded, ","); idx != -1 {
			first = strings.TrimSpace(forwarded[:idx])
		}
		if addr, err := netip.ParseAddr(first); err == nil {
			return addr.String()
		}
	}
	return remoteIP
}

func requestViaTrustedProxy(r *http.Request, trustedProxies []string) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	remoteIP := remoteAddrIP(r.RemoteAddr)
	addr, err := netip.ParseAddr(remoteIP)
	if err != nil {
		return false
	}
	for _, candidate := range trustedProxies {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(candidate); err == nil && prefix.Contains(addr) {
			return true
		}
		if proxyIP, err := netip.ParseAddr(candidate); err == nil && proxyIP == addr {
			return true
		}
	}
	return false
}

func remoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
