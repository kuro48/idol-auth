# Auth Platform Blueprint

## Goal

複数アプリから共通利用できるアカウント認証 API を Go で提供する。

ただし、認証・認可の中核ロジックは自前実装せず、実績のある OSS / ライブラリに委譲する。

## Recommended Architecture

- API Gateway / Auth Facade: Go
- Identity / Login / MFA / Passkey: Ory Kratos
- OAuth2 / OIDC / Client management / Token issuance: Ory Hydra
- Database: PostgreSQL
- Cache / rate limit backend: Redis
- Reverse proxy / TLS termination: Caddy or Nginx
- Observability: OpenTelemetry + Prometheus + structured logs

## Core Design Principles

- パスワードハッシュ、セッション、CSRF、OIDC、OAuth2 トークン発行は自前実装しない
- ブラウザ向け認証は cookie ベースの browser flow を使い、SPA でも CSRF 保護を壊さない
- 各アプリは OAuth2/OIDC client として登録し、認証基盤を直接共有しない
- Go アプリは「認証ロジック本体」ではなく「統合 API / 管理 API / ドメイン制御」に集中する
- 機密設定はすべて環境変数・secret manager から注入する
- トークンは短命 access token + refresh token rotation 前提で運用する

## Delivery Plan

### Step 1: Foundations

- Go モジュール初期化
- `chi` または `gin` のどちらかに統一
- `config`, `http`, `domain`, `infra`, `internal` の基本構成を作る
- Docker Compose で `postgres`, `redis`, `kratos`, `hydra`, `migrate`, `app` を起動可能にする
- `.env.example` と secrets の読み込み基盤を用意する

Exit criteria:
- `docker compose up` で基盤コンテナが起動する
- Go アプリが health check を返す

### Step 2: Identity Base

- Ory Kratos の self-service flows を有効化する
- identity schema を設計する
- email/password を有効化する
- email verification / recovery を有効化する
- 管理 API から identity を検索・停止・削除できる管理レイヤを Go 側に作る

Exit criteria:
- 登録・ログイン・ログアウト・メール検証・パスワード再設定が通る
- identity schema に沿ったプロフィール項目が保存される

### Step 3: OAuth2 / OIDC Base

- Ory Hydra を Kratos と接続する
- login / consent ハンドラを Go で実装する
- 最初の first-party client を登録する
- `authorization_code + PKCE` を標準フローにする
- `client_credentials` は machine-to-machine 用に限定する

Exit criteria:
- Web アプリが OIDC ログインできる
- access token / refresh token / ID token が発行される

### Step 4: Auth Facade API

- Go 側に公開 API を定義する
- 候補:
  - `POST /admin/apps`
  - `POST /admin/apps/{id}/clients`
  - `GET /admin/users`
  - `POST /admin/users/{id}/disable`
  - `POST /admin/users/{id}/force-logout`
  - `GET /.well-known/openid-configuration` は Hydra に委譲
- 管理 API と公開 API の認可境界を分離する

Exit criteria:
- 各アプリのクライアント登録と利用制御が API 経由で可能
- 管理 API は管理者権限なしで実行できない

### Step 5: Security Hardening

- rate limiting
- audit log
- CSRF / CORS / secure cookie 設定
- trusted proxies / HTTPS 前提設定
- brute force / enumeration 対策
- refresh token rotation の設定
- 監査イベントの保存

Exit criteria:
- セキュリティ設定がコード・設定ファイルに明示される
- 主要な不正アクセス系テストが追加される

### Step 6: Multi-App Support

- アプリごとの redirect URI 制約
- スコープ設計
- tenant 的なアプリ境界の扱いを決める
- first-party / third-party client の区別を導入する
- アプリ停止時のクライアント失効フローを作る

Exit criteria:
- 複数アプリを同一認証基盤に安全に収容できる
- クライアント単位の権限・失効・監査が可能

### Step 7: Operations

- migration 手順確立
- バックアップ / リストア
- 監視・アラート
- structured logging
- インシデント時の token / client / session revoke 手順を整備

Exit criteria:
- 運用 runbook がある
- 障害時に revoke と復旧の手順が明文化される

## Security Requirements

- ブラウザアプリは Kratos の browser flow を使う
- SPA / server-side app で native API flow を誤用しない
- machine-to-machine のみ `client_credentials` を許可する
- public client は PKCE 必須
- confidential client は secret の保管場所を限定する
- MFA は初期段階から設計対象に含める
- passkey は後付けではなく拡張可能な前提で schema / UI を作る
- 管理 API は一般ユーザー認証系と別の認可ポリシーで守る

## First Implementation Slice

最初の着手は以下が適切:

1. Docker Compose で Kratos / Hydra / Postgres / Redis / Go app を起動
2. Go アプリの雛形と health check
3. Kratos identity schema と self-service flow の疎通
4. Hydra login / consent の最小実装
5. サンプルクライアント 1 個で OIDC ログイン確認

## Decision Log

- 認証本体を Go の自作コードで持たない
- Go は auth facade と管理 API に責務を限定する
- 中核 OSS は Ory 系を第一候補とする
