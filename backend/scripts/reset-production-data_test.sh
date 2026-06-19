#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
env_file="$tmpdir/reset.env"
docker_log="$tmpdir/docker.log"

cat >"$env_file" <<'EOF'
APP_HOSTNAME=prod.example.com
POSTGRES_PASSWORD=test-password
EOF

cat >"$tmpdir/docker" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >"$docker_log"
EOF
chmod +x "$tmpdir/docker"

set +e
output="$(
  PATH="$tmpdir:$PATH" \
  ENV_FILE="$env_file" \
  SKIP_REDEPLOY_AFTER_RESET=true \
  bash scripts/reset-production-data.sh </dev/null 2>&1
)"
status=$?
set -e

if (( status != 0 )) && grep -q "refusing to reset production data" <<<"$output"; then
  echo "reset-production-data.sh should not require confirmation" >&2
  echo "$output" >&2
  exit 1
elif (( status != 0 )); then
  echo "reset-production-data.sh failed unexpectedly" >&2
  echo "$output" >&2
  exit 1
fi

if [[ ! -s "$docker_log" ]]; then
  echo "expected docker compose down to be called" >&2
  echo "$output" >&2
  exit 1
fi

if ! grep -q "compose --env-file $env_file -f docker-compose.yml down --volumes --remove-orphans" "$docker_log"; then
  echo "unexpected docker invocation:" >&2
  cat "$docker_log" >&2
  exit 1
fi
