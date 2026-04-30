package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	swaggerdocs "github.com/ryunosukekurokawa/idol-auth/docs/swagger"
	"github.com/ryunosukekurokawa/idol-auth/internal/config"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/account"
	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/profile"
	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
	"github.com/ryunosukekurokawa/idol-auth/internal/infra/db"
	"github.com/ryunosukekurokawa/idol-auth/internal/infra/hydra"
	"github.com/ryunosukekurokawa/idol-auth/internal/infra/kratos"
)

const shutdownTimeout = 10 * time.Second
const deletionWorkerInterval = time.Minute

// @title idol-auth API
// @version 1.0.0
// @description Ory Kratos / Hydra をバックエンドにした認証・認可プラットフォームの API です。
// @description
// @description - Auth API: Hydra login / consent / logout を仲介するブラウザ向け API
// @description - Admin API: アプリ登録、OIDC クライアント発行、ユーザー管理、監査ログ取得 API
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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
	configureSwagger(cfg.App.BaseURL)

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
	hydraAdmin := hydra.NewAdminClient(cfg.Ory.HydraAdminURL)

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
	limiter := apphttp.NewInMemoryRateLimiter(60, time.Minute)
	router := apphttp.NewRouter(apphttp.RouterConfig{
		App:        cfg.App,
		Admin:      cfg.Admin,
		Ory:        cfg.Ory,
		Security:   cfg.Security,
		Limiter:    limiter,
		ProfileSvc: profileService,
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

func configureSwagger(baseURL string) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	if u.Host != "" {
		swaggerdocs.SwaggerInfo.Host = u.Host
	}
	if u.Scheme != "" {
		swaggerdocs.SwaggerInfo.Schemes = []string{u.Scheme}
	}
	swaggerdocs.SwaggerInfo.BasePath = "/"
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
