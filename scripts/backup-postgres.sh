#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env.production}"
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backups}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing env file: $ENV_FILE" >&2
  exit 1
fi

set -a
. "$ENV_FILE"
set +a

mkdir -p "$BACKUP_DIR"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
archive_path="$BACKUP_DIR/idol_auth_${timestamp}.sql.gz"

echo "==> Creating PostgreSQL backup at $archive_path"
docker compose --env-file "$ENV_FILE" -f docker-compose.production.yml exec -T postgres \
  pg_dump -U idol -d idol_auth | gzip -c >"$archive_path"

echo "backup complete: $archive_path"
