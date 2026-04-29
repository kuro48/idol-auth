# idol-auth

[![CI](https://github.com/kuro48/idol-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/kuro48/idol-auth/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](go.mod)

**OAuth2/OIDC 認証基盤**。Ory Hydra + Kratos を使い、複数アプリで 1 つの ID プールを共有するための認証サーバーです。

このリポジトリには次の 2 つが入っています。

- Hydra の login / consent / logout を処理する認証ブリッジ API
- アプリ登録、OIDC クライアント発行、ロール管理、監査ログ取得を行う Admin API

## 開発者向け概要

- 言語: Go
- 主要依存: `chi`, `pgx`, Ory Hydra, Ory Kratos
- ローカル起動: `docker compose` または `make up`
- 本番想定: `Caddy + Docker Compose`

## クイックスタート

前提:

- Docker / Docker Compose
- Go 1.25+（ローカルで `go test` する場合）

起動:

```bash
cp .env.example .env
make up
make wait
```

主要 URL:

| サービス | URL |
|---------|-----|
| API | http://localhost:8080 |
| Demo UI | http://localhost:3002 |
| Kratos public | http://localhost:4433 |
| Hydra public | http://localhost:4444 |
| Mailpit | http://localhost:8025 |

ローカル smoke:

```bash
./scripts/smoke-local-auth.sh
```

停止:

```bash
make down
```

## よく使うコマンド

```bash
make up
make wait
go test ./...
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run ./cmd/configcheck
```

## リポジトリの見方

- `cmd/server`: メイン API
- `cmd/demo`: ローカル開発用デモクライアント
- `cmd/portal`: 本番向け Kratos self-service UI
- `internal/http`: HTTP ルーティングと auth/admin フロー
- `internal/domain`: アプリ登録、管理操作、監査ログのドメイン
- `internal/infra`: DB、Hydra、Kratos 連携
- `integration`: E2E 相当の統合テスト

## ドキュメント

- [API リファレンス](docs/API.md)
- [アーキテクチャ](docs/ARCHITECTURE.md)
- [デプロイと運用](docs/deployment.md)
- [セキュリティポリシー](SECURITY.md)
- [ライセンス](LICENSE)

## 本番デプロイ

本番用の Ory 設定はテンプレートから生成します。

```bash
cp .env.production.example .env.production
set -a; . ./.env.production; set +a
make render-production-config
make config-check
docker compose -f docker-compose.production.yml up -d --build
```

詳細は [docs/deployment.md](docs/deployment.md) を参照してください。
