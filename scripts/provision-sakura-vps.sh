#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DEPLOY_USER="${DEPLOY_USER:-deploy}"
APP_DIR="${APP_DIR:-/opt/idol-auth}"
REPO_URL="${REPO_URL:-}"
REPO_REF="${REPO_REF:-main}"
SETUP_SWAP="${SETUP_SWAP:-1}"
SWAP_SIZE_GB="${SWAP_SIZE_GB:-2}"

"$ROOT_DIR/scripts/bootstrap-ubuntu-nix.sh"

if [[ "$SETUP_SWAP" == "1" ]] && ! swapon --show | grep -q '/swapfile'; then
  fallocate -l "${SWAP_SIZE_GB}G" /swapfile
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  if ! grep -q '^/swapfile ' /etc/fstab; then
    echo '/swapfile none swap sw 0 0' >>/etc/fstab
  fi
fi

mkdir -p "$(dirname "$APP_DIR")"
if [[ ! -d "$APP_DIR/.git" ]]; then
  if [[ -z "$REPO_URL" ]]; then
    mkdir -p "$APP_DIR"
  else
    git clone --branch "$REPO_REF" "$REPO_URL" "$APP_DIR"
  fi
fi
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$APP_DIR"

if [[ -f "$APP_DIR/deploy/systemd/idol-auth.service" ]]; then
  cp "$APP_DIR/deploy/systemd/idol-auth.service" /etc/systemd/system/
fi
if [[ -f "$APP_DIR/deploy/systemd/idol-auth-backup.service" ]]; then
  cp "$APP_DIR/deploy/systemd/idol-auth-backup.service" /etc/systemd/system/
fi
if [[ -f "$APP_DIR/deploy/systemd/idol-auth-backup.timer" ]]; then
  cp "$APP_DIR/deploy/systemd/idol-auth-backup.timer" /etc/systemd/system/
fi
systemctl daemon-reload

cat <<EOF
provision complete

host:
- deploy user: ${DEPLOY_USER}
- app dir: ${APP_DIR}
- swap: ${SWAP_SIZE_GB}G

next steps:
1. log in as ${DEPLOY_USER}
2. cd ${APP_DIR}
3. cp .env.production.example .env.production
4. edit .env.production with real domains and secrets
5. ./scripts/run-nix-app.sh deploy-production .env.production
6. sudo systemctl enable --now idol-auth.service idol-auth-backup.timer
EOF
