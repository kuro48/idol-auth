package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kuro48/idol-auth/internal/domain/app"
)

// AppWebhookRepository reads and writes the webhook_url / webhook_secret columns on apps.
type AppWebhookRepository struct {
	pool *pgxpool.Pool
}

func NewAppWebhookRepository(pool *pgxpool.Pool) *AppWebhookRepository {
	return &AppWebhookRepository{pool: pool}
}

func (r *AppWebhookRepository) GetConfig(ctx context.Context, appID uuid.UUID) (app.WebhookConfig, bool, error) {
	var cfg app.WebhookConfig
	cfg.AppID = appID

	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(webhook_url, ''), COALESCE(webhook_secret, '')
		FROM apps
		WHERE id = $1
	`, appID).Scan(&cfg.WebhookURL, &cfg.WebhookSecret)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app.WebhookConfig{}, false, nil
		}
		return app.WebhookConfig{}, false, fmt.Errorf("get app webhook config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return app.WebhookConfig{}, false, nil
	}
	return cfg, true, nil
}

func (r *AppWebhookRepository) SetConfig(ctx context.Context, appID uuid.UUID, webhookURL, webhookSecret string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE apps
		SET webhook_url = $2, webhook_secret = $3, updated_at = NOW()
		WHERE id = $1
	`, appID, webhookURL, webhookSecret)
	if err != nil {
		return fmt.Errorf("set app webhook config: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrAppNotFound
	}
	return nil
}
