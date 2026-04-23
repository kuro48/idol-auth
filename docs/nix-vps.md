# Nix on Sakura VPS

## Goal

`Ubuntu` のまま `Nix` を追加し、この repo の開発・デプロイ用 CLI を `flake.nix` で固定する。
アプリの本番実行は引き続き `docker-compose.production.yml` を使う。

## Why this shape

- `NixOS` に寄せなくても `さくらVPS` で使いやすい
- `Docker` と `Caddy` の運用は変えなくてよい
- 開発機と VPS で同じ `Go` / `Make` / `Perl` / `docker compose` CLI を使える
- 将来ほかの VM やクラウドへ移すときも `flake.nix` をそのまま持ち出せる

## Unified bootstrap

`root` で一度だけ:

```bash
sudo REPO_URL=<your-repo-url> REPO_REF=main ./scripts/provision-sakura-vps.sh
```

この script は以下を行う。

- `deploy` ユーザー作成
- `ufw` で `22/80/443` を開放
- `Docker Engine` と `docker compose` 導入
- `Nix` daemon install
- `2GB swap` 作成
- `/opt/idol-auth` 配置
- `systemd` unit 配置

## Daily commands

開発用 shell:

```bash
nix --extra-experimental-features "nix-command flakes" develop
```

設定 render:

```bash
./scripts/run-nix-app.sh render-production-config
```

設定検証:

```bash
./scripts/run-nix-app.sh config-check
```

デプロイ:

```bash
./scripts/run-nix-app.sh deploy-production .env.production
```

DB backup:

```bash
./scripts/run-nix-app.sh backup-postgres .env.production
```

## What Nix manages

- `go`
- `make`
- `git`
- `curl`
- `jq`
- `openssl`
- `perl`
- `docker compose` client
- `pg_dump`
- `redis-cli`

## What Nix does not replace

- `Docker Engine` daemon
- `systemd`
- `Caddy` container runtime
- VPS の firewall と DNS
- `.env.production` の secret 管理

## Recommended production flow

1. `git pull`
2. `cp .env.production.example .env.production`
3. `.env.production` に実ドメインと secret を入れる
4. `./scripts/run-nix-app.sh deploy-production .env.production`
5. `sudo systemctl enable --now idol-auth.service idol-auth-backup.timer`
