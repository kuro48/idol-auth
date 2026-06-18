package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kuro48/idol-auth/internal/config"
	"github.com/kuro48/idol-auth/internal/domain/account"
	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
	"github.com/kuro48/idol-auth/internal/domain/app"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
	"github.com/kuro48/idol-auth/internal/domain/loginhistory"
	"github.com/kuro48/idol-auth/internal/domain/profile"
	apphttp "github.com/kuro48/idol-auth/internal/http"
	"github.com/kuro48/idol-auth/internal/infra/db"
	"github.com/kuro48/idol-auth/internal/infra/hydra"
	"github.com/kuro48/idol-auth/internal/infra/kratos"
	"github.com/kuro48/idol-auth/internal/infra/mail"
	"github.com/kuro48/idol-auth/internal/infra/webhook"
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
	webhookRepo := db.NewAppWebhookRepository(dbPool)
	webhookDispatcher := webhook.NewDispatcher(webhookRepo)
	accountNotifier := mail.NewAccountSMTPNotifier(cfg.Mail, kratosAdmin)

	accountOpts := []account.ServiceOption{
		account.WithWebhookDispatcher(webhookDispatcher),
		account.WithProfileReader(kratosAdmin),
	}
	if accountNotifier != nil {
		accountOpts = append(accountOpts, account.WithAccountMailer(accountNotifier))
	}
	accountService := account.NewService(
		accountRepo,
		accountRepo,
		appService,
		kratosAdmin,
		kratosAdmin,
		tokenRepo,
		auditRepo,
		time.Now,
		30*24*time.Hour,
		accountOpts...,
	)
	kratosFrontend := kratos.NewFrontendClient(cfg.Ory.KratosPublicURL, cfg.Ory.KratosBrowserURL)
	authService := apphttp.NewAuthServiceWithOptions(
		cfg.App.BaseURL,
		hydra.NewFlowClient(cfg.Ory.HydraAdminURL),
		kratosFrontend,
		accountService,
		kratosAdmin,
	)
	loginHistoryService := loginhistory.NewService(db.NewLoginEventRepository(dbPool))
	apphttp.SetLoginRecorder(authService, newLoginRecorder(loginHistoryService, accountNotifier, kratosAdmin))
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
		mail.NewSMTPNotifier(cfg.Mail, cfg.Admin.AllowedEmails),
		time.Now,
	)
	limiter := apphttp.NewInMemoryRateLimiter(60, time.Minute)
	router := apphttp.NewRouter(apphttp.RouterConfig{
		App:                cfg.App,
		Admin:              cfg.Admin,
		Ory:                cfg.Ory,
		Security:           cfg.Security,
		Limiter:            limiter,
		AccountSvc:         accountService,
		ProfileSvc:         profileService,
		PublicSvc:          publicService,
		AdminAppRegSvc:     appRegService,
		DeveloperAppRegSvc: appRegService,
		WebhookRepo:        webhookRepo,
		SessionMgr:         kratosAdmin,
		LoginHistorySvc:    loginHistoryService,
		EmailVerifSvc:      kratosAdmin,
		PasswordChangeSvc:  kratosFrontend,
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

// deviceLoginNotifier sends an email when a login from a new device is detected.
type deviceLoginNotifier interface {
	NotifyNewDeviceLogin(ctx context.Context, identityID, ipAddress, userAgent string, at time.Time) error
}

// profileGetter retrieves an identity's profile including notification preferences.
type profileGetter interface {
	GetIdentityProfile(ctx context.Context, identityID string) (profile.Profile, error)
}

// loginRecorderAdapter bridges apphttp.LoginRecorder to loginhistory.Service.
// RecordObservedSession returns immediately and persists the event from a
// background goroutine so request handling latency is unaffected.
// When mailer and profiler are set, it also detects new-device logins and
// sends a notification when security_alerts preferences are enabled.
type loginRecorderAdapter struct {
	svc     *loginhistory.Service
	mailer  deviceLoginNotifier
	profiler profileGetter
}

func newLoginRecorder(svc *loginhistory.Service, mailer deviceLoginNotifier, profiler profileGetter) *loginRecorderAdapter {
	return &loginRecorderAdapter{svc: svc, mailer: mailer, profiler: profiler}
}

func (a *loginRecorderAdapter) RecordObservedSession(_ context.Context, session apphttp.KratosSession) {
	if a == nil || a.svc == nil {
		return
	}
	if session.ID == "" || session.IdentityID == "" {
		return
	}
	evt := loginhistory.Event{
		IdentityID:      session.IdentityID,
		SessionID:       session.ID,
		AuthenticatedAt: session.AuthenticatedAt,
		IssuedAt:        session.IssuedAt,
		AAL:             session.AuthenticatorAssuranceLevel,
		Methods:         session.Methods,
		IPAddress:       session.IPAddress,
		UserAgent:       session.UserAgent,
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Check for new device BEFORE recording so the current session is not included.
		isNew := false
		if a.mailer != nil && a.profiler != nil && strings.TrimSpace(evt.UserAgent) != "" {
			since := evt.AuthenticatedAt.Add(-30 * 24 * time.Hour)
			var err error
			isNew, err = a.svc.IsNewDevice(ctx, evt.IdentityID, evt.UserAgent, since)
			if err != nil {
				slog.Warn("check new device failed", "identity_id", evt.IdentityID, "error", err)
				isNew = false
			}
		}

		if err := a.svc.Record(ctx, evt); err != nil {
			slog.Warn("login history record failed", "session_id", evt.SessionID, "error", err)
			return
		}

		if !isNew {
			return
		}
		p, err := a.profiler.GetIdentityProfile(ctx, evt.IdentityID)
		if err != nil {
			slog.Warn("get profile for new device check failed", "identity_id", evt.IdentityID, "error", err)
			return
		}
		if !p.NotificationPreferences.SecurityAlerts {
			return
		}
		if err := a.mailer.NotifyNewDeviceLogin(ctx, evt.IdentityID, evt.IPAddress, evt.UserAgent, evt.AuthenticatedAt); err != nil {
			slog.Warn("new device login notification failed", "identity_id", evt.IdentityID, "error", err)
		}
	}()
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
