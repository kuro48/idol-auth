#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/dist/production-bundle}"

include_paths=(
  docker-compose.yml
  deploy/hydra/hydra.production.yml.tmpl
  deploy/kratos/identity.schema.json
  deploy/kratos/kratos.production.yml.tmpl
  deploy/postgres
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

file_count="$(find "$OUT_DIR" -type f | wc -l | tr -d ' ')"
bundle_size="$(du -sh "$OUT_DIR" | awk '{print $1}')"

echo "production bundle written to $OUT_DIR"
echo "files: $file_count"
echo "size: $bundle_size"
