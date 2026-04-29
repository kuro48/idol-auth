# デプロイと運用

このドキュメントは、本番公開やステージング確認に必要な手順を 1 か所にまとめたものです。

## 前提構成

- 1 Linux VM
- Docker Engine / Docker Compose
- `docker-compose.production.yml`
- `Caddy` を唯一の公開入口として利用
- 公開ポートは `80/443` のみ
- `hydra admin`, `kratos admin`, `postgres`, `redis` は内部ネットワーク限定

## 1. 本番用設定を作る

```bash
cp .env.production.example .env.production
```

最低限、次を埋めます。

- `APP_BASE_URL`
- `APP_HOSTNAME`
- `PORTAL_APP_URL`
- `PORTAL_HOSTNAME`
- `HYDRA_HOSTNAME`
- `ACME_EMAIL`
- `POSTGRES_PASSWORD`
- `REDIS_PASSWORD`
- `ADMIN_BOOTSTRAP_TOKEN`
- `KRATOS_SECRETS_*`
- `HYDRA_SYSTEM_SECRET`
- `KRATOS_SMTP_CONNECTION_URI`
- `ADMIN_ALLOWED_CIDR`
- `TRUSTED_PROXIES`

公開前チェック:

- `APP_ENV=production`
- `APP_BASE_URL`, `KRATOS_BROWSER_URL`, `HYDRA_BROWSER_URL` は `https://`
- `SESSION_COOKIE_SECURE=true`
- `LOG_LEVEL` は `debug` ではない
- `ADMIN_BOOTSTRAP_TOKEN` は 32 文字以上のランダム値

## 2. 本番用 Ory 設定を生成する

```bash
set -a; . ./.env.production; set +a
make render-production-config
make config-check
docker compose -f docker-compose.production.yml config
```

生成物は `dist/kratos/kratos.yml` と `dist/hydra/hydra.yml` です。リポジトリにはコミットしません。

## 3. デプロイする

```bash
docker compose -f docker-compose.production.yml up -d --build
```

初回に Linux VPS を整える必要がある場合は、リポジトリ内のプロビジョニングスクリプトを参照してください。

## 4. 公開前に確認すること

### Admin API の到達制御

`ADMIN_ALLOWED_CIDR` で `/v1/admin/*` を operator ネットワークに制限します。

例:

```bash
ADMIN_ALLOWED_CIDR=203.0.113.10/32
ADMIN_ALLOWED_CIDR=10.8.0.0/24
```

確認:

```bash
curl -sf -H "Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>" \
  https://<APP_HOSTNAME>/v1/admin/apps
```

別ネットワークからは `403` になることを確認します。

### ヘルスチェック

```bash
curl -fsS https://<APP_HOSTNAME>/healthz
curl -fsS https://<APP_HOSTNAME>/readyz
```

ホスト内部からは:

```bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/readyz
```

### Smoke / E2E

ローカルやステージングでは最低限これを通します。

```bash
make up
make wait
./scripts/smoke-local-auth.sh
make down
```

確認対象:

- 新規登録
- MFA enrollment
- first-party OIDC
- partner consent
- ログアウト
- admin disable / enable
- session revoke
- audit log read

### セキュリティ確認

```bash
go test ./...
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## 運用

### 日常デプロイ

```bash
git pull
ENV_FILE=.env.production ./scripts/deploy-production.sh
```

### バックアップ

```bash
./scripts/backup-postgres.sh .env.production
```

systemd unit:

- `deploy/systemd/idol-auth.service`
- `deploy/systemd/idol-auth-backup.service`
- `deploy/systemd/idol-auth-backup.timer`

### リストア

```bash
gunzip -c backups/idol_auth_<timestamp>.sql.gz | \
  docker compose --env-file .env.production -f docker-compose.production.yml \
  exec -T postgres psql -U idol -d idol_auth
```

その後に login, MFA, OIDC を再確認します。

### ロールバック

```bash
git fetch
git checkout <tag-or-commit>
./scripts/run-nix-app.sh deploy-production .env.production
```

### ブレークグラス

`ADMIN_BOOTSTRAP_TOKEN` を緊急ローテーションする場合:

```bash
openssl rand -hex 32
```

`.env.production` を更新して app コンテナだけ再起動します。

```bash
docker compose -f docker-compose.production.yml --env-file .env.production \
  up -d --no-deps --build app
```

## 監視の最低ライン

- `/healthz` の外形監視
- `/readyz` の外形監視
- container restart
- disk usage
- repeated auth failures
- SMTP delivery failures

無料構成なら UptimeRobot などで十分です。
