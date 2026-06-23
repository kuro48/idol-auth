#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/dist}"
ENV_FILE="${ENV_FILE:-}"

if [[ -z "$ENV_FILE" ]]; then
  if [[ -f "$ROOT_DIR/.env" ]]; then
    ENV_FILE="$ROOT_DIR/.env"
  elif [[ -f "$ROOT_DIR/../.env" ]]; then
    ENV_FILE="$ROOT_DIR/../.env"
  fi
fi

if [[ -n "$ENV_FILE" ]]; then
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "missing env file: $ENV_FILE" >&2
    exit 1
  fi
  eval "$("$ROOT_DIR/scripts/export-env-file.sh" "$ENV_FILE")"
fi

required_vars=(
  APP_BASE_URL
  HYDRA_HOSTNAME
  SESSION_COOKIE_DOMAIN
  KRATOS_TOTP_ISSUER
  KRATOS_SMTP_FROM_ADDRESS
  KRATOS_SMTP_FROM_NAME
  ORY_LOG_LEVEL
  WEBAUTHN_RP_ID
  WEBAUTHN_RP_DISPLAY_NAME
)

for var_name in "${required_vars[@]}"; do
  if [[ -z "${!var_name:-}" ]]; then
    echo "missing required env: $var_name" >&2
    exit 1
  fi
done

# Derive browser-facing URLs from canonical base URLs.
# These are always the same domain in this deployment topology
# and do not need to be set separately in the deploy .env file.
export HYDRA_HOSTNAME

render_template() {
  local src="$1"
  local dest="$2"
  mkdir -p "$(dirname "$dest")"
  perl -pe 's/\{\{([A-Z0-9_]+)\}\}/exists $ENV{$1} ? $ENV{$1} : die "missing required env: $1\n"/ge' "$src" >"$dest"
}

render_template "$ROOT_DIR/deploy/kratos/kratos.production.yml.tmpl" "$OUT_DIR/kratos/kratos.yml"
render_template "$ROOT_DIR/deploy/hydra/hydra.production.yml.tmpl" "$OUT_DIR/hydra/hydra.yml"
cp "$ROOT_DIR/deploy/kratos/identity.schema.json" "$OUT_DIR/kratos/identity.schema.json"

echo "rendered production config to $OUT_DIR"
