# idol-auth

Ory スタック（Hydra + Kratos）をベースにした OAuth2/OIDC 認証基盤。
アプリ登録・クライアント発行・ロール管理・監査ログを一元管理するコントロールプレーンと、
ログイン/コンセント/ログアウトフローを処理するブリッジサーバーを提供する。

## クイックスタート

```bash
cp .env.example .env        # シークレットを編集
docker compose up --build   # Postgres / Redis / Kratos / Hydra / App / Demo を起動
```

| サービス | URL |
|---------|-----|
| アプリ API | http://localhost:8080 |
| Demo UI | http://localhost:3002 |
| Mailpit (メール確認) | http://localhost:8025 |
| Hydra (public) | http://localhost:4444 |
| Kratos (public) | http://localhost:4433 |

## コマンド一覧

<!-- AUTO-GENERATED from cmd/ -->
| コマンド | 用途 | 実行タイミング |
|---------|------|--------------|
| `go run ./cmd/server` | HTTP サーバー起動 | 常時稼働 |
| `go run ./cmd/migrate` | DB マイグレーション実行 | デプロイ時 1 回 |
| `go run ./cmd/adminctl set-roles` | ID にロールを付与する CLI | 管理者が随時 |
| `go run ./cmd/demo` | OIDC デモクライアント + Kratos UI | 開発・ステージング |
| `go run ./cmd/portal` | 本番向け Kratos self-service UI | 常時稼働 |
| `go run ./cmd/configcheck` | 設定バリデーションのみ実行 | CI / startupProbe |
<!-- /AUTO-GENERATED -->

## テスト

```bash
go test ./...
go test -race ./...
```

## Nix

`Nix` で開発ツールと運用 CLI を固定できる。

```bash
nix --extra-experimental-features "nix-command flakes" develop
make test
```

本番系の操作も `nix run` で実行できる。

```bash
nix --extra-experimental-features "nix-command flakes" run .#render-production-config
nix --extra-experimental-features "nix-command flakes" run .#config-check
nix --extra-experimental-features "nix-command flakes" run .#deploy-production -- .env.production
```

`Ubuntu on Sakura VPS` では、初回だけ `root` で provisioning script を実行し、その後は `run-nix-app.sh` に寄せる。

```bash
sudo ./scripts/provision-sakura-vps.sh
./scripts/run-nix-app.sh deploy-production .env.production
```

## ドキュメント

- [アーキテクチャ詳細](docs/ARCHITECTURE.md)
- [リリースチェックリスト](docs/release-checklist.md)

## Production

本番用の Ory 設定は開発用ファイルと分離している。

```bash
cp .env.production.example .env.production
set -a; . ./.env.production; set +a
make render-production-config
make config-check
docker compose -f docker-compose.production.yml up -d --build
```

`Nix` を使う場合は、同じ操作を `nix run` に置き換えられる。

```bash
cp .env.production.example .env.production
set -a; . ./.env.production; set +a
nix --extra-experimental-features "nix-command flakes" run .#render-production-config
nix --extra-experimental-features "nix-command flakes" run .#config-check
nix --extra-experimental-features "nix-command flakes" run .#deploy-production -- .env.production
```

`docker-compose.production.yml` は `portal` を含み、`demo` や `mailpit` は含まない。  
`dist/kratos/kratos.yml` と `dist/hydra/hydra.yml` は `scripts/render-production-config.sh` で生成する。

個人開発向けの本番構成は `Caddy + Docker Compose` を前提にしている。  
公開ポートは `80/443` のみで、`deploy/caddy/Caddyfile` が `app` / `portal` / `hydra public` へ中継する。

運用手順は [docs/operations.md](docs/operations.md) を参照。
