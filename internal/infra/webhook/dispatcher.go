// Package webhook delivers signed HTTP callbacks to app-registered endpoints.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/app"
)

// Event is the JSON payload sent to each registered webhook endpoint.
type Event struct {
	Type       string    `json:"type"`
	OccurredAt time.Time `json:"occurred_at"`
	AppID      string    `json:"app_id"`
	IdentityID string    `json:"identity_id,omitempty"`
}

// Dispatcher fetches an app's webhook config and fires events asynchronously.
type Dispatcher struct {
	repo   app.WebhookRepository
	client *http.Client
}

func NewDispatcher(repo app.WebhookRepository) *Dispatcher {
	return &Dispatcher{
		repo:   repo,
		client: newHTTPClient(),
	}
}

// DispatchAsync looks up the app's webhook URL and delivers the event in a goroutine.
// Errors are logged but never returned — the caller must not depend on delivery.
func (d *Dispatcher) DispatchAsync(ctx context.Context, appID uuid.UUID, eventType, identityID string) {
	go func() {
		bCtx := context.Background()
		cfg, ok, err := d.repo.GetConfig(bCtx, appID)
		if err != nil {
			slog.Error("webhook: failed to fetch config", "app_id", appID, "error", err)
			return
		}
		if !ok {
			return
		}

		event := Event{
			Type:       eventType,
			OccurredAt: time.Now().UTC(),
			AppID:      appID.String(),
			IdentityID: identityID,
		}
		if err := d.send(bCtx, cfg.WebhookURL, cfg.WebhookSecret, event); err != nil {
			slog.Warn("webhook: delivery failed", "app_id", appID, "event", eventType, "error", err)
		}
	}()
}

func (d *Dispatcher) send(ctx context.Context, webhookURL, secret string, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event.Type)
	if secret != "" {
		req.Header.Set("X-Webhook-Signature", sign(payload, secret))
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("endpoint returned %d", resp.StatusCode)
	}
	return nil
}

func sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// GenerateSecret returns a random 32-byte hex secret suitable for webhook signing.
func GenerateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate webhook secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// ValidateURL returns an error if s is not a valid public HTTPS webhook URL.
func ValidateURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("webhook_url is required")
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return fmt.Errorf("webhook_url must be a valid URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("webhook_url must use https")
	}
	if u.User != nil {
		return fmt.Errorf("webhook_url must not include credentials")
	}
	host := strings.TrimSpace(u.Hostname())
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("webhook_url host is not allowed")
	}
	if forbidden, err := isForbiddenWebhookIP(host); err == nil && forbidden {
		return fmt.Errorf("webhook_url host is not allowed")
	}
	return nil
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	resolver := net.DefaultResolver
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addrs, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve webhook target: %w", err)
		}
		var publicAddr netip.Addr
		for _, resolved := range addrs {
			addr, ok := netip.AddrFromSlice(resolved.IP)
			if !ok {
				continue
			}
			addr = addr.Unmap()
			if isForbiddenWebhookAddr(addr) {
				return nil, fmt.Errorf("webhook target resolves to forbidden address")
			}
			if !publicAddr.IsValid() {
				publicAddr = addr
			}
		}
		if !publicAddr.IsValid() {
			return nil, fmt.Errorf("webhook target did not resolve to a usable address")
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(publicAddr.String(), port))
	}
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if err := ValidateURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect target rejected: %w", err)
			}
			return nil
		},
	}
}

func isForbiddenWebhookIP(raw string) (bool, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return false, err
	}
	return isForbiddenWebhookAddr(addr.Unmap()), nil
}

func isForbiddenWebhookAddr(addr netip.Addr) bool {
	if !addr.IsValid() {
		return true
	}
	if !addr.IsGlobalUnicast() {
		return true
	}
	if addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return true
	}
	for _, prefix := range forbiddenWebhookPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

var forbiddenWebhookPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
}
