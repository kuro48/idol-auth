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
  POSTGRES_PASSWORD
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

urlencode() {
  perl -e 'my $s = shift; $s =~ s/([^A-Za-z0-9._~-])/sprintf("%%%02X", ord($1))/eg; print $s' "$1"
}

POSTGRES_PASSWORD_ENCODED="$(urlencode "$POSTGRES_PASSWORD")"
export DEPLOY_DATABASE_URL="postgres://idol:${POSTGRES_PASSWORD_ENCODED}@postgres:5432/idol_auth?sslmode=disable"
export DEPLOY_KRATOS_DSN="${DEPLOY_DATABASE_URL}&search_path=kratos"
export DEPLOY_HYDRA_DSN="${DEPLOY_DATABASE_URL}&search_path=hydra"
export DATABASE_URL="$DEPLOY_DATABASE_URL"
export KRATOS_DSN="$DEPLOY_KRATOS_DSN"
export HYDRA_DSN="$DEPLOY_HYDRA_DSN"

validate_postgres_sslmode() {
  local var_name="$1"
  local dsn="${!var_name}"
  local query
  local param
  local key
  local value
  local sslmode=""
  local -a params

  if [[ "$dsn" != *\?* ]]; then
    return 0
  fi

  query="${dsn#*\?}"
  IFS='&' read -ra params <<<"$query"
  for param in "${params[@]}"; do
    key="${param%%=*}"
    value="${param#*=}"
    key="${key//$'\r'/}"
    value="${value//$'\r'/}"
    key="${key#"${key%%[![:space:]]*}"}"
    key="${key%"${key##*[![:space:]]}"}"
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    if [[ "$key" == "sslmode" ]]; then
      sslmode="$value"
      break
    fi
  done

  case "$sslmode" in
    "" | disable | allow | prefer | require | verify-ca | verify-full)
      ;;
    *)
      printf '%s has invalid sslmode: %q\n' "$var_name" "$sslmode" >&2
      echo "valid sslmode values: disable, allow, prefer, require, verify-ca, verify-full" >&2
      echo "for the bundled postgres service, use sslmode=disable" >&2
      exit 1
      ;;
  esac
}

validate_postgres_sslmode DEPLOY_DATABASE_URL
validate_postgres_sslmode DEPLOY_KRATOS_DSN
validate_postgres_sslmode DEPLOY_HYDRA_DSN

echo "==> Rendering production config"
./scripts/render-production-config.sh

echo "==> Validating application config"
docker run --rm --pull always --env-file "$ENV_FILE" -e DATABASE_URL ghcr.io/kuro48/idol-auth/configcheck:latest

echo "==> Validating production compose"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml config >/dev/null

echo "==> Pulling images"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml pull

echo "==> Resetting one-shot migration containers"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml rm -f -s kratos-migrate hydra-migrate migrate

echo "==> Deploying production stack"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml up -d

echo "==> Restarting Kratos to pick up fresh config"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml restart kratos

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
