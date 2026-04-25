POSTGRES_PASSWORD ?= postgrespass
REDIS_PASSWORD ?= redispass
HYDRA_SYSTEM_SECRET ?= 0123456789abcdef0123456789abcdef
ADMIN_BOOTSTRAP_TOKEN ?= bootstrap-token
DEMO_PORT ?= 3002
DEMO_APP_URL ?= http://localhost:3002
CORS_ALLOWED_ORIGINS ?= http://localhost:3002
APP_URL ?= $(DEMO_APP_URL)
AUTH_URL ?= http://localhost:8080
KRATOS_BROWSER_URL ?= http://localhost:4433
MAILPIT_URL ?= http://localhost:8025

COMPOSE_ENV = POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
	REDIS_PASSWORD=$(REDIS_PASSWORD) \
	HYDRA_SYSTEM_SECRET=$(HYDRA_SYSTEM_SECRET) \
	ADMIN_BOOTSTRAP_TOKEN=$(ADMIN_BOOTSTRAP_TOKEN) \
	DEMO_PORT=$(DEMO_PORT) \
	DEMO_APP_URL=$(DEMO_APP_URL) \
	CORS_ALLOWED_ORIGINS=$(CORS_ALLOWED_ORIGINS)

.PHONY: up down test vuln check-health e2e wait verify-local config-check render-production-config nix-develop nix-config-check nix-render-production-config nix-deploy-production nix-backup-postgres

up:
	$(COMPOSE_ENV) docker compose up -d --build

down:
	$(COMPOSE_ENV) docker compose down

test:
	go test ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check-health:
	@curl -fsS $(AUTH_URL)/healthz && echo " /healthz OK" || echo " /healthz FAILED"
	@curl -fsS $(AUTH_URL)/readyz  && echo " /readyz  OK" || echo " /readyz  FAILED"

config-check:
	go run ./cmd/configcheck

render-production-config:
	./scripts/render-production-config.sh

nix-develop:
	nix --extra-experimental-features "nix-command flakes" develop

nix-config-check:
	nix --extra-experimental-features "nix-command flakes" run .#config-check

nix-render-production-config:
	nix --extra-experimental-features "nix-command flakes" run .#render-production-config

nix-deploy-production:
	nix --extra-experimental-features "nix-command flakes" run .#deploy-production -- .env.production

nix-backup-postgres:
	nix --extra-experimental-features "nix-command flakes" run .#backup-postgres -- .env.production

e2e:
	RUN_E2E=1 APP_URL=$(APP_URL) AUTH_URL=$(AUTH_URL) KRATOS_BROWSER_URL=$(KRATOS_BROWSER_URL) MAILPIT_URL=$(MAILPIT_URL) go test ./integration/... -v

wait:
	@echo "Waiting for app and demo to become ready..."
	@until curl -fsS $(AUTH_URL)/healthz >/dev/null; do sleep 1; done
	@until curl -fsS $(AUTH_URL)/readyz >/dev/null; do sleep 1; done
	@until curl -fsS $(APP_URL)/ >/dev/null; do sleep 1; done

verify-local: up wait test e2e
