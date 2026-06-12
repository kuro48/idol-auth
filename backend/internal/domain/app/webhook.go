package app

import (
	"context"

	"github.com/google/uuid"
)

// WebhookConfig holds the delivery endpoint and signing secret for an app's webhook.
type WebhookConfig struct {
	AppID         uuid.UUID
	WebhookURL    string
	WebhookSecret string
}

// WebhookRepository manages the webhook configuration stored alongside each app.
type WebhookRepository interface {
	GetConfig(ctx context.Context, appID uuid.UUID) (WebhookConfig, bool, error)
	SetConfig(ctx context.Context, appID uuid.UUID, webhookURL, webhookSecret string) error
}
