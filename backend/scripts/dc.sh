#!/usr/bin/env bash
# Wrapper for docker compose that injects runtime-generated DEPLOY_* variables.
# Usage: ./scripts/dc.sh [docker compose args...]
# Example: ./scripts/dc.sh logs -f app
#          ./scripts/dc.sh ps
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing env file: $ENV_FILE" >&2
  exit 1
fi

eval "$("$ROOT_DIR/scripts/export-env-file.sh" "$ENV_FILE")"

urlencode() {
  perl -e 'my $s = shift; $s =~ s/([^A-Za-z0-9._~-])/sprintf("%%%02X", ord($1))/eg; print $s' "$1"
}

POSTGRES_PASSWORD_ENCODED="$(urlencode "$POSTGRES_PASSWORD")"
export DEPLOY_DATABASE_URL="postgres://idol:${POSTGRES_PASSWORD_ENCODED}@postgres:5432/idol_auth?sslmode=disable"
export DEPLOY_KRATOS_DSN="${DEPLOY_DATABASE_URL}&search_path=kratos"
export DEPLOY_HYDRA_DSN="${DEPLOY_DATABASE_URL}&search_path=hydra"

exec docker compose --env-file "$ENV_FILE" -f "$ROOT_DIR/docker-compose.yml" "$@"
