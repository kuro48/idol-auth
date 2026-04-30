# アーキテクチャ詳細

## 概要

idol-auth は「共有アカウント型」認証基盤。複数のアプリが 1 つの Kratos ID プールと Hydra OAuth2 サーバーを共有し、各アプリに対して独立した OIDC クライアントを発行する。

```
ブラウザ / SPA / ネイティブアプリ
       │
       │  OAuth2 PKCE / authorization_code
       ▼
┌──────────────────────────────────────────┐
│              Hydra (ORY)                 │  :4444 public / :4445 admin
│  トークン発行・イントロスペクション         │
└──────────┬───────────────────────────────┘
           │ login_challenge / consent_challenge
           ▼
┌──────────────────────────────────────────┐
│           idol-auth (app)                │  :8080
│  ┌─────────────────┐  ┌───────────────┐  │
│  │  /v1/auth/*     │  │  /v1/admin/*  │  │
│  │  認証ブリッジ    │  │ コントロール  │  │
│  └────────┬────────┘  └───────┬───────┘  │
└───────────┼────────────────────┼──────────┘
            │                    │
            ▼                    ▼
┌─────────────────┐   ┌──────────────────────┐
│  Kratos (ORY)   │   │  PostgreSQL           │
│  ID・セッション  │   │  apps / oidc_clients  │
│  :4433 public   │   │  audit_logs 等        │
│  :4434 admin    │   └──────────────────────┘
└─────────────────┘
```

---

## コンポーネント

### Hydra
- OAuth2 / OIDC トークンサーバー
- login / consent / logout の各フローで idol-auth へリダイレクト
- Admin API (`/admin/oauth2/*`) 経由でクライアント CRUD

### Kratos
- ID・セッション・MFA・パスワードリセットを管理
- ブラウザフロー（登録・ログイン・設定）の UI はデモアプリまたは外部 UI が担当
- `metadata_public.roles` にロールを保存

### idol-auth (app)
- Hydra からの challenge を受け取り、Kratos セッションを検証して accept/reject する**ブリッジ**
- アプリ登録・OIDC クライアント発行・共有アカウント連携管理の**コントロールプレーン API**

---

## 共有アカウントモデル

- `global identity`
  - Kratos identity が shared account 本体
  - メール、電話番号、パスワード、MFA、roles はここに集約
- `app membership`
  - 各 app から見た利用関係
  - 初回 login / consent 成功時に自動作成
  - app 側の「退会」は identity 本体削除ではなく membership の revoke
- `account center`
  - 現在の shared account から接続済み app を確認
  - app ごとの連携解除
  - shared account の完全削除予約

shared account を使う前提なので、third-party app に identity 本体の削除権限は渡さない。app には `management token` を発行し、`/v1/apps/self/*` で自分の membership だけ扱わせる。

---

## 認証フロー

### ログインフロー

```
クライアント → Hydra /oauth2/auth → /v1/auth/login?login_challenge=...
    │
    ▼ idol-auth
    1. Hydra.GetLoginRequest(challenge)
       skip=true かつ subject あり → AcceptLoginRequest → Hydra リダイレクト先に転送
    2. skip=false → Kratos.ToSession()
       セッションなし → Kratos ログイン UI へリダイレクト
       セッションあり → AcceptLoginRequest → Hydra リダイレクト先に転送
```

### コンセントフロー

```
Hydra → /v1/auth/consent?consent_challenge=...
    │
    ▼ idol-auth
    1. Hydra.GetConsentRequest(challenge)
       skip=true or client.skip_consent=true
         → Kratos セッションからロールを取得
         → AccessToken / IDToken に roles クレームを注入
         → AcceptConsentRequest → リダイレクト
    2. skip=false
       → セッションなし → Kratos ログイン UI へ
       → セッションあり → コンセント確認 HTML を返す (CSRF 付き)
         → POST /v1/auth/consent (action=accept|deny)
```

### ログアウトフロー

```
クライアント → Hydra /oauth2/sessions/logout → /v1/auth/logout?logout_challenge=...
    │
    ▼ idol-auth → Hydra.AcceptLogoutRequest → リダイレクト
```

---

## API リファレンス

エンドポイント一覧と request / response の一次情報は Swagger UI を参照。

- `http://localhost:8080/docs`
- `http://localhost:8080/docs/doc.json`

主要な責務は次の 3 系統。

- `/v1/auth/*`
  - Hydra login / consent / logout bridge
- `/v1/admin/*`
  - app 登録、OIDC client 発行、management token 発行、監査
- `/v1/account/*` と `/v1/apps/self/*`
  - shared account 本体の自己管理と、app-scoped membership 管理

---

## Admin API 認証

2 種類の認証を受け付ける。

### Bootstrap Token（完全権限）

```
Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>
```

- GET / POST / PATCH / DELETE すべて可能
- 本番では厳重に管理し、ローテーション必須

### Session 認証（読み取り限定）

- Kratos セッション Cookie を送信
- **MFA 必須**（`authenticator_assurance_level = aal2`）
- `ADMIN_ALLOWED_EMAILS` または `ADMIN_ALLOWED_ROLES` のいずれかに一致すること
- 書き込み系操作（POST / PATCH / DELETE）は不可（Bootstrap Token が必要）

### App Management Token

```
Authorization: Bearer <management_token>
```

- `POST /v1/admin/apps` の response または `POST /v1/admin/apps/{appID}/management-token` で発行
- `GET /v1/apps/self/users`
- `DELETE /v1/apps/self/users/{identityID}`

この token で触れるのは、その app に紐づく membership だけ。Kratos identity 本体の削除や global role 変更は不可。

---

## データモデル

```
apps
├── id            UUID PK
├── name          TEXT
├── slug          TEXT UNIQUE
├── type          TEXT  (web | spa | native | m2m)
├── party_type    TEXT  (first_party | third_party)
├── status        TEXT  (active | disabled)
├── description   TEXT
└── created_at / updated_at / created_by / updated_by  TIMESTAMPTZ / TEXT

oidc_clients
├── id                         UUID PK
├── app_id                     UUID FK → apps.id
├── hydra_client_id            TEXT UNIQUE
├── client_type                TEXT  (public | confidential)
├── status                     TEXT  (active | disabled | rotated)
├── token_endpoint_auth_method TEXT
├── pkce_required              BOOLEAN
└── created_at / updated_at / created_by / updated_by

app_management_tokens
├── id            UUID PK
├── app_id        UUID FK → apps.id
├── token_hash    TEXT UNIQUE
├── token_prefix  TEXT
├── status        TEXT  (active | rotated)
└── created_at / updated_at / created_by / updated_by

app_user_memberships
├── id            UUID PK
├── app_id        UUID FK → apps.id
├── identity_id   TEXT
├── status        TEXT  (active | revoked)
├── profile       JSONB
└── created_at / updated_at / created_by / updated_by

account_deletion_requests
├── id            UUID PK
├── identity_id   TEXT UNIQUE
├── status        TEXT  (scheduled | cancelled | completed)
├── reason        TEXT
├── requested_at  TIMESTAMPTZ
├── scheduled_for TIMESTAMPTZ
├── cancelled_at  TIMESTAMPTZ NULL
├── completed_at  TIMESTAMPTZ NULL
└── last_actor_id TEXT

oidc_client_redirect_uris   (client_id ごとのリダイレクト URI 正規化テーブル)
oidc_client_scopes          (client_id ごとのスコープ正規化テーブル)

audit_logs
├── id           UUID PK
├── event_type   TEXT  (app.created | oidc_client.created |
│                        identity.roles.updated | identity.disabled | identity.deleted)
├── actor_type   TEXT  (admin_client)
├── actor_id     TEXT  (メールアドレスまたは "bootstrap-admin")
├── target_type  TEXT  (app | client | user)
├── target_id    TEXT
├── result       TEXT  (success | failure)
├── metadata     JSONB
└── occurred_at  TIMESTAMPTZ
```

Kratos の ID データ（メール・パスワード・MFA・`metadata_public.roles`）は **Kratos の DB スキーマ（`search_path=kratos`）** に格納。idol-auth の DB は管理メタデータと監査ログのみ保持する。

---

## 環境変数

<!-- AUTO-GENERATED from internal/config/config.go + .env.example -->

### 必須

| 変数 | 説明 |
|-----|------|
| `DATABASE_URL` | PostgreSQL 接続文字列 |
| `KRATOS_PUBLIC_URL` | Kratos public API（コンテナ内部通信用） |
| `KRATOS_ADMIN_URL` | Kratos admin API（コンテナ内部通信用） |
| `HYDRA_PUBLIC_URL` | Hydra public API（コンテナ内部通信用） |
| `HYDRA_ADMIN_URL` | Hydra admin API（コンテナ内部通信用） |

### オプション（本番では実質必須）

| 変数 | デフォルト | 説明 |
|-----|-----------|------|
| `APP_ENV` | `development` | `production` にするとバリデーションが厳格化 |
| `APP_PORT` | `8080` | リスニングポート |
| `APP_BASE_URL` | `http://localhost:8080` | 本番では `https://` 必須 |
| `ADMIN_BOOTSTRAP_TOKEN` | `` | Admin API の Bootstrap Token。本番では必須 |
| `ADMIN_ALLOWED_EMAILS` | `` | カンマ区切りの管理者メール |
| `ADMIN_ALLOWED_ROLES` | `` | カンマ区切りの管理者ロール |
| `KRATOS_BROWSER_URL` | `http://localhost:4433` | ブラウザリダイレクト用 Kratos URL。本番では `https://` 必須 |
| `HYDRA_BROWSER_URL` | `http://localhost:4444` | ブラウザリダイレクト用 Hydra URL。本番では `https://` 必須 |
| `SESSION_COOKIE_SECURE` | `true` | 本番では `true` 必須 |
| `TRUSTED_PROXIES` | `` | カンマ区切り CIDR。本番では必須 |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`。本番では `debug` 不可 |

<!-- /AUTO-GENERATED -->

---

## マルチバイナリ構成

```
cmd/
├── server/      常時稼働する HTTP サーバー
├── migrate/     init container として 1 回だけ実行するマイグレーター
├── adminctl/    オペレーターが手動実行する管理 CLI (set-roles)
├── demo/        OIDC デモクライアント + Kratos UI（開発・ステージングのみ）
├── portal/      Kratos self-service UI（本番・ステージング）
└── configcheck/ 設定読み込みのみ行う軽量バリデーター（CI / startupProbe）
```

各 `cmd/` は独立した Docker イメージとしてビルドされる（`Dockerfile` の multi-stage `target` を参照）。  
本番イメージには `app` / `migrate` / `portal` を含める。

---

## レート制限

`/v1/auth/*` に in-memory スライディングウィンドウ方式のレート制限を適用。  
`RouterConfig.Limiter` に `RateLimiter` を渡すことで有効化（`nil` で無効）。

クライアント IP の解決優先順位: `X-Real-IP` → `X-Forwarded-For`（先頭 IP） → `RemoteAddr`  
ただし `X-Forwarded-*` を信頼するのは `TRUSTED_PROXIES` に含まれる送信元のみ。

---

## Production Assets

- `docker-compose.production.yml` は `demo` / `mailpit` を含まない本番向け構成
- `deploy/kratos/kratos.production.yml.tmpl` と `deploy/hydra/hydra.production.yml.tmpl` は本番テンプレート
- `scripts/render-production-config.sh` で `dist/kratos/kratos.yml` と `dist/hydra/hydra.yml` を生成する

---

## セキュリティヘッダー

全レスポンスに自動付与:

```
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```

コンセント確認ページには追加で:

```
Cache-Control: no-store
Content-Security-Policy: default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; style-src 'unsafe-inline'
```

---

## ローカル開発

```bash
# 依存サービスを起動
docker compose up postgres redis kratos hydra mailpit

# アプリをローカルで起動
DATABASE_URL=postgres://idol:change_me_postgres@localhost:5432/idol_auth \
  KRATOS_PUBLIC_URL=http://localhost:4433 \
  KRATOS_ADMIN_URL=http://localhost:4434 \
  HYDRA_PUBLIC_URL=http://localhost:4444 \
  HYDRA_ADMIN_URL=http://localhost:4445 \
  ADMIN_BOOTSTRAP_TOKEN=<dev-bootstrap-token> \
  go run ./cmd/server

# マイグレーション単体実行
DATABASE_URL=postgres://idol:change_me_postgres@localhost:5432/idol_auth \
  go run ./cmd/migrate

# 設定チェックのみ
DATABASE_URL=... KRATOS_PUBLIC_URL=... go run ./cmd/configcheck
```
