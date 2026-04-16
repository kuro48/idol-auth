# Project Structure and Tech Stack

## 1. Purpose

この文書は初期リリース向けの Go アプリ構成と採用技術を定義する。

対象:

- ディレクトリ構成
- ライブラリ選定
- 設定管理
- DB migration 方針
- テスト方針

## 2. Design Priorities

- 認証本体を自前実装しない
- 保守しやすい最小構成にする
- 境界を明確にし、Ory 依存を閉じ込める
- テスト可能な設計にする
- 初期構築が重くなりすぎないようにする

## 3. Recommended Go Stack

### 3.1 HTTP

- Router: `github.com/go-chi/chi/v5`
- Rationale:
  - 標準 `net/http` に近く薄い
  - ミドルウェア構成が明快
  - 学習コストが低い

### 3.2 Configuration

- `github.com/kelseyhightower/envconfig` または `github.com/caarlos0/env/v11`
- Recommendation: `caarlos0/env/v11`
- Rationale:
  - 環境変数ベースで十分
  - struct で型安全に読み込める

### 3.3 Logging

- 標準 `log/slog`
- Rationale:
  - Go 標準で十分
  - structured logging を無理なく導入できる

### 3.4 Database Access

- PostgreSQL driver: `github.com/jackc/pgx/v5`
- Query tool: `github.com/jackc/pgx/v5/pgxpool`
- Recommendation:
  - ORM は使わず、初期リリースは SQL ベース
- Rationale:
  - control plane のテーブル数は少ない
  - SQL を明示した方が監査・保守しやすい

### 3.5 Migrations

- `github.com/golang-migrate/migrate/v4`
- Rationale:
  - Go から扱いやすい
  - Docker / CI でも扱いやすい

### 3.6 Validation

- `github.com/go-playground/validator/v10`
- Rationale:
  - admin API の入力検証を明示できる

### 3.7 ID generation

- UUID: `github.com/google/uuid`

### 3.8 HTTP client for Ory integration

- まずは標準 `net/http`
- 必要があれば Ory 公式 SDK を限定導入
- Recommendation:
  - 初期は Ory SDK の全面依存を避け、薄い adapter を自作する
- Rationale:
  - 自前認証ではないが、外部 API 依存は adapter 層で閉じたい

### 3.9 Testing

- 標準 `testing`
- `github.com/stretchr/testify`
- integration 用に Docker Compose

## 4. Recommended Directory Layout

```text
idol-auth/
  cmd/
    server/
      main.go
  internal/
    app/
      bootstrap.go
    config/
      config.go
    http/
      router.go
      middleware/
      handler/
    domain/
      app/
      admin/
      audit/
      client/
    usecase/
      app/
      admin/
      auth/
    infra/
      db/
        migrations/
        repository/
      ory/
        kratos/
        hydra/
      logging/
      clock/
      idgen/
    authz/
      policy/
    platform/
      health/
  api/
    openapi/
  deploy/
    docker/
    kratos/
    hydra/
  scripts/
  tests/
    integration/
  plans/
```

## 5. Layer Responsibilities

### 5.1 `cmd/server`

- エントリポイント
- config 読み込み
- DI 初期化
- HTTP server 起動

### 5.2 `internal/config`

- 環境変数読み込み
- 設定 struct 定義
- 起動時検証

### 5.3 `internal/http`

- router
- middleware
- request / response binding
- handler

### 5.4 `internal/domain`

- エンティティ
- 値オブジェクト
- ドメインルール

### 5.5 `internal/usecase`

- ユースケース単位のアプリケーションロジック
- repository / Ory adapter の組み合わせ

### 5.6 `internal/infra`

- PostgreSQL access
- Ory integration adapter
- logging
- time / UUID などの技術依存

### 5.7 `internal/authz`

- admin API 認可ポリシー
- role 判定

## 6. API Organization

初期リリースでは API を 2 系統に分ける。

- public support endpoints
- admin endpoints

例:

- `/healthz`
- `/readyz`
- `/v1/auth/session`
- `/v1/auth/logout`
- `/v1/admin/...`

Hydra / Kratos 本体エンドポイントは別サービスとして扱う。

## 7. Config Model

環境変数の責務を分ける。

### 7.1 App config

- `APP_ENV`
- `APP_PORT`
- `APP_BASE_URL`

### 7.2 Database config

- `DATABASE_URL`

### 7.3 Redis config

- `REDIS_ADDR`
- `REDIS_PASSWORD`

### 7.4 Ory config

- `KRATOS_PUBLIC_URL`
- `KRATOS_ADMIN_URL`
- `HYDRA_PUBLIC_URL`
- `HYDRA_ADMIN_URL`

### 7.5 Security config

- `SESSION_COOKIE_SECURE`
- `TRUSTED_PROXIES`
- `ADMIN_JWT_ISSUER` or equivalent

### 7.6 Audit / observability

- `LOG_LEVEL`
- `OTEL_EXPORTER_OTLP_ENDPOINT`

## 8. Storage Strategy

### 8.1 PostgreSQL

用途:

- app registry
- client metadata
- admin role data
- audit logs

### 8.2 Redis

用途候補:

- rate limit
- short-lived operation state
- logout / revoke 補助

初期リリースで無理に多用途化しない。

## 9. Ory Integration Structure

推奨パッケージ:

- `internal/infra/ory/kratos`
- `internal/infra/ory/hydra`

各パッケージは以下を持つ。

- client
- request/response mapping
- error mapping
- minimal operations interface

重要:

- Ory の API 型を usecase 層へ漏らしすぎない
- Go アプリの内部モデルと外部 API モデルを分離する

## 10. Testing Strategy

### 10.1 Unit tests

対象:

- 入力検証
- ドメインルール
- usecase
- 認可ポリシー

### 10.2 Integration tests

対象:

- DB repository
- Ory adapter
- admin API

### 10.3 End-to-end verification

対象:

- registration
- first login with TOTP enrollment
- SSO login across two apps
- admin user disable

## 11. Non-Recommendations

初期リリースでは以下を避ける。

- 重い ORM 導入
- CQRS の過剰分割
- microservices 化
- 独自 JWT 実装
- 独自 session 実装
- 複雑な plugin 機構

## 12. Recommended Next Step

次は開発用の実行環境を定義する。

- Docker Compose のサービス構成
- volumes
- ports
- migration 起動順
- local development flow
