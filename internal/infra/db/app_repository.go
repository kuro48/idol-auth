package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
)

type AppRepository struct {
	pool *pgxpool.Pool
}

func NewAppRepository(pool *pgxpool.Pool) *AppRepository {
	return &AppRepository{pool: pool}
}

func (r *AppRepository) Create(ctx context.Context, entity app.App) (app.App, error) {
	const query = `
		INSERT INTO apps (
			id, name, slug, type, party_type, status, description,
			created_at, updated_at, created_by, updated_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, name, slug, type, party_type, status, description, created_at, updated_at, created_by, updated_by
	`

	row := r.pool.QueryRow(ctx, query,
		entity.ID, entity.Name, entity.Slug, entity.Type, entity.PartyType, entity.Status, entity.Description,
		entity.CreatedAt, entity.UpdatedAt, entity.CreatedBy, entity.UpdatedBy,
	)
	if err := scanApp(row, &entity); err != nil {
		return app.App{}, fmt.Errorf("insert app: %w", err)
	}
	return entity, nil
}

func (r *AppRepository) List(ctx context.Context) ([]app.App, error) {
	const query = `
		SELECT id, name, slug, type, party_type, status, description, created_at, updated_at, created_by, updated_by
		FROM apps
		ORDER BY created_at ASC, id ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}
	defer rows.Close()

	var out []app.App
	for rows.Next() {
		var entity app.App
		if err := scanApp(rows, &entity); err != nil {
			return nil, fmt.Errorf("scan app: %w", err)
		}
		out = append(out, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate apps: %w", err)
	}
	return out, nil
}

func (r *AppRepository) GetByID(ctx context.Context, id uuid.UUID) (app.App, error) {
	const query = `
		SELECT id, name, slug, type, party_type, status, description, created_at, updated_at, created_by, updated_by
		FROM apps
		WHERE id = $1
	`

	var entity app.App
	if err := scanApp(r.pool.QueryRow(ctx, query, id), &entity); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app.App{}, app.ErrAppNotFound
		}
		return app.App{}, fmt.Errorf("get app by id: %w", err)
	}
	return entity, nil
}

func scanApp(scanner interface{ Scan(...any) error }, entity *app.App) error {
	return scanner.Scan(
		&entity.ID,
		&entity.Name,
		&entity.Slug,
		&entity.Type,
		&entity.PartyType,
		&entity.Status,
		&entity.Description,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&entity.CreatedBy,
		&entity.UpdatedBy,
	)
}
