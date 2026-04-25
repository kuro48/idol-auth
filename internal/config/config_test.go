package config_test

import (
	"os"
	"testing"

	"github.com/ryunosukekurokawa/idol-auth/internal/config"
)

// setRequiredOryVars sets the Ory-related required env vars for the duration of the test.
func setRequiredOryVars(t *testing.T) {
	t.Helper()
	t.Setenv("KRATOS_PUBLIC_URL", "http://localhost:4433")
	t.Setenv("KRATOS_ADMIN_URL", "http://localhost:4434")
	t.Setenv("HYDRA_PUBLIC_URL", "http://localhost:4444")
	t.Setenv("HYDRA_ADMIN_URL", "http://localhost:4445")
	t.Setenv("ADMIN_BOOTSTRAP_TOKEN", "dev-bootstrap-token-for-testing-env")
}

func TestLoad_DefaultValues(t *testing.T) {
	// Arrange
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	setRequiredOryVars(t)

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.App.Port)
	}
	if cfg.App.Env != "development" {
		t.Errorf("expected default env %q, got %q", "development", cfg.App.Env)
	}
	if cfg.App.BaseURL != "http://localhost:8080" {
		t.Errorf("expected default base URL %q, got %q", "http://localhost:8080", cfg.App.BaseURL)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected default log level %q, got %q", "info", cfg.Log.Level)
	}
	if !cfg.Security.CookieSecure {
		t.Error("expected CookieSecure to default to true")
	}
	if cfg.Admin.BootstrapToken != "dev-bootstrap-token-for-testing-env" {
		t.Errorf("expected admin bootstrap token to be loaded")
	}
	if len(cfg.Admin.AllowedEmails) != 0 {
		t.Errorf("expected no admin allowed emails by default, got %v", cfg.Admin.AllowedEmails)
	}
	if len(cfg.Admin.AllowedRoles) != 0 {
		t.Errorf("expected no admin allowed roles by default, got %v", cfg.Admin.AllowedRoles)
	}
	if cfg.Ory.KratosBrowserURL != "http://localhost:4433" {
		t.Errorf("expected default Kratos browser URL, got %q", cfg.Ory.KratosBrowserURL)
	}
	if cfg.Ory.HydraBrowserURL != "http://localhost:4444" {
		t.Errorf("expected default Hydra browser URL, got %q", cfg.Ory.HydraBrowserURL)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Arrange
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/prod")
	setRequiredOryVars(t)
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_BASE_URL", "https://example.com")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1,10.0.0.2")
	t.Setenv("KRATOS_BROWSER_URL", "http://browser-kratos:4433")
	t.Setenv("HYDRA_BROWSER_URL", "http://browser-hydra:4444")
	t.Setenv("ADMIN_ALLOWED_EMAILS", "admin@example.com,ops@example.com")
	t.Setenv("ADMIN_ALLOWED_ROLES", "admin,platform-operator")

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.App.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.App.Port)
	}
	if cfg.App.Env != "development" {
		t.Errorf("expected env %q, got %q", "development", cfg.App.Env)
	}
	if cfg.App.BaseURL != "https://example.com" {
		t.Errorf("expected base URL %q, got %q", "https://example.com", cfg.App.BaseURL)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level %q, got %q", "debug", cfg.Log.Level)
	}
	if cfg.Security.CookieSecure {
		t.Error("expected CookieSecure to be false")
	}
	if len(cfg.Security.TrustedProxies) != 2 {
		t.Errorf("expected 2 trusted proxies, got %d", len(cfg.Security.TrustedProxies))
	}
	if cfg.DB.URL != "postgres://user:pass@db:5432/prod" {
		t.Errorf("expected DB URL %q, got %q", "postgres://user:pass@db:5432/prod", cfg.DB.URL)
	}
	if cfg.Ory.KratosBrowserURL != "http://browser-kratos:4433" {
		t.Errorf("unexpected KratosBrowserURL: %q", cfg.Ory.KratosBrowserURL)
	}
	if cfg.Ory.HydraBrowserURL != "http://browser-hydra:4444" {
		t.Errorf("unexpected HydraBrowserURL: %q", cfg.Ory.HydraBrowserURL)
	}
	if len(cfg.Admin.AllowedEmails) != 2 || cfg.Admin.AllowedEmails[0] != "admin@example.com" || cfg.Admin.AllowedEmails[1] != "ops@example.com" {
		t.Errorf("unexpected admin allowed emails: %v", cfg.Admin.AllowedEmails)
	}
	if len(cfg.Admin.AllowedRoles) != 2 || cfg.Admin.AllowedRoles[0] != "admin" || cfg.Admin.AllowedRoles[1] != "platform-operator" {
		t.Errorf("unexpected admin allowed roles: %v", cfg.Admin.AllowedRoles)
	}
}

func TestLoad_OryURLs(t *testing.T) {
	// Arrange
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("KRATOS_PUBLIC_URL", "http://kratos:4433")
	t.Setenv("KRATOS_ADMIN_URL", "http://kratos:4434")
	t.Setenv("HYDRA_PUBLIC_URL", "http://hydra:4444")
	t.Setenv("HYDRA_ADMIN_URL", "http://hydra:4445")
	t.Setenv("ADMIN_BOOTSTRAP_TOKEN", "dev-bootstrap-token-for-testing-env")
	t.Setenv("KRATOS_BROWSER_URL", "http://localhost:4433")
	t.Setenv("HYDRA_BROWSER_URL", "http://localhost:4444")

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Ory.KratosPublicURL != "http://kratos:4433" {
		t.Errorf("unexpected KratosPublicURL: %q", cfg.Ory.KratosPublicURL)
	}
	if cfg.Ory.KratosAdminURL != "http://kratos:4434" {
		t.Errorf("unexpected KratosAdminURL: %q", cfg.Ory.KratosAdminURL)
	}
	if cfg.Ory.HydraPublicURL != "http://hydra:4444" {
		t.Errorf("unexpected HydraPublicURL: %q", cfg.Ory.HydraPublicURL)
	}
	if cfg.Ory.HydraAdminURL != "http://hydra:4445" {
		t.Errorf("unexpected HydraAdminURL: %q", cfg.Ory.HydraAdminURL)
	}
	if cfg.Ory.KratosBrowserURL != "http://localhost:4433" {
		t.Errorf("unexpected KratosBrowserURL: %q", cfg.Ory.KratosBrowserURL)
	}
	if cfg.Ory.HydraBrowserURL != "http://localhost:4444" {
		t.Errorf("unexpected HydraBrowserURL: %q", cfg.Ory.HydraBrowserURL)
	}
}

func TestLoad_MissingDatabaseURL_ReturnsError(t *testing.T) {
	// Arrange — explicitly unset DATABASE_URL
	prev, wasSet := os.LookupEnv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	if wasSet {
		t.Cleanup(func() { os.Setenv("DATABASE_URL", prev) })
	}
	setRequiredOryVars(t)

	// Act
	_, err := config.Load()

	// Assert
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}
}

func TestLoad_ProductionRejectsInsecureSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/prod")
	setRequiredOryVars(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_BASE_URL", "http://example.com")
	t.Setenv("KRATOS_BROWSER_URL", "http://kratos.example.com")
	t.Setenv("HYDRA_BROWSER_URL", "http://hydra.example.com")
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("TRUSTED_PROXIES", "")

	_, err := config.Load()

	if err == nil {
		t.Fatal("expected production config validation error")
	}
}

func TestLoad_ProductionAllowsHardenedSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/prod")
	setRequiredOryVars(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_BASE_URL", "https://auth.example.com")
	t.Setenv("KRATOS_BROWSER_URL", "https://accounts.example.com")
	t.Setenv("HYDRA_BROWSER_URL", "https://login.example.com")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,192.168.0.0/16")

	cfg, err := config.Load()

	if err != nil {
		t.Fatalf("expected hardened production config to load, got %v", err)
	}
	if cfg.App.Env != "production" {
		t.Fatalf("expected production env, got %q", cfg.App.Env)
	}
}
