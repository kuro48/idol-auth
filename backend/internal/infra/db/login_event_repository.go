package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kuro48/idol-auth/internal/domain/loginhistory"
)

type LoginEventRepository struct {
	pool *pgxpool.Pool
}

func NewLoginEventRepository(pool *pgxpool.Pool) *LoginEventRepository {
	return &LoginEventRepository{pool: pool}
}

func (r *LoginEventRepository) Record(ctx context.Context, evt loginhistory.Event) error {
	const query = `
		INSERT INTO login_events (
			identity_id, session_id, authenticated_at, issued_at, aal, methods,
			ip_address, user_agent, recorded_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (session_id) DO NOTHING
	`
	if _, err := r.pool.Exec(ctx, query,
		evt.IdentityID,
		evt.SessionID,
		evt.AuthenticatedAt,
		evt.IssuedAt,
		evt.AAL,
		evt.Methods,
		nullableString(evt.IPAddress),
		nullableString(evt.UserAgent),
		evt.RecordedAt,
	); err != nil {
		return fmt.Errorf("insert login event: %w", err)
	}
	return nil
}

func (r *LoginEventRepository) ListByIdentity(ctx context.Context, identityID string, limit int) ([]loginhistory.Event, error) {
	const query = `
		SELECT id, identity_id, session_id, authenticated_at, issued_at, aal, methods,
			COALESCE(ip_address::text, ''), COALESCE(user_agent, ''), recorded_at
		FROM login_events
		WHERE identity_id = $1
		ORDER BY authenticated_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, identityID, limit)
	if err != nil {
		return nil, fmt.Errorf("list login events: %w", err)
	}
	defer rows.Close()

	out := make([]loginhistory.Event, 0, limit)
	for rows.Next() {
		var evt loginhistory.Event
		if err := rows.Scan(
			&evt.ID,
			&evt.IdentityID,
			&evt.SessionID,
			&evt.AuthenticatedAt,
			&evt.IssuedAt,
			&evt.AAL,
			&evt.Methods,
			&evt.IPAddress,
			&evt.UserAgent,
			&evt.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan login event: %w", err)
		}
		if evt.Methods == nil {
			evt.Methods = []string{}
		}
		out = append(out, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate login events: %w", err)
	}
	return out, nil
}
