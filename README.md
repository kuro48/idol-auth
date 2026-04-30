# idol-auth

[![CI](https://github.com/kuro48/idol-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/kuro48/idol-auth/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](go.mod)

**OAuth2/OIDC 認証基盤**。Ory Hydra + Kratos を使い、複数アプリで 1 つの ID プールを共有するための認証サーバーです。

このリポジトリには次の 2 つが入っています。

- Hydra の login / consent / logout を処理する認証ブリッジ API
- アプリ登録、OIDC クライアント発行、共有アカウント連携、監査ログ取得を行う API 群

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
| API Docs (Swagger UI) | http://localhost:8080/docs |
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

- 開発者向け API サイト: `http://localhost:8080/docs`
- Swagger JSON: `http://localhost:8080/docs/doc.json`
- [アーキテクチャ](docs/ARCHITECTURE.md)
- [デプロイと運用](docs/deployment.md)
- [セキュリティポリシー](SECURITY.md)
- [ライセンス](LICENSE)

## Shared Account Model

- Kratos identity が共有アカウント本体です。複数アプリで同じ identity を使います。
- 各アプリの「登録」は identity 作成そのものではなく、初回 login / consent 時に `app membership` を作る動きです。
- 各アプリの「削除」は `DELETE /v1/apps/self/users/{identityID}` で membership を無効化します。共有アカウント本体は残ります。
- 共有アカウント本体の完全削除は `POST /v1/account/deletion` で中央管理します。
- app-scoped API は app 作成時または `POST /v1/admin/apps/{appID}/management-token` で発行される `management_token` を使います。

## 本番デプロイ

本番用の Ory 設定はテンプレートから生成します。

```bash
cp .env.production.example .env.production
# 設定生成・検証
make nix-render-production-config
make nix-config-check
# デプロイ
make nix-deploy-production
```

詳細は [docs/deployment.md](docs/deployment.md) を参照してください。
