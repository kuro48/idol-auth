package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReadinessChecker struct {
	pool *pgxpool.Pool
}

func NewReadinessChecker(pool *pgxpool.Pool) *ReadinessChecker {
	return &ReadinessChecker{pool: pool}
}

func (r *ReadinessChecker) Ready(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
