package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ryunosukekurokawa/idol-auth/internal/config"
	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
	"github.com/ryunosukekurokawa/idol-auth/internal/infra/db"
	"github.com/ryunosukekurokawa/idol-auth/internal/infra/hydra"
	"github.com/ryunosukekurokawa/idol-auth/internal/infra/kratos"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	setupLogger(cfg.Log.Level)

	dbPool, err := db.NewPool(context.Background(), cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("init db pool: %w", err)
	}
	defer dbPool.Close()

	appService := app.NewService(
		db.NewAppRepository(dbPool),
		db.NewOIDCClientRepository(dbPool),
		db.NewAuditRepository(dbPool),
		hydra.NewAdminClient(cfg.Ory.HydraAdminURL),
		time.Now,
	)
	adminService := admindomain.NewService(
		appService,
		kratos.NewAdminClient(cfg.Ory.KratosAdminURL),
		db.NewAuditRepository(dbPool),
		time.Now,
	)
	authService := apphttp.NewAuthService(
		cfg.App.BaseURL,
		hydra.NewFlowClient(cfg.Ory.HydraAdminURL),
		kratos.NewFrontendClient(cfg.Ory.KratosPublicURL, cfg.Ory.KratosBrowserURL),
	)
	limiter := apphttp.NewInMemoryRateLimiter(60, time.Minute)
	router := apphttp.NewRouter(apphttp.RouterConfig{
		App:      cfg.App,
		Admin:    cfg.Admin,
		Ory:      cfg.Ory,
		Security: cfg.Security,
		Limiter:  limiter,
	}, adminService, db.NewReadinessChecker(dbPool), authService)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "addr", srv.Addr, "env", cfg.App.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server listen error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	slog.Info("server stopped")
	return nil
}

func setupLogger(level string) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})))
}
