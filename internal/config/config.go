package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	Ory      OryConfig
	Admin    AdminConfig
	Security SecurityConfig
	Log      LogConfig
}

type AppConfig struct {
	Env     string `env:"APP_ENV"      envDefault:"development"`
	Port    int    `env:"APP_PORT"     envDefault:"8080"`
	BaseURL string `env:"APP_BASE_URL" envDefault:"http://localhost:8080"`
}

type DBConfig struct {
	URL string `env:"DATABASE_URL,required"`
}

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR"     envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
}

type OryConfig struct {
	KratosPublicURL  string `env:"KRATOS_PUBLIC_URL,required"`
	KratosAdminURL   string `env:"KRATOS_ADMIN_URL,required"`
	HydraPublicURL   string `env:"HYDRA_PUBLIC_URL,required"`
	HydraAdminURL    string `env:"HYDRA_ADMIN_URL,required"`
	KratosBrowserURL string `env:"KRATOS_BROWSER_URL" envDefault:"http://localhost:4433"`
	HydraBrowserURL  string `env:"HYDRA_BROWSER_URL"  envDefault:"http://localhost:4444"`
}

type AdminConfig struct {
	BootstrapToken string   `env:"ADMIN_BOOTSTRAP_TOKEN"`
	AllowedEmails  []string `env:"ADMIN_ALLOWED_EMAILS" envSeparator:","`
	AllowedRoles   []string `env:"ADMIN_ALLOWED_ROLES" envSeparator:","`
}

type SecurityConfig struct {
	CookieSecure   bool     `env:"SESSION_COOKIE_SECURE" envDefault:"true"`
	TrustedProxies []string `env:"TRUSTED_PROXIES"       envSeparator:","`
}

type LogConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: parse env: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	// Token strength is enforced whenever a token is set, regardless of environment.
	// This prevents weak tokens from being used in staging or shared dev environments.
	token := strings.TrimSpace(c.Admin.BootstrapToken)
	if token != "" {
		if len(token) < 32 {
			return fmt.Errorf("config: ADMIN_BOOTSTRAP_TOKEN must be at least 32 characters")
		}
		if isKnownWeakToken(token) {
			return fmt.Errorf("config: ADMIN_BOOTSTRAP_TOKEN appears to be a well-known weak value; rotate it")
		}
	}

	if strings.EqualFold(strings.TrimSpace(c.App.Env), "production") {
		if !c.Security.CookieSecure {
			return fmt.Errorf("config: production requires SESSION_COOKIE_SECURE=true")
		}
		if strings.EqualFold(strings.TrimSpace(c.Log.Level), "debug") {
			return fmt.Errorf("config: production forbids LOG_LEVEL=debug")
		}
		if len(c.Security.TrustedProxies) == 0 {
			return fmt.Errorf("config: production requires TRUSTED_PROXIES")
		}
		if err := requireHTTPSURL("APP_BASE_URL", c.App.BaseURL); err != nil {
			return err
		}
		if err := requireHTTPSURL("KRATOS_BROWSER_URL", c.Ory.KratosBrowserURL); err != nil {
			return err
		}
		if err := requireHTTPSURL("HYDRA_BROWSER_URL", c.Ory.HydraBrowserURL); err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("config: production requires ADMIN_BOOTSTRAP_TOKEN")
		}
	}
	return nil
}

var knownWeakTokens = []string{
	"localdev", "changeme", "secret", "password", "test", "development",
	"admin", "change_me", "please_change_me", "bootstrap", "token",
}

func isKnownWeakToken(token string) bool {
	lower := strings.ToLower(token)
	for _, weak := range knownWeakTokens {
		if lower == weak {
			return true
		}
	}
	return false
}

func requireHTTPSURL(name, raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("config: parse %s: %w", name, err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("config: production requires %s to use https", name)
	}
	if parsed.Host == "" {
		return fmt.Errorf("config: production requires %s host", name)
	}
	return nil
}
