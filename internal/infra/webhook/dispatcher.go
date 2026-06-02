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
	"net/http"
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
		client: &http.Client{Timeout: 10 * time.Second},
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

// ValidateURL returns an error if s is not a valid https webhook URL.
func ValidateURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("webhook_url is required")
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return fmt.Errorf("webhook_url must be a valid URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("webhook_url must use https")
	}
	return nil
}
