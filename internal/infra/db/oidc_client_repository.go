package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
)

type OIDCClientRepository struct {
	pool *pgxpool.Pool
}

func NewOIDCClientRepository(pool *pgxpool.Pool) *OIDCClientRepository {
	return &OIDCClientRepository{pool: pool}
}

func (r *OIDCClientRepository) Create(ctx context.Context, entity app.OIDCClient) (app.OIDCClient, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return app.OIDCClient{}, fmt.Errorf("begin oidc client transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertClient = `
		INSERT INTO oidc_clients (
			id, hydra_client_id, app_id, client_type, status, token_endpoint_auth_method,
			pkce_required, created_at, updated_at, created_by, updated_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	if _, err := tx.Exec(ctx, insertClient,
		entity.ID, entity.HydraClientID, entity.AppID, entity.ClientType, entity.Status, entity.TokenEndpointAuthMethod,
		entity.PKCERequired, entity.CreatedAt, entity.UpdatedAt, entity.CreatedBy, entity.UpdatedBy,
	); err != nil {
		return app.OIDCClient{}, fmt.Errorf("insert oidc client: %w", err)
	}

	const insertRedirect = `
		INSERT INTO oidc_client_redirect_uris (id, oidc_client_id, uri, kind)
		VALUES ($1,$2,$3,$4)
	`
	for _, uri := range entity.RedirectURIs {
		if _, err := tx.Exec(ctx, insertRedirect, uuid.New(), entity.ID, uri, "login_callback"); err != nil {
			return app.OIDCClient{}, fmt.Errorf("insert redirect uri: %w", err)
		}
	}
	for _, uri := range entity.PostLogoutRedirectURIs {
		if _, err := tx.Exec(ctx, insertRedirect, uuid.New(), entity.ID, uri, "post_logout_callback"); err != nil {
			return app.OIDCClient{}, fmt.Errorf("insert post logout redirect uri: %w", err)
		}
	}

	const insertScope = `
		INSERT INTO oidc_client_scopes (id, oidc_client_id, scope)
		VALUES ($1,$2,$3)
	`
	for _, scope := range entity.Scopes {
		if _, err := tx.Exec(ctx, insertScope, uuid.New(), entity.ID, scope); err != nil {
			return app.OIDCClient{}, fmt.Errorf("insert scope: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return app.OIDCClient{}, fmt.Errorf("commit oidc client transaction: %w", err)
	}

	return entity, nil
}

func (r *OIDCClientRepository) ListByAppID(ctx context.Context, appID uuid.UUID) ([]app.OIDCClient, error) {
	const query = `
		SELECT
			c.id,
			c.hydra_client_id,
			c.app_id,
			c.client_type,
			c.status,
			c.token_endpoint_auth_method,
			c.pkce_required,
			COALESCE(
				ARRAY(
					SELECT uri FROM oidc_client_redirect_uris
					WHERE oidc_client_id = c.id AND kind = 'login_callback'
					ORDER BY uri
				),
				ARRAY[]::text[]
			) AS redirect_uris,
			COALESCE(
				ARRAY(
					SELECT uri FROM oidc_client_redirect_uris
					WHERE oidc_client_id = c.id AND kind = 'post_logout_callback'
					ORDER BY uri
				),
				ARRAY[]::text[]
			) AS post_logout_redirect_uris,
			COALESCE(
				ARRAY(
					SELECT scope FROM oidc_client_scopes
					WHERE oidc_client_id = c.id
					ORDER BY scope
				),
				ARRAY[]::text[]
			) AS scopes,
			c.created_at,
			c.updated_at,
			c.created_by,
			c.updated_by
		FROM oidc_clients c
		WHERE c.app_id = $1
		ORDER BY c.created_at ASC, c.id ASC
	`

	rows, err := r.pool.Query(ctx, query, appID)
	if err != nil {
		return nil, fmt.Errorf("list oidc clients: %w", err)
	}
	defer rows.Close()

	var out []app.OIDCClient
	for rows.Next() {
		var entity app.OIDCClient
		if err := rows.Scan(
			&entity.ID,
			&entity.HydraClientID,
			&entity.AppID,
			&entity.ClientType,
			&entity.Status,
			&entity.TokenEndpointAuthMethod,
			&entity.PKCERequired,
			&entity.RedirectURIs,
			&entity.PostLogoutRedirectURIs,
			&entity.Scopes,
			&entity.CreatedAt,
			&entity.UpdatedAt,
			&entity.CreatedBy,
			&entity.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan oidc client: %w", err)
		}
		out = append(out, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oidc clients: %w", err)
	}
	return out, nil
}
