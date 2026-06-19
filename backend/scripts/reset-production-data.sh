#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env}"
SKIP_REDEPLOY_AFTER_RESET="${SKIP_REDEPLOY_AFTER_RESET:-false}"
export ENV_FILE

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing env file: $ENV_FILE" >&2
  exit 1
fi

eval "$("$ROOT_DIR/scripts/export-env-file.sh" "$ENV_FILE")"

required_vars=(
  APP_HOSTNAME
  POSTGRES_PASSWORD
)

for var_name in "${required_vars[@]}"; do
  if [[ -z "${!var_name:-}" ]]; then
    echo "missing required env: $var_name" >&2
    exit 1
  fi
done

expected_confirmation="reset ${APP_HOSTNAME} production data"
provided_confirmation="${RESET_PRODUCTION_DATA_CONFIRM:-}"
redeploy_after_reset=true

case "$SKIP_REDEPLOY_AFTER_RESET" in
  true | TRUE | 1 | yes | YES)
    redeploy_after_reset=false
    ;;
  false | FALSE | 0 | no | NO | "")
    ;;
  *)
    echo "invalid SKIP_REDEPLOY_AFTER_RESET: $SKIP_REDEPLOY_AFTER_RESET" >&2
    echo "valid values: true, false" >&2
    exit 1
    ;;
esac

if [[ -z "$provided_confirmation" && -t 0 ]]; then
  cat >&2 <<EOF
This deletes all Postgres data and uploaded files managed by docker compose:
  - postgres_data
  - uploads_data

Target host: ${APP_HOSTNAME}

Type this exact confirmation to continue:
  ${expected_confirmation}

EOF
  read -r -p "Confirmation: " provided_confirmation
fi

if [[ "$provided_confirmation" != "$expected_confirmation" ]]; then
  printf -v confirm_arg "%q" "$expected_confirmation"
  printf -v env_file_arg "%q" "$ENV_FILE"
  cat >&2 <<EOF
refusing to reset production data

This deletes all Postgres data and uploaded files managed by docker compose:
  - postgres_data
  - uploads_data

Target host: ${APP_HOSTNAME}

Run again with this exact confirmation:
  RESET_PRODUCTION_DATA_CONFIRM=${confirm_arg} ENV_FILE=${env_file_arg} bash scripts/reset-production-data.sh

Set SKIP_REDEPLOY_AFTER_RESET=true to leave the stack stopped after deleting volumes.
EOF
  exit 1
fi

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

echo "==> Resetting production data for ${APP_HOSTNAME}"
echo "==> Stopping stack and deleting compose-managed data volumes"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml down --volumes --remove-orphans

echo "==> Production data volumes deleted"

if [[ "$redeploy_after_reset" != true ]]; then
  echo "==> Redeploy skipped; stack is stopped"
  exit 0
fi

echo "==> Redeploying production stack with empty data volumes"
exec "$ROOT_DIR/scripts/deploy-production.sh"
