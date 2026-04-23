#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env.production}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing env file: $ENV_FILE" >&2
  exit 1
fi

set -a
. "$ENV_FILE"
set +a

echo "==> Rendering production config"
./scripts/render-production-config.sh

echo "==> Validating application config"
make config-check

echo "==> Validating production compose"
docker compose --env-file "$ENV_FILE" -f docker-compose.production.yml config >/dev/null

echo "==> Pulling base images"
docker compose --env-file "$ENV_FILE" -f docker-compose.production.yml pull || true

echo "==> Deploying production stack"
docker compose --env-file "$ENV_FILE" -f docker-compose.production.yml up -d --build

echo "==> Waiting for app readiness"
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:8080/readyz" >/dev/null 2>&1; then
    echo "production stack deployed"
    exit 0
  fi
  sleep 2
done

echo "app readiness check timed out" >&2
exit 1
