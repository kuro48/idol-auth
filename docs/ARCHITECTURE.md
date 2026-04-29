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
- アプリ登録・OIDC クライアント発行・ユーザー管理の**コントロールプレーン API**

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

<!-- AUTO-GENERATED from internal/http/router.go -->

### ヘルスチェック

| Method | Path | 説明 |
|--------|------|------|
| GET | `/healthz` | 常に 200 OK を返す liveness probe |
| GET | `/readyz` | DB 接続を確認する readiness probe（タイムアウト 2s） |

### Auth API (`/v1/auth/*`)

レート制限付き（`RouterConfig.Limiter` が設定されている場合）。

| Method | Path | 説明 |
|--------|------|------|
| GET | `/v1/auth/providers` | Kratos / Hydra の各ブラウザフロー URL を返す |
| GET | `/v1/auth/session` | 現在の Kratos セッション情報を返す |
| POST | `/v1/auth/logout` | Hydra logout URL を返す |
| GET | `/v1/auth/logout/start` | Hydra logout URL へブラウザリダイレクト |
| GET | `/v1/auth/logout/callback` | Hydra logout_challenge を処理してリダイレクト |
| GET | `/v1/auth/login` | Hydra login_challenge を処理してリダイレクト |
| GET | `/v1/auth/consent` | Hydra consent_challenge を処理（確認画面またはリダイレクト） |
| POST | `/v1/auth/consent` | コンセント確認フォームの送信（action=accept\|deny） |

### Admin API (`/v1/admin/*`)

すべて認証必須。認証方式は後述。

| Method | Path | 説明 |
|--------|------|------|
| GET | `/v1/admin/apps` | アプリ一覧 |
| POST | `/v1/admin/apps` | アプリ作成 |
| GET | `/v1/admin/apps/{appID}/clients` | OIDC クライアント一覧 |
| POST | `/v1/admin/apps/{appID}/clients` | OIDC クライアント作成 |
| GET | `/v1/admin/users` | ユーザー検索（`?identifier=`, `?state=active\|inactive`） |
| POST | `/v1/admin/users/{identityID}/disable` | ユーザー無効化 |
| POST | `/v1/admin/users/{identityID}/enable` | ユーザー再有効化 |
| POST | `/v1/admin/users/{identityID}/revoke-sessions` | ユーザーの全セッション失効 |
| DELETE | `/v1/admin/users/{identityID}` | ユーザー削除 |
| PUT | `/v1/admin/identities/{identityID}/roles` | ロール設定 |
| GET | `/v1/admin/audit-logs` | 監査ログ一覧（フィルタ可） |

<!-- /AUTO-GENERATED -->

---

## Admin API 認証

2 種類の認証を受け付ける。

### Bootstrap Token（完全権限）

```
Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>
```

- GET / POST / PUT / DELETE すべて可能
- 本番では厳重に管理し、ローテーション必須

### Session 認証（読み取り限定）

- Kratos セッション Cookie を送信
- **MFA 必須**（`authenticator_assurance_level = aal2`）
- `ADMIN_ALLOWED_EMAILS` または `ADMIN_ALLOWED_ROLES` のいずれかに一致すること
- 書き込み系操作（POST / PUT / DELETE）は不可（Bootstrap Token が必要）

---

## データモデル

<!-- AUTO-GENERATED from internal/infra/db/migrations/ -->

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

<!-- /AUTO-GENERATED -->

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
