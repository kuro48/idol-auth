package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	Ory      OryConfig
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
	KratosPublicURL string `env:"KRATOS_PUBLIC_URL,required"`
	KratosAdminURL  string `env:"KRATOS_ADMIN_URL,required"`
	HydraPublicURL  string `env:"HYDRA_PUBLIC_URL,required"`
	HydraAdminURL   string `env:"HYDRA_ADMIN_URL,required"`
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
	return cfg, nil
}
