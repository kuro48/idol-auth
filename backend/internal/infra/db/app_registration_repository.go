package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

type AppRegistrationRepository struct {
	pool *pgxpool.Pool
}

func NewAppRegistrationRepository(pool *pgxpool.Pool) *AppRegistrationRepository {
	return &AppRegistrationRepository{pool: pool}
}

func (r *AppRegistrationRepository) Create(ctx context.Context, req appreg.Request) (appreg.Request, error) {
	const query = `
		INSERT INTO app_registration_requests (
			id, identity_id, status, name, slug, type, description,
			homepage_url, privacy_policy_url, terms_url, contact_email, organization, purpose,
			redirect_uris, post_logout_redirect_uris, scopes,
			version, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
		)
		RETURNING id, identity_id, status, name, slug, type, description,
			homepage_url, privacy_policy_url, terms_url, contact_email, organization, purpose,
			redirect_uris, post_logout_redirect_uris, scopes,
			reviewer_id, reviewer_note, decided_at, created_app_id, created_client_id,
			version, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query,
		req.ID, req.IdentityID, string(req.Status), req.Name, req.Slug, req.Type, req.Description,
		req.HomepageURL, req.PrivacyPolicyURL, req.TermsURL, req.ContactEmail, req.Organization, req.Purpose,
		req.RedirectURIs, req.PostLogoutRedirectURIs, req.Scopes,
		req.Version, req.CreatedAt, req.UpdatedAt,
	)
	result, err := scanRequest(row)
	if err != nil {
		return appreg.Request{}, fmt.Errorf("create app registration request: %w", err)
	}
	return result, nil
}

func (r *AppRegistrationRepository) Update(ctx context.Context, req appreg.Request) (appreg.Request, error) {
	const query = `
		UPDATE app_registration_requests SET
			status                    = $1,
			name                      = $2,
			slug                      = $3,
			type                      = $4,
			description               = $5,
			homepage_url              = $6,
			privacy_policy_url        = $7,
			terms_url                 = $8,
			contact_email             = $9,
			organization              = $10,
			purpose                   = $11,
			redirect_uris             = $12,
			post_logout_redirect_uris = $13,
			scopes                    = $14,
			reviewer_id               = $15,
			reviewer_note             = $16,
			decided_at                = $17,
			created_app_id            = $18,
			created_client_id         = $19,
			version                   = $20,
			updated_at                = $21
		WHERE id = $22 AND version = $23
		RETURNING id, identity_id, status, name, slug, type, description,
			homepage_url, privacy_policy_url, terms_url, contact_email, organization, purpose,
			redirect_uris, post_logout_redirect_uris, scopes,
			reviewer_id, reviewer_note, decided_at, created_app_id, created_client_id,
			version, created_at, updated_at
	`
	prevVersion := req.Version - 1
	row := r.pool.QueryRow(ctx, query,
		string(req.Status), req.Name, req.Slug, req.Type, req.Description,
		req.HomepageURL, req.PrivacyPolicyURL, req.TermsURL, req.ContactEmail, req.Organization, req.Purpose,
		req.RedirectURIs, req.PostLogoutRedirectURIs, req.Scopes,
		nullableString(req.ReviewerID), nullableString(req.ReviewerNote),
		nullableTime(req.DecidedAt), req.CreatedAppID, req.CreatedClientID,
		req.Version, req.UpdatedAt,
		req.ID, prevVersion,
	)
	result, err := scanRequest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appreg.Request{}, appreg.ErrAlreadyDecided
		}
		return appreg.Request{}, fmt.Errorf("update app registration request: %w", err)
	}
	return result, nil
}

func (r *AppRegistrationRepository) GetByID(ctx context.Context, id uuid.UUID) (appreg.Request, error) {
	const query = `
		SELECT id, identity_id, status, name, slug, type, description,
			homepage_url, privacy_policy_url, terms_url, contact_email, organization, purpose,
			redirect_uris, post_logout_redirect_uris, scopes,
			reviewer_id, reviewer_note, decided_at, created_app_id, created_client_id,
			version, created_at, updated_at
		FROM app_registration_requests
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	result, err := scanRequest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appreg.Request{}, appreg.ErrRequestNotFound
		}
		return appreg.Request{}, fmt.Errorf("get app registration request: %w", err)
	}
	return result, nil
}

func (r *AppRegistrationRepository) ListByIdentity(ctx context.Context, identityID string) ([]appreg.Request, error) {
	const query = `
		SELECT id, identity_id, status, name, slug, type, description,
			homepage_url, privacy_policy_url, terms_url, contact_email, organization, purpose,
			redirect_uris, post_logout_redirect_uris, scopes,
			reviewer_id, reviewer_note, decided_at, created_app_id, created_client_id,
			version, created_at, updated_at
		FROM app_registration_requests
		WHERE identity_id = $1
		ORDER BY created_at DESC
	`
	return r.queryList(ctx, query, identityID)
}

func (r *AppRegistrationRepository) ListForAdmin(ctx context.Context, statusFilter string) ([]appreg.Request, error) {
	if statusFilter != "" {
		const query = `
			SELECT id, identity_id, status, name, slug, type, description,
				homepage_url, privacy_policy_url, terms_url, contact_email, organization, purpose,
				redirect_uris, post_logout_redirect_uris, scopes,
				reviewer_id, reviewer_note, decided_at, created_app_id, created_client_id,
				version, created_at, updated_at
			FROM app_registration_requests
			WHERE status = $1
			ORDER BY created_at ASC
		`
		return r.queryList(ctx, query, statusFilter)
	}
	const query = `
		SELECT id, identity_id, status, name, slug, type, description,
			homepage_url, privacy_policy_url, terms_url, contact_email, organization, purpose,
			redirect_uris, post_logout_redirect_uris, scopes,
			reviewer_id, reviewer_note, decided_at, created_app_id, created_client_id,
			version, created_at, updated_at
		FROM app_registration_requests
		ORDER BY created_at ASC
	`
	return r.queryList(ctx, query)
}

func (r *AppRegistrationRepository) HasPendingByNameAndIdentity(ctx context.Context, name, identityID string, excludeID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM app_registration_requests
			WHERE identity_id = $1
			  AND name = $2
			  AND status IN ('pending','under_review','changes_requested')
			  AND id != $3
		)
	`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, identityID, name, excludeID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check pending duplicate: %w", err)
	}
	return exists, nil
}

func (r *AppRegistrationRepository) AppendEvent(ctx context.Context, event appreg.Event) error {
	const query = `
		INSERT INTO app_registration_events (id, request_id, actor_type, actor_id, event_type, note, occurred_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`
	if _, err := r.pool.Exec(ctx, query,
		event.ID, event.RequestID, event.ActorType, event.ActorID, event.EventType,
		nullableString(event.Note), event.OccurredAt,
	); err != nil {
		return fmt.Errorf("append registration event: %w", err)
	}
	return nil
}

func (r *AppRegistrationRepository) queryList(ctx context.Context, query string, args ...any) ([]appreg.Request, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list app registration requests: %w", err)
	}
	defer rows.Close()

	var out []appreg.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan app registration request: %w", err)
		}
		out = append(out, req)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate app registration requests: %w", err)
	}
	return out, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRequest(s scanner) (appreg.Request, error) {
	var (
		req          appreg.Request
		status       string
		reviewerID   *string
		reviewerNote *string
		decidedAt    *time.Time
		createdAppID *uuid.UUID
		createdClient *uuid.UUID
	)
	if err := s.Scan(
		&req.ID, &req.IdentityID, &status, &req.Name, &req.Slug, &req.Type, &req.Description,
		&req.HomepageURL, &req.PrivacyPolicyURL, &req.TermsURL, &req.ContactEmail, &req.Organization, &req.Purpose,
		&req.RedirectURIs, &req.PostLogoutRedirectURIs, &req.Scopes,
		&reviewerID, &reviewerNote, &decidedAt, &createdAppID, &createdClient,
		&req.Version, &req.CreatedAt, &req.UpdatedAt,
	); err != nil {
		return appreg.Request{}, err
	}
	req.Status = appreg.Status(status)
	if reviewerID != nil {
		req.ReviewerID = *reviewerID
	}
	if reviewerNote != nil {
		req.ReviewerNote = *reviewerNote
	}
	req.DecidedAt = decidedAt
	req.CreatedAppID = createdAppID
	req.CreatedClientID = createdClient
	return req, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
