#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/dist/production-bundle}"

include_paths=(
  .gitignore
  .dockerignore
  .env.production.example
  Dockerfile
  docker-compose.production.yml
  go.mod
  go.sum
  cmd/configcheck
  cmd/migrate
  cmd/portal
  cmd/server
  deploy/caddy
  deploy/hydra
  deploy/kratos
  deploy/postgres
  docs/swagger/docs.go
  internal/config
  internal/demo
  internal/domain
  internal/http
  internal/infra
  internal/oshi
  scripts/backup-postgres.sh
  scripts/deploy-production.sh
  scripts/export-env-file.sh
  scripts/render-production-config.sh
)

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

copy_path() {
  local rel_path="$1"
  local src="$ROOT_DIR/$rel_path"
  local dest_dir="$OUT_DIR/$(dirname "$rel_path")"

  if [[ ! -e "$src" ]]; then
    echo "missing required path: $rel_path" >&2
    exit 1
  fi

  mkdir -p "$dest_dir"
  cp -R "$src" "$dest_dir/"
}

for path in "${include_paths[@]}"; do
  copy_path "$path"
done

find "$OUT_DIR" -type f -name '*_test.go' -delete

file_count="$(find "$OUT_DIR" -type f | wc -l | tr -d ' ')"
bundle_size="$(du -sh "$OUT_DIR" | awk '{print $1}')"

echo "production bundle written to $OUT_DIR"
echo "files: $file_count"
echo "size: $bundle_size"
