package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// poolMaxConns caps connections at a level suitable for a single-instance
	// personal auth server on a small VPS. Raise if /readyz latency spikes
	// under concurrent load, indicating pool exhaustion.
	poolMaxConns = 10

	// poolMinConns keeps warm connections so requests after an idle period
	// do not pay connection setup latency.
	poolMinConns = 2

	// poolMaxConnLifetime rotates connections periodically to recover from
	// silent TCP breakages or server-side connection limits.
	poolMaxConnLifetime = 60 * time.Minute

	// poolMaxConnIdleTime evicts connections idle longer than this duration,
	// reducing resource usage on the database server during quiet periods.
	poolMaxConnIdleTime = 5 * time.Minute

	// poolHealthCheckPeriod pings idle connections so stale ones are detected
	// before being handed to a caller.
	poolHealthCheckPeriod = 30 * time.Second
)

// NewPool creates a pgxpool connection pool with tuned parameters and verifies
// connectivity with a ping.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}
	cfg.MaxConns = poolMaxConns
	cfg.MinConns = poolMinConns
	cfg.MaxConnLifetime = poolMaxConnLifetime
	cfg.MaxConnIdleTime = poolMaxConnIdleTime
	cfg.HealthCheckPeriod = poolHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}
