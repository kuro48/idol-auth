package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const migrationsPath = "file:///migrations"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	// golang-migrate pgx/v5 driver requires pgx5:// scheme
	dsn = strings.Replace(dsn, "postgresql://", "pgx5://", 1)
	dsn = strings.Replace(dsn, "postgres://", "pgx5://", 1)

	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		slog.Error("create migrator", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			slog.Info("no migrations to apply")
			return
		}
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations applied successfully")
}
