# DEV-ONLY defaults — override via environment or .env before running make up outside localhost.
POSTGRES_PASSWORD ?= postgrespass
REDIS_PASSWORD ?= redispass
HYDRA_SYSTEM_SECRET ?= 0123456789abcdef0123456789abcdef
ADMIN_BOOTSTRAP_TOKEN ?= dev-bootstrap-token-0123456789abcdef0123456789abcdef
CORS_ALLOWED_ORIGINS ?= http://localhost:3000
AUTH_URL ?= http://localhost:8080
KRATOS_BROWSER_URL ?= http://localhost:4433
MAILPIT_URL ?= http://localhost:8025

COMPOSE_ENV = POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
	REDIS_PASSWORD=$(REDIS_PASSWORD) \
	HYDRA_SYSTEM_SECRET=$(HYDRA_SYSTEM_SECRET) \
	ADMIN_BOOTSTRAP_TOKEN=$(ADMIN_BOOTSTRAP_TOKEN) \
	CORS_ALLOWED_ORIGINS=$(CORS_ALLOWED_ORIGINS)

.PHONY: up down test vuln swagger check-health e2e wait verify-local config-check render-production-config production-bundle nix-develop nix-config-check nix-render-production-config nix-deploy-production frontend-dev frontend-build frontend-openapi

up:
	$(COMPOSE_ENV) docker compose up -d --build

down:
	$(COMPOSE_ENV) docker compose down

test:
	cd backend && go test ./...

swagger:
	cd backend && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs/swagger --ot go,json --parseInternal --generatedTime=false --exclude dist

vuln:
	cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check-health:
	@curl -fsS $(AUTH_URL)/healthz && echo " /healthz OK" || echo " /healthz FAILED"
	@curl -fsS $(AUTH_URL)/readyz  && echo " /readyz  OK" || echo " /readyz  FAILED"

config-check:
	cd backend && go run ./cmd/configcheck

render-production-config:
	./backend/scripts/render-production-config.sh

production-bundle:
	./backend/scripts/build-production-bundle.sh

nix-develop:
	cd backend && nix --extra-experimental-features "nix-command flakes" develop

nix-config-check:
	cd backend && nix --extra-experimental-features "nix-command flakes" run .#config-check

nix-render-production-config:
	cd backend && nix --extra-experimental-features "nix-command flakes" run .#render-production-config

nix-deploy-production:
	cd backend && nix --extra-experimental-features "nix-command flakes" run .#deploy-production -- ../.env

e2e:
	cd backend && RUN_E2E=1 AUTH_URL=$(AUTH_URL) KRATOS_BROWSER_URL=$(KRATOS_BROWSER_URL) MAILPIT_URL=$(MAILPIT_URL) go test ./integration/... -v

wait:
	@echo "Waiting for app to become ready..."
	@until curl -fsS $(AUTH_URL)/healthz >/dev/null; do sleep 1; done
	@until curl -fsS $(AUTH_URL)/readyz >/dev/null; do sleep 1; done

verify-local: up wait test e2e

frontend-openapi:
	$(MAKE) swagger
	cp backend/docs/swagger/swagger.json frontend/public/openapi.json

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
