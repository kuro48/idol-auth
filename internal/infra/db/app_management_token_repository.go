package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
)

type AppManagementTokenRepository struct {
	pool *pgxpool.Pool
}

func NewAppManagementTokenRepository(pool *pgxpool.Pool) *AppManagementTokenRepository {
	return &AppManagementTokenRepository{pool: pool}
}

func (r *AppManagementTokenRepository) Replace(ctx context.Context, appID uuid.UUID, rawToken, actorID string, now time.Time) error {
	tokenHash := hashManagementToken(rawToken)
	prefix := rawToken
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin app management token transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE app_management_tokens
		SET status = 'rotated', updated_at = $2, updated_by = $3
		WHERE app_id = $1 AND status = 'active'
	`, appID, now, actorID); err != nil {
		return fmt.Errorf("rotate existing app management tokens: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO app_management_tokens (
			id, app_id, token_hash, token_prefix, status,
			created_at, updated_at, created_by, updated_by
		) VALUES ($1,$2,$3,$4,'active',$5,$5,$6,$6)
	`, uuid.New(), appID, tokenHash, prefix, now, actorID); err != nil {
		return fmt.Errorf("insert app management token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit app management token transaction: %w", err)
	}
	return nil
}

func (r *AppManagementTokenRepository) ResolveAppByToken(ctx context.Context, rawToken string) (app.App, error) {
	const query = `
		SELECT a.id, a.name, a.slug, a.type, a.party_type, a.status, a.description,
		       a.created_at, a.updated_at, a.created_by, a.updated_by
		FROM app_management_tokens t
		JOIN apps a ON a.id = t.app_id
		WHERE t.token_hash = $1 AND t.status = 'active'
	`
	var entity app.App
	if err := scanApp(r.pool.QueryRow(ctx, query, hashManagementToken(rawToken)), &entity); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app.App{}, app.ErrAppNotFound
		}
		return app.App{}, fmt.Errorf("resolve app by management token: %w", err)
	}
	return entity, nil
}

func hashManagementToken(rawToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	return hex.EncodeToString(sum[:])
}
