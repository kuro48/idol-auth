package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/account"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Upsert(ctx context.Context, membership account.AppMembership) (account.AppMembership, error) {
	const query = `
		INSERT INTO app_user_memberships (
			id, app_id, identity_id, status, profile, created_at, updated_at, created_by, updated_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (app_id, identity_id)
		DO UPDATE SET
			status = EXCLUDED.status,
			profile = EXCLUDED.profile,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
		RETURNING id, app_id, identity_id, status, profile, created_at, updated_at, created_by, updated_by
	`
	var profile []byte
	if len(membership.Profile) > 0 {
		profile = membership.Profile
	} else {
		profile = []byte(`{}`)
	}
	if err := r.pool.QueryRow(ctx, query,
		membership.ID, membership.AppID, membership.IdentityID, membership.Status, profile,
		membership.CreatedAt, membership.UpdatedAt, membership.CreatedBy, membership.UpdatedBy,
	).Scan(
		&membership.ID,
		&membership.AppID,
		&membership.IdentityID,
		&membership.Status,
		&membership.Profile,
		&membership.CreatedAt,
		&membership.UpdatedAt,
		&membership.CreatedBy,
		&membership.UpdatedBy,
	); err != nil {
		return account.AppMembership{}, fmt.Errorf("upsert app membership: %w", err)
	}
	appDetails, err := r.getAppDetails(ctx, membership.AppID)
	if err != nil {
		return account.AppMembership{}, err
	}
	membership.AppName = appDetails.Name
	membership.AppSlug = appDetails.Slug
	membership.PartyType = appDetails.PartyType
	return membership, nil
}

func (r *AccountRepository) ListByIdentity(ctx context.Context, identityID string) ([]account.AppMembership, error) {
	return r.list(ctx, `
		SELECT m.id, m.app_id, a.slug, a.name, a.party_type, m.identity_id, m.status, m.profile, m.created_at, m.updated_at, m.created_by, m.updated_by
		FROM app_user_memberships m
		JOIN apps a ON a.id = m.app_id
		WHERE m.identity_id = $1
		ORDER BY a.created_at ASC, a.id ASC
	`, strings.TrimSpace(identityID))
}

func (r *AccountRepository) ListByAppID(ctx context.Context, appID uuid.UUID) ([]account.AppMembership, error) {
	return r.list(ctx, `
		SELECT m.id, m.app_id, a.slug, a.name, a.party_type, m.identity_id, m.status, m.profile, m.created_at, m.updated_at, m.created_by, m.updated_by
		FROM app_user_memberships m
		JOIN apps a ON a.id = m.app_id
		WHERE m.app_id = $1
		ORDER BY m.created_at ASC, m.id ASC
	`, appID)
}

func (r *AccountRepository) list(ctx context.Context, query string, arg any) ([]account.AppMembership, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, fmt.Errorf("list app memberships: %w", err)
	}
	defer rows.Close()
	var out []account.AppMembership
	for rows.Next() {
		var membership account.AppMembership
		if err := rows.Scan(
			&membership.ID,
			&membership.AppID,
			&membership.AppSlug,
			&membership.AppName,
			&membership.PartyType,
			&membership.IdentityID,
			&membership.Status,
			&membership.Profile,
			&membership.CreatedAt,
			&membership.UpdatedAt,
			&membership.CreatedBy,
			&membership.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan app membership: %w", err)
		}
		out = append(out, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate app memberships: %w", err)
	}
	return out, nil
}

func (r *AccountRepository) UpdateStatus(ctx context.Context, appID uuid.UUID, identityID string, status account.MembershipStatus, actorID string, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE app_user_memberships
		SET status = $3, updated_at = $4, updated_by = $5
		WHERE app_id = $1 AND identity_id = $2
	`, appID, strings.TrimSpace(identityID), status, now, actorID)
	if err != nil {
		return fmt.Errorf("update app membership status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrAppNotFound
	}
	return nil
}

func (r *AccountRepository) UpdateStatusByIdentity(ctx context.Context, identityID string, status account.MembershipStatus, actorID string, now time.Time) error {
	if _, err := r.pool.Exec(ctx, `
		UPDATE app_user_memberships
		SET status = $2, updated_at = $3, updated_by = $4
		WHERE identity_id = $1
	`, strings.TrimSpace(identityID), status, now, actorID); err != nil {
		return fmt.Errorf("update app membership statuses by identity: %w", err)
	}
	return nil
}

func (r *AccountRepository) UpsertScheduled(ctx context.Context, request account.DeletionRequest) (account.DeletionRequest, error) {
	const query = `
		INSERT INTO account_deletion_requests (
			id, identity_id, status, reason, requested_at, scheduled_for, cancelled_at, completed_at, last_actor_id
		) VALUES ($1,$2,$3,$4,$5,$6,NULL,NULL,$7)
		ON CONFLICT (identity_id)
		DO UPDATE SET
			id = EXCLUDED.id,
			status = EXCLUDED.status,
			reason = EXCLUDED.reason,
			requested_at = EXCLUDED.requested_at,
			scheduled_for = EXCLUDED.scheduled_for,
			cancelled_at = NULL,
			completed_at = NULL,
			last_actor_id = EXCLUDED.last_actor_id
		RETURNING id, identity_id, status, reason, requested_at, scheduled_for, cancelled_at, completed_at, last_actor_id
	`
	var cancelledAt, completedAt *time.Time
	if err := r.pool.QueryRow(ctx, query,
		request.ID, request.IdentityID, request.Status, request.Reason, request.RequestedAt, request.ScheduledFor, request.LastActorID,
	).Scan(
		&request.ID,
		&request.IdentityID,
		&request.Status,
		&request.Reason,
		&request.RequestedAt,
		&request.ScheduledFor,
		&cancelledAt,
		&completedAt,
		&request.LastActorID,
	); err != nil {
		return account.DeletionRequest{}, fmt.Errorf("upsert account deletion request: %w", err)
	}
	request.CancelledAt = cancelledAt
	request.CompletedAt = completedAt
	return request, nil
}

func (r *AccountRepository) GetByIdentity(ctx context.Context, identityID string) (account.DeletionRequest, error) {
	var request account.DeletionRequest
	var cancelledAt, completedAt *time.Time
	if err := r.pool.QueryRow(ctx, `
		SELECT id, identity_id, status, reason, requested_at, scheduled_for, cancelled_at, completed_at, COALESCE(last_actor_id, '')
		FROM account_deletion_requests
		WHERE identity_id = $1
	`, strings.TrimSpace(identityID)).Scan(
		&request.ID,
		&request.IdentityID,
		&request.Status,
		&request.Reason,
		&request.RequestedAt,
		&request.ScheduledFor,
		&cancelledAt,
		&completedAt,
		&request.LastActorID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return account.DeletionRequest{}, account.ErrDeletionRequestNotFound
		}
		return account.DeletionRequest{}, fmt.Errorf("get account deletion request: %w", err)
	}
	request.CancelledAt = cancelledAt
	request.CompletedAt = completedAt
	return request, nil
}

func (r *AccountRepository) Cancel(ctx context.Context, identityID, actorID string, cancelledAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE account_deletion_requests
		SET status = 'cancelled', cancelled_at = $2, last_actor_id = $3
		WHERE identity_id = $1 AND status = 'scheduled'
	`, strings.TrimSpace(identityID), cancelledAt, actorID)
	if err != nil {
		return fmt.Errorf("cancel account deletion request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return account.ErrDeletionRequestNotFound
	}
	return nil
}

func (r *AccountRepository) ListDue(ctx context.Context, dueBefore time.Time, limit int) ([]account.DeletionRequest, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, identity_id, status, reason, requested_at, scheduled_for, cancelled_at, completed_at, COALESCE(last_actor_id, '')
		FROM account_deletion_requests
		WHERE status = 'scheduled' AND scheduled_for <= $1
		ORDER BY scheduled_for ASC
		LIMIT $2
	`, dueBefore, limit)
	if err != nil {
		return nil, fmt.Errorf("list due account deletion requests: %w", err)
	}
	defer rows.Close()
	var out []account.DeletionRequest
	for rows.Next() {
		var request account.DeletionRequest
		var cancelledAt, completedAt *time.Time
		if err := rows.Scan(
			&request.ID,
			&request.IdentityID,
			&request.Status,
			&request.Reason,
			&request.RequestedAt,
			&request.ScheduledFor,
			&cancelledAt,
			&completedAt,
			&request.LastActorID,
		); err != nil {
			return nil, fmt.Errorf("scan due account deletion request: %w", err)
		}
		request.CancelledAt = cancelledAt
		request.CompletedAt = completedAt
		out = append(out, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due account deletion requests: %w", err)
	}
	return out, nil
}

func (r *AccountRepository) MarkCompleted(ctx context.Context, identityID, actorID string, completedAt time.Time) error {
	if _, err := r.pool.Exec(ctx, `
		UPDATE account_deletion_requests
		SET status = 'completed', completed_at = $2, last_actor_id = $3
		WHERE identity_id = $1
	`, strings.TrimSpace(identityID), completedAt, actorID); err != nil {
		return fmt.Errorf("mark account deletion request completed: %w", err)
	}
	return nil
}

func (r *AccountRepository) getAppDetails(ctx context.Context, appID uuid.UUID) (app.App, error) {
	const query = `
		SELECT id, name, slug, type, party_type, status, description, created_at, updated_at, created_by, updated_by
		FROM apps
		WHERE id = $1
	`
	var entity app.App
	if err := scanApp(r.pool.QueryRow(ctx, query, appID), &entity); err != nil {
		return app.App{}, fmt.Errorf("get app details: %w", err)
	}
	return entity, nil
}

func (*AccountRepository) normalizeProfile(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}
