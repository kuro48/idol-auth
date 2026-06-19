#!/usr/bin/env bash
# Sets up a GitHub Actions self-hosted runner on a Linux server as a systemd service.
# Run this script ONCE on the production server (as a non-root user with sudo access).
#
# Usage:
#   ./setup-gh-runner.sh <github_repo_url> <registration_token>
#
# Where:
#   github_repo_url   = https://github.com/OWNER/REPO
#   registration_token = token from GitHub → Settings → Actions → Runners → New self-hosted runner
#
# Example:
#   ./setup-gh-runner.sh https://github.com/kuro48/idol-auth AXXXXXXXXXXXXXXXXXX
set -euo pipefail

REPO_URL="${1:?Usage: $0 <github_repo_url> <registration_token>}"
REG_TOKEN="${2:?Usage: $0 <github_repo_url> <registration_token>}"

RUNNER_DIR="$HOME/actions-runner"
RUNNER_VERSION="2.323.0"
SERVICE_NAME="github-actions-runner"

# Detect CPU architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  RUNNER_ARCH="linux-x64" ;;
  aarch64) RUNNER_ARCH="linux-arm64" ;;
  armv7l)  RUNNER_ARCH="linux-arm" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

RUNNER_TAR="actions-runner-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz"
RUNNER_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_TAR}"

echo "==> Architecture: $ARCH → runner package: $RUNNER_ARCH"
echo "==> Creating runner directory: $RUNNER_DIR"
mkdir -p "$RUNNER_DIR"

if [[ ! -f "$RUNNER_DIR/run.sh" ]]; then
  echo "==> Downloading runner v${RUNNER_VERSION}"
  curl -sSL "$RUNNER_URL" -o "/tmp/$RUNNER_TAR"
  tar -xzf "/tmp/$RUNNER_TAR" -C "$RUNNER_DIR"
  rm "/tmp/$RUNNER_TAR"
fi

echo "==> Configuring runner"
"$RUNNER_DIR/config.sh" \
  --url "$REPO_URL" \
  --token "$REG_TOKEN" \
  --name "linux-server" \
  --labels "self-hosted,Linux,${ARCH}" \
  --work "$RUNNER_DIR/_work" \
  --unattended \
  --replace

echo "==> Installing systemd service"
RUNNER_USER="$(whoami)"
sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" > /dev/null <<SERVICE
[Unit]
Description=GitHub Actions self-hosted runner
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=${RUNNER_USER}
WorkingDirectory=${RUNNER_DIR}
ExecStart=${RUNNER_DIR}/run.sh
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${SERVICE_NAME}

[Install]
WantedBy=multi-user.target
SERVICE

sudo systemctl daemon-reload
sudo systemctl enable "${SERVICE_NAME}"
sudo systemctl restart "${SERVICE_NAME}"

echo ""
echo "Self-hosted runner installed and started."
echo "Status:  sudo systemctl status ${SERVICE_NAME}"
echo "Logs:    sudo journalctl -u ${SERVICE_NAME} -f"
echo ""
echo "Next: set DEPLOY_ENV_FILE variable in GitHub repo settings:"
echo "  GitHub → Settings → Variables (not Secrets) → Actions → New repository variable"
echo "  Name:  DEPLOY_ENV_FILE"
echo "  Value: /path/to/.env  (full path to the .env file on this server)"
