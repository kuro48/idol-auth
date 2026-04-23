#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

DEPLOY_USER="${DEPLOY_USER:-deploy}"
INSTALL_DOCKER="${INSTALL_DOCKER:-1}"
ENABLE_UFW="${ENABLE_UFW:-1}"

apt-get update
apt-get install -y ca-certificates curl git gnupg make sudo ufw xz-utils

if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  adduser --disabled-password --gecos "" "$DEPLOY_USER"
fi
usermod -aG sudo "$DEPLOY_USER"

if [[ "$ENABLE_UFW" == "1" ]]; then
  ufw allow OpenSSH
  ufw allow 80/tcp
  ufw allow 443/tcp
  ufw --force enable
fi

if [[ "$INSTALL_DOCKER" == "1" ]] && ! command -v docker >/dev/null 2>&1; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    >/etc/apt/sources.list.d/docker.list
  apt-get update
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

if getent group docker >/dev/null 2>&1; then
  usermod -aG docker "$DEPLOY_USER"
fi

if ! command -v nix >/dev/null 2>&1; then
  su - "$DEPLOY_USER" -c 'bash -lc "curl -L https://nixos.org/nix/install | sh -s -- --daemon"'
fi

cat <<EOF
bootstrap complete

next steps:
1. log in again as ${DEPLOY_USER}
2. cd /opt/idol-auth
3. nix --extra-experimental-features "nix-command flakes" develop
4. nix --extra-experimental-features "nix-command flakes" run .#deploy-production -- .env.production
EOF
