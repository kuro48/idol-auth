#!/usr/bin/env bash
# Sets up a GitHub Actions self-hosted runner on this machine as a launchd service.
# Run this script ONCE on the production Mac Mini.
#
# Usage:
#   ./setup-gh-runner.sh <github_repo_url> <registration_token>
#
# Where:
#   github_repo_url   = https://github.com/OWNER/REPO
#   registration_token = token from GitHub → Settings → Actions → Runners → New self-hosted runner
#
# Example:
#   ./setup-gh-runner.sh https://github.com/kuro48/idol-auth ghp_XXXXXXXXXXXX
set -euo pipefail

REPO_URL="${1:?Usage: $0 <github_repo_url> <registration_token>}"
REG_TOKEN="${2:?Usage: $0 <github_repo_url> <registration_token>}"

RUNNER_DIR="$HOME/actions-runner"
RUNNER_VERSION="2.323.0"
RUNNER_ARCH="osx-arm64"
RUNNER_TAR="actions-runner-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz"
RUNNER_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_TAR}"

LAUNCHD_LABEL="com.github.actions.runner"
LAUNCHD_PLIST="$HOME/Library/LaunchAgents/${LAUNCHD_LABEL}.plist"

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
  --name "mac-mini" \
  --labels "self-hosted,macOS,ARM64" \
  --work "$RUNNER_DIR/_work" \
  --unattended \
  --replace

echo "==> Installing launchd service"
cat > "$LAUNCHD_PLIST" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${LAUNCHD_LABEL}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${RUNNER_DIR}/run.sh</string>
  </array>
  <key>WorkingDirectory</key>
  <string>${RUNNER_DIR}</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>${RUNNER_DIR}/runner.log</string>
  <key>StandardErrorPath</key>
  <string>${RUNNER_DIR}/runner.log</string>
</dict>
</plist>
PLIST

launchctl unload "$LAUNCHD_PLIST" 2>/dev/null || true
launchctl load "$LAUNCHD_PLIST"

echo ""
echo "Self-hosted runner installed and started."
echo "Logs: tail -f $RUNNER_DIR/runner.log"
echo ""
echo "Next: set DEPLOY_ENV_FILE variable in GitHub repo settings:"
echo "  GitHub → Settings → Variables (not Secrets) → Actions → New repository variable"
echo "  Name:  DEPLOY_ENV_FILE"
echo "  Value: /path/to/.env  (full path to the .env file on this machine)"
