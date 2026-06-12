#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/dist}"

required_vars=(
  APP_BASE_URL
  PORTAL_APP_URL
  KRATOS_BROWSER_URL
  HYDRA_BROWSER_URL
  SESSION_COOKIE_DOMAIN
  KRATOS_TOTP_ISSUER
  KRATOS_SMTP_FROM_ADDRESS
  KRATOS_SMTP_FROM_NAME
  ORY_LOG_LEVEL
)

for var_name in "${required_vars[@]}"; do
  if [[ -z "${!var_name:-}" ]]; then
    echo "missing required env: $var_name" >&2
    exit 1
  fi
done

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
