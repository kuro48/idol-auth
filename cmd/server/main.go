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

	"github.com/kuro48/idol-auth/internal/config"
	"github.com/kuro48/idol-auth/internal/domain/account"
	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
	"github.com/kuro48/idol-auth/internal/domain/app"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
	"github.com/kuro48/idol-auth/internal/domain/profile"
	apphttp "github.com/kuro48/idol-auth/internal/http"
	"github.com/kuro48/idol-auth/internal/infra/db"
	"github.com/kuro48/idol-auth/internal/infra/hydra"
	"github.com/kuro48/idol-auth/internal/infra/kratos"
	"github.com/kuro48/idol-auth/internal/infra/mail"
)

const shutdownTimeout = 10 * time.Second
const deletionWorkerInterval = time.Minute

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

	auditRepo := db.NewAuditRepository(dbPool)
	oidcRepo := db.NewOIDCClientRepository(dbPool)
	tokenRepo := db.NewAppManagementTokenRepository(dbPool)
	accountRepo := db.NewAccountRepository(dbPool)
	kratosAdmin := kratos.NewAdminClient(cfg.Ory.KratosAdminURL)
	kratosNative := kratos.NewNativeClient(cfg.Ory.KratosPublicURL)
	hydraAdmin := hydra.NewAdminClient(cfg.Ory.HydraAdminURL)
	hydraFacade := hydra.NewFacadeClient(cfg.Ory.HydraPublicURL, cfg.Ory.HydraAdminURL)

	appService := app.NewService(
		db.NewAppRepository(dbPool),
		oidcRepo,
		auditRepo,
		hydraAdmin,
		time.Now,
		tokenRepo,
	)
	adminService := admindomain.NewService(
		appService,
		kratosAdmin,
		auditRepo,
		time.Now,
	)
	accountService := account.NewService(
		accountRepo,
		accountRepo,
		appService,
		kratosAdmin,
		kratosAdmin,
		tokenRepo,
		auditRepo,
		time.Now,
		0,
	)
	authService := apphttp.NewAuthServiceWithOptions(
		cfg.App.BaseURL,
		hydra.NewFlowClient(cfg.Ory.HydraAdminURL),
		kratos.NewFrontendClient(cfg.Ory.KratosPublicURL, cfg.Ory.KratosBrowserURL),
		accountService,
		kratosAdmin,
	)
	profileService := profile.NewService(kratosAdmin)
	publicService := apphttp.NewPublicAuthService(
		hydraFacade,
		kratosNative,
		cfg.Ory.HydraBrowserURL,
		cfg.Ory.KratosBrowserURL,
		cfg.Ory.KratosPublicURL,
	)
	appRegService := appreg.NewService(
		db.NewAppRegistrationRepository(dbPool),
		mail.NewSMTPNotifier(cfg.Mail),
		time.Now,
	)
	limiter := apphttp.NewInMemoryRateLimiter(60, time.Minute)
	router := apphttp.NewRouter(apphttp.RouterConfig{
		App:            cfg.App,
		Admin:          cfg.Admin,
		Ory:            cfg.Ory,
		Security:       cfg.Security,
		Limiter:        limiter,
		ProfileSvc:     profileService,
		PublicSvc:      publicService,
		AdminAppRegSvc:     appRegService,
		DeveloperAppRegSvc: appRegService,
	}, adminService, db.NewReadinessChecker(dbPool), authService, accountService)

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

	go runDeletionWorker(ctx, accountService)

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

func runDeletionWorker(ctx context.Context, accountSvc *account.Service) {
	if accountSvc == nil {
		return
	}
	ticker := time.NewTicker(deletionWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := accountSvc.ProcessDueDeletionRequests(ctx, 50); err != nil {
				slog.Error("account deletion worker failed", "error", err)
			}
		}
	}
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
