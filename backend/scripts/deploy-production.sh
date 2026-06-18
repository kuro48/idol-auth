#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing env file: $ENV_FILE" >&2
  exit 1
fi

eval "$("$ROOT_DIR/scripts/export-env-file.sh" "$ENV_FILE")"

required_vars=(
  DATABASE_URL
  KRATOS_DSN
  HYDRA_DSN
  KRATOS_SMTP_CONNECTION_URI
  KRATOS_SECRETS_DEFAULT
  KRATOS_SECRETS_COOKIE
  KRATOS_SECRETS_CIPHER
  HYDRA_SYSTEM_SECRET
)

for var_name in "${required_vars[@]}"; do
  if [[ -z "${!var_name:-}" ]]; then
    echo "missing required env: $var_name" >&2
    exit 1
  fi
done

echo "==> Rendering production config"
./scripts/render-production-config.sh

echo "==> Validating application config"
docker run --rm --env-file "$ENV_FILE" ghcr.io/kuro48/idol-auth/configcheck:latest

echo "==> Validating production compose"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml config >/dev/null

echo "==> Pulling images"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml pull

echo "==> Deploying production stack"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml up -d

echo "==> Waiting for app readiness"
for i in $(seq 1 150); do
  if docker compose --env-file "$ENV_FILE" -f docker-compose.yml exec -T app wget -qO- http://localhost:8080/readyz >/dev/null 2>&1; then
    echo "production stack deployed"
    exit 0
  fi
  echo "  waiting... (${i}/150)"
  sleep 2
done

echo "app readiness check timed out" >&2
echo "==> Recent migration logs" >&2
docker compose --env-file "$ENV_FILE" -f docker-compose.yml logs --tail=120 kratos-migrate hydra-migrate migrate >&2 || true
exit 1
