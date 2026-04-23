package demo

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port                  int    `env:"DEMO_PORT" envDefault:"3002"`
	AppURL                string `env:"DEMO_APP_URL" envDefault:"http://localhost:3002"`
	AuthInternalURL       string `env:"DEMO_AUTH_INTERNAL_URL,required"`
	KratosPublicURL       string `env:"DEMO_KRATOS_PUBLIC_URL" envDefault:"http://localhost:4433"`
	KratosBrowserURL      string `env:"DEMO_KRATOS_BROWSER_URL,required"`
	HydraPublicURL        string `env:"DEMO_HYDRA_PUBLIC_URL" envDefault:"http://localhost:4444"`
	HydraBrowserURL       string `env:"DEMO_HYDRA_BROWSER_URL,required"`
	AdminToken            string `env:"ADMIN_BOOTSTRAP_TOKEN,required"`
	AppName               string `env:"DEMO_CLIENT_APP_NAME" envDefault:"Idol Demo App"`
	AppSlug               string `env:"DEMO_CLIENT_APP_SLUG" envDefault:"idol-demo-app"`
	AppDescription        string `env:"DEMO_CLIENT_APP_DESCRIPTION" envDefault:"local development demo client"`
	PartnerAppName        string `env:"DEMO_PARTNER_APP_NAME" envDefault:"Idol Partner Demo"`
	PartnerAppSlug        string `env:"DEMO_PARTNER_APP_SLUG" envDefault:"idol-partner-demo"`
	PartnerAppDescription string `env:"DEMO_PARTNER_APP_DESCRIPTION" envDefault:"local development third-party demo client"`
}

type PortalConfig struct {
	Port             int    `env:"PORTAL_PORT" envDefault:"3003"`
	AppURL           string `env:"PORTAL_APP_URL,required"`
	KratosPublicURL  string `env:"PORTAL_KRATOS_PUBLIC_URL" envDefault:"http://localhost:4433"`
	KratosBrowserURL string `env:"PORTAL_KRATOS_BROWSER_URL,required"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("demo config: parse env: %w", err)
	}
	return cfg, nil
}

func LoadPortalConfig() (*PortalConfig, error) {
	cfg := &PortalConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("portal config: parse env: %w", err)
	}
	return cfg, nil
}
