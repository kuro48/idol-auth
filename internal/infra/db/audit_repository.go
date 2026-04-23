package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Write(ctx context.Context, entry audit.Log) error {
	const query = `
		INSERT INTO audit_logs (
			id, event_type, actor_type, actor_id, target_type, target_id,
			result, ip_address, user_agent, request_id, metadata, occurred_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	if _, err := r.pool.Exec(ctx, query,
		entry.ID, entry.EventType, entry.ActorType, entry.ActorID, entry.TargetType, entry.TargetID,
		entry.Result, nullableString(entry.IPAddress), nullableString(entry.UserAgent), nullableString(entry.RequestID), metadataOrEmptyObject(entry.Metadata), entry.OccurredAt,
	); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *AuditRepository) List(ctx context.Context, params audit.ListParams) ([]audit.Log, error) {
	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var (
		sb   strings.Builder
		args []any
	)
	sb.WriteString(`
		SELECT id, event_type, actor_type, actor_id, target_type, target_id, result, COALESCE(ip_address::text, ''), COALESCE(user_agent, ''), COALESCE(request_id, ''), metadata, occurred_at
		FROM audit_logs
		WHERE 1=1
	`)
	appendFilter := func(column, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		args = append(args, strings.TrimSpace(value))
		fmt.Fprintf(&sb, " AND %s = $%d", column, len(args))
	}
	appendFilter("actor_type", params.ActorType)
	appendFilter("actor_id", params.ActorID)
	appendFilter("target_type", params.TargetType)
	appendFilter("target_id", params.TargetID)
	appendFilter("event_type", params.EventType)

	args = append(args, limit, params.Offset)
	fmt.Fprintf(&sb, " ORDER BY occurred_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var out []audit.Log
	for rows.Next() {
		var entry audit.Log
		if err := rows.Scan(
			&entry.ID,
			&entry.EventType,
			&entry.ActorType,
			&entry.ActorID,
			&entry.TargetType,
			&entry.TargetID,
			&entry.Result,
			&entry.IPAddress,
			&entry.UserAgent,
			&entry.RequestID,
			&entry.Metadata,
			&entry.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}
	return out, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func metadataOrEmptyObject(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}
