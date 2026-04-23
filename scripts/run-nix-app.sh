#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <flake-app> [args...]" >&2
  exit 1
fi

NIX_BIN="${NIX_BIN:-}"
if [[ -z "$NIX_BIN" ]]; then
  if command -v nix >/dev/null 2>&1; then
    NIX_BIN="$(command -v nix)"
  elif [[ -x /nix/var/nix/profiles/default/bin/nix ]]; then
    NIX_BIN="/nix/var/nix/profiles/default/bin/nix"
  else
    echo "nix binary not found" >&2
    exit 1
  fi
fi

app="$1"
shift

exec "$NIX_BIN" --extra-experimental-features "nix-command flakes" run ".#${app}" -- "$@"
