# API リファレンス

`idol-auth` は大きく 2 系統の API を提供します。

- `Auth API`: Ory Hydra の login / consent / logout を仲介するブラウザ向け API
- `Admin API`: アプリ登録、OIDC クライアント発行、ユーザー管理、監査ログ取得を行う管理 API

ローカル開発時のベース URL は `http://localhost:8080` です。

## 共通仕様

### Content-Type

- JSON を受け取るエンドポイントは `Content-Type: application/json`
- `POST /v1/auth/consent` は HTML フォーム送信のため `application/x-www-form-urlencoded`

### 認証

`/healthz` と `/readyz` 以外は用途ごとに認証前提が異なります。

- `Auth API`: Kratos セッション Cookie を前提に Hydra / Kratos と連携
- `Admin API`: `Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>` または管理者セッション

管理者セッションで `Admin API` を呼ぶ条件:

- Kratos セッションが有効
- MFA 済み (`authenticator_assurance_level = aal2`)
- `ADMIN_ALLOWED_EMAILS` または `ADMIN_ALLOWED_ROLES` に一致

管理者セッションで実行できるのは読み取り系のみです。`POST` / `PUT` / `DELETE` の変更系は `ADMIN_BOOTSTRAP_TOKEN` が必要です。

### エラーレスポンス

JSON エンドポイントのエラーは基本的に以下の形です。

```json
{
  "error": "message"
}
```

よく使う HTTP ステータス:

- `400 Bad Request`: JSON 不正、必須パラメータ不足、入力値不正
- `401 Unauthorized`: Admin API の認証なし
- `403 Forbidden`: MFA 不足、管理者権限なし、consent subject 不整合、CSRF 不正
- `404 Not Found`: 対象 app が存在しない
- `409 Conflict`: 無効化された app に対する client 作成
- `429 Too Many Requests`: bootstrap token の失敗が多すぎる
- `502 Bad Gateway`: Hydra / Kratos など upstream 連携失敗
- `503 Service Unavailable`: auth/admin service 未初期化、readyz 未準備

## ヘルスチェック

### `GET /healthz`

プロセスの liveness を返します。

レスポンス例:

```json
{
  "status": "ok"
}
```

### `GET /readyz`

依存サービスの readiness を返します。readiness checker が失敗すると `503` で `{"status":"not_ready"}` を返します。

レスポンス例:

```json
{
  "status": "ok"
}
```

## Auth API

`/v1/auth/*` は主に Hydra からブラウザリダイレクトで呼ばれる API です。JSON API とブラウザフロー向けのリダイレクト / HTML が混在します。

### `GET /v1/auth/providers`

Kratos / Hydra のブラウザフロー URL 一覧を返します。フロントエンドや Demo UI が各画面への導線を作るときに使います。

レスポンス例:

```json
{
  "login_url": "http://localhost:4433/self-service/login/browser",
  "registration_url": "http://localhost:4433/self-service/registration/browser",
  "recovery_url": "http://localhost:4433/self-service/recovery/browser",
  "verification_url": "http://localhost:4433/self-service/verification/browser",
  "settings_url": "http://localhost:4433/self-service/settings/browser",
  "logout_url": "http://localhost:4444/oauth2/sessions/logout"
}
```

### `GET /v1/auth/session`

現在の Kratos セッションを要約して返します。未認証なら `200` で `authenticated: false` を返します。

レスポンス例:

```json
{
  "authenticated": true,
  "subject": "8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73",
  "identity_id": "8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73",
  "email": "admin@example.com",
  "roles": ["admin"],
  "methods": ["password", "totp"],
  "authenticator_assurance_level": "aal2"
}
```

### `POST /v1/auth/logout`

Hydra の logout 開始 URL を返します。`Accept` ヘッダーでレスポンス形式を切り替えられます。

- `Accept: application/json`（または省略）→ `200` で JSON を返す
- `Accept: text/html` などブラウザ系 → `303 See Other` で Hydra logout URL へリダイレクト

JSON レスポンス例:

```json
{
  "logout_url": "http://localhost:4444/oauth2/sessions/logout"
}
```

### `GET /v1/auth/logout/start`

ブラウザ向けの logout 開始エイリアスです。常に `303 See Other` で Hydra logout URL へリダイレクトします。`<a href="/v1/auth/logout/start">` のようにリンクから遷移させるときに使います。

### `GET /v1/auth/logout/callback`

Hydra の `logout_challenge` を accept して `302` リダイレクトします。Hydra からのコールバック先として使います。

クエリパラメータ:

- `logout_challenge`: 必須

主なエラー:

- `400`: `logout_challenge` 不足
- `502`: Hydra 連携失敗

### `GET /v1/auth/logout` (非推奨)

`GET /v1/auth/logout/callback` の旧パスです。後方互換のため引き続き動作しますが、
レスポンスに `Deprecation: true` および `Sunset: 2027-05-01` ヘッダーが付与されます。
新規実装では `GET /v1/auth/logout/callback` を使ってください。

### `GET /v1/auth/login`

Hydra の `login_challenge` を処理します。

クエリパラメータ:

- `login_challenge`: 必須

挙動:

- Hydra 側で `skip=true` ならそのまま login accept して `302` リダイレクト
- Kratos セッションがなければ Kratos の login browser flow へ `302`
- セッションはあるが MFA 未完了なら Kratos settings flow へ `302`
- セッションが有効で MFA 済みなら Hydra login accept 後に `302`

主なエラー:

- `400`: `login_challenge` 不足
- `502`: Hydra / Kratos 連携失敗

### `GET /v1/auth/consent`

Hydra の `consent_challenge` を処理します。

クエリパラメータ:

- `consent_challenge`: 必須

挙動:

- `skip=true` または client が first-party の場合は自動 accept して `302`
- セッションが無ければ Kratos login flow へ `302`
- セッションはあるが MFA 未完了なら Kratos settings flow へ `302`
- third-party client で確認が必要な場合は HTML の consent 画面を返す
- consent request の `subject` と現在セッションが不一致なら `403`

自動 accept 時には、現在セッションの `roles` を正規化して Access Token / ID Token の claims に注入します。

### `POST /v1/auth/consent`

HTML consent 画面の送信先です。JSON ではなくフォーム送信です。

フォーム項目:

- `consent_challenge`: 必須
- `csrf_token`: 必須
- `action`: 必須。`accept` または `deny`

挙動:

- `GET /v1/auth/consent` が出した CSRF Cookie と `csrf_token` を照合
- `action=deny` なら Hydra に reject を返して `302`
- `action=accept` なら roles claim を付けて consent accept 後に `302`
- first-party / skip consent の場合も subject 整合性を検証してから自動 accept

主なエラー:

- `400`: challenge 不足、action 不正、フォーム不正
- `403`: CSRF 不正、subject 不整合
- `502`: Hydra / Kratos 連携失敗

### `GET /v1/auth/logout`

Hydra の `logout_challenge` を accept して `302` リダイレクトします。

クエリパラメータ:

- `logout_challenge`: 必須

主なエラー:

- `400`: `logout_challenge` 不足
- `502`: Hydra 連携失敗

## Admin API

`/v1/admin/*` はすべて認証必須です。

- 読み取り系 (`GET`): bootstrap token または管理者セッション
- 変更系 (`POST` / `PUT` / `DELETE`): bootstrap token 必須

### `GET /v1/admin/apps`

登録済みアプリ一覧を返します。

レスポンス例:

```json
{
  "items": [
    {
      "id": "4bba3a6f-19ea-4b2d-9341-61eeb31b9cc2",
      "name": "Demo Web App",
      "slug": "demo-web",
      "type": "web",
      "party_type": "first_party",
      "status": "active",
      "description": "Internal demo app",
      "created_at": "2026-04-25T10:00:00Z",
      "updated_at": "2026-04-25T10:00:00Z",
      "created_by": "bootstrap-admin",
      "updated_by": "bootstrap-admin"
    }
  ]
}
```

### `POST /v1/admin/apps`

新しい app を登録します。`201 Created` で app オブジェクトを返します。レスポンスには `Location: /v1/admin/apps/{id}` ヘッダーが付きます。

#### 最小リクエスト例

```json
{
  "name": "Demo Web App",
  "type": "web",
  "party_type": "first_party"
}
```

#### 入力ルール

| フィールド | 必須 | 説明 |
|---|---|---|
| `name` | ○ | 1-100 文字 |
| `slug` | △ | 英小文字・数字・`-` のみ。省略すると `name` から自動生成 |
| `type` | ○ | `web` / `spa` / `native` / `m2m`。エイリアス: `webapp`・`server` → `web`、`single-page`・`single_page` → `spa`、`mobile` → `native` |
| `party_type` | ○ | `first_party` / `third_party` |
| `description` | ✗ | 任意 |

> **Note**: slug は `name` から決定的に生成されます（`"My SPA"` → `"my-spa"`）。CI や IaC で冪等実行する場合は slug を明示してください。DB の unique 制約が重複を分かりやすいエラーで返します。

#### ワンコール登録（app + OIDC client）

`client` ブロック、または上位に `redirect_uris` を渡すと app 作成と同時に OIDC client も発行されます。

**`client` ブロック方式**（フル指定）:

```json
{
  "name": "Demo Web App",
  "type": "web",
  "party_type": "first_party",
  "client": {
    "name": "Demo Browser Client",
    "client_type": "confidential",
    "redirect_uris": ["https://example.com/callback"],
    "scopes": ["openid", "email"]
  }
}
```

**`redirect_uris` 省略形**（`client_type` / `name` は app から自動推論）:

```json
{
  "name": "My SPA",
  "type": "spa",
  "party_type": "first_party",
  "redirect_uris": ["https://example.com/callback"]
}
```

両方の方式でレスポンスは `app` / `client` / `client_secret` を含むオブジェクトになります:

```json
{
  "app": { "id": "...", "name": "My SPA", "slug": "my-spa", "type": "spa", "..." : "..." },
  "client": { "id": "...", "client_type": "public", "pkce_required": true, "..." : "..." },
  "client_secret": ""
}
```

`client_secret` は confidential client の初回発行時のみ値が入ります。

### `GET /v1/admin/apps/{appID}/clients`

指定 app に紐づく OIDC client 一覧を返します。

パスパラメータ:

- `appID`: UUID

レスポンス例:

```json
{
  "items": [
    {
      "id": "f4e3f55f-bb7a-4787-a4fb-b346b6dc1a82",
      "hydra_client_id": "demo-web-9f7a2b4c",
      "app_id": "4bba3a6f-19ea-4b2d-9341-61eeb31b9cc2",
      "client_type": "public",
      "status": "active",
      "token_endpoint_auth_method": "none",
      "pkce_required": true,
      "redirect_uris": ["http://localhost:3002/callback"],
      "post_logout_redirect_uris": ["http://localhost:3002/"],
      "scopes": ["email", "openid", "profile"],
      "created_at": "2026-04-25T10:05:00Z",
      "updated_at": "2026-04-25T10:05:00Z",
      "created_by": "bootstrap-admin",
      "updated_by": "bootstrap-admin"
    }
  ]
}
```

### `POST /v1/admin/apps/{appID}/clients`

指定 app 向けに OIDC client を作成します。`201 Created` でレスポンスを返し、`Location: /v1/admin/apps/{appID}/clients/{id}` ヘッダーが付きます。

リクエスト JSON:

```json
{
  "name": "Demo Browser Client",
  "redirect_uris": ["http://localhost:3002/callback"],
  "post_logout_redirect_uris": ["http://localhost:3002/"],
  "scopes": ["openid", "profile", "email"]
}
```

#### スマートデフォルト

省略可能なフィールドはサーバーが自動推論します:

| フィールド | 省略時の挙動 |
|---|---|
| `name` | 親 app の `name` を使用 |
| `client_type` | app type から推論: `spa`・`native` → `public`、`web`・`m2m` → `confidential` |
| `token_endpoint_auth_method` | `client_type` から推論: `public` → `none`、`confidential` → `client_secret_basic` |
| `scopes` | `["openid"]` |

> レスポンスの `client` オブジェクトには `client_type`・`token_endpoint_auth_method`・`pkce_required` が必ず含まれるため、推論結果を確認できます。

#### 入力ルール

- `redirect_uris`:
  - `web` / `spa` / `native` app では必須、`m2m` app では不可
  - `https://...` は常に許可
  - `http://...` は loopback host のみ許可
  - カスタム scheme は `native` app のみ許可
- `scopes`: `openid` は `web` / `spa` / `native` で必須

#### app type ごとのポリシー

- `web` / `spa` / `native`: grant types `authorization_code` + `refresh_token`、PKCE 必須
- `m2m`: grant type `client_credentials`、`confidential` client のみ

first-party app から作られた client は Hydra 側で `skip_consent=true` になります。

レスポンス例:

```json
{
  "client": {
    "id": "f4e3f55f-bb7a-4787-a4fb-b346b6dc1a82",
    "hydra_client_id": "demo-web-9f7a2b4c",
    "app_id": "4bba3a6f-19ea-4b2d-9341-61eeb31b9cc2",
    "client_type": "public",
    "status": "active",
    "token_endpoint_auth_method": "none",
    "pkce_required": true,
    "redirect_uris": ["http://localhost:3002/callback"],
    "post_logout_redirect_uris": ["http://localhost:3002/"],
    "scopes": ["email", "openid", "profile"],
    "created_at": "2026-04-25T10:05:00Z",
    "updated_at": "2026-04-25T10:05:00Z",
    "created_by": "bootstrap-admin",
    "updated_by": "bootstrap-admin"
  },
  "client_secret": ""
}
```

`client_secret` は confidential client の初回発行時のみ値が入ります。public client では空文字です。

### `GET /v1/admin/users`

Kratos identity を検索します。

クエリパラメータ:

- `identifier`: メールアドレスや電話番号などの識別子で部分一致検索
- `state`: `active` または `inactive`

レスポンス例:

```json
{
  "items": [
    {
      "id": "8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73",
      "schema_id": "default",
      "state": "active",
      "email": "user@example.com",
      "phone": "",
      "primary_identifier_type": "email",
      "roles": ["viewer"]
    }
  ]
}
```

### `{userRef}` について

`/v1/admin/users/{userRef}` 系エンドポイントの `{userRef}` には UUID またはメールアドレス（URL エンコード）を渡せます。

- UUID: `f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81`
- email: `user%40example.com`（`@` を `%40` にエンコード）

メールアドレスを渡した場合、サーバーが Kratos で identity を検索して UUID に解決します。

### `PATCH /v1/admin/users/{userRef}`

state と roles を一度に更新できる汎用 PATCH です。`state` と `roles` はどちらか一方のみでも両方でも指定できます。

リクエスト JSON:

```json
{
  "state": "active",
  "roles": ["admin", "support"]
}
```

| フィールド | 説明 |
|---|---|
| `state` | `active` または `inactive`。省略可 |
| `roles` | roles の全置き換え。省略可。`[]` で全ロール削除 |

両方省略すると `400 Bad Request` になります。

レスポンスは更新後の identity オブジェクトです。`roles` のみ更新した場合は `{"identity_id": "...", "roles": [...]}` を返します。

### `PUT /v1/admin/users/{userRef}/roles`

指定 identity の roles を丸ごと置き換えます。

リクエスト JSON:

```json
{
  "roles": ["admin", "support"]
}
```

レスポンス例:

```json
{
  "identity_id": "8a7b9e7b-0f84-4f54-a7e7-1ef8d8aa4f73",
  "roles": ["admin", "support"]
}
```

### `PUT /v1/admin/identities/{userRef}/roles` (非推奨)

`PUT /v1/admin/users/{userRef}/roles` の旧パスです。後方互換のため動作しますが、
レスポンスに `Deprecation: true` および `Sunset: 2027-05-01` ヘッダーが付与されます。
新規実装では `/users/` パスを使ってください。

### `POST /v1/admin/users/{userRef}/disable`

指定ユーザーを無効化します。レスポンスは更新後の identity オブジェクトです。

### `POST /v1/admin/users/{userRef}/enable`

指定ユーザーを再有効化します。レスポンスは更新後の identity オブジェクトです。

### `POST /v1/admin/users/{userRef}/revoke-sessions`

指定ユーザーの全セッションを失効します。レスポンスは `204 No Content` です。

### `DELETE /v1/admin/users/{userRef}`

指定ユーザーを削除します。レスポンスは `204 No Content` です。

### `GET /v1/admin/audit-logs`

監査ログを取得します。

このエンドポイントの各 item は明示的な JSON tag を持たないため、レスポンスキーは `EventType` のような Go のフィールド名ベースになります。

クエリパラメータ:

- `actor_type`
- `actor_id`
- `target_type`
- `target_id`
- `event_type`
- `limit`: 0 以上
- `offset`: 0 以上

レスポンス例:

```json
{
  "items": [
    {
      "ID": "ab1fdcff-65f8-4bf2-8ea0-f543b4b5d1fb",
      "EventType": "oidc_client.created",
      "ActorType": "admin_client",
      "ActorID": "bootstrap-admin",
      "TargetType": "client",
      "TargetID": "f4e3f55f-bb7a-4787-a4fb-b346b6dc1a82",
      "Result": "success",
      "IPAddress": "127.0.0.1",
      "UserAgent": "curl/8.7.1",
      "RequestID": "01HT...",
      "Metadata": {
        "app_slug": "demo-web",
        "client_name": "Demo Browser Client"
      },
      "OccurredAt": "2026-04-25T10:05:00Z"
    }
  ]
}
```

## よく使う呼び出し例

### app + OIDC client を一度に登録（省略形）

`redirect_uris` を渡すだけで app と client が同時に作られます。`client_type` / `name` / `slug` はすべて自動推論されます。

```bash
curl -X POST http://localhost:8080/v1/admin/apps \
  -H 'Authorization: Bearer dev-bootstrap-token-0123456789abcdef0123456789abcdef' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "My SPA",
    "type": "spa",
    "party_type": "first_party",
    "redirect_uris": ["https://example.com/callback"]
  }'
```

### app 作成のみ

```bash
curl -X POST http://localhost:8080/v1/admin/apps \
  -H 'Authorization: Bearer dev-bootstrap-token-0123456789abcdef0123456789abcdef' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Demo Web App",
    "type": "web",
    "party_type": "first_party",
    "description": "Internal demo app"
  }'
```

### OIDC client 作成（最小）

`name`・`client_type`・`token_endpoint_auth_method` はすべて省略可能です。

```bash
curl -X POST http://localhost:8080/v1/admin/apps/<APP_ID>/clients \
  -H 'Authorization: Bearer dev-bootstrap-token-0123456789abcdef0123456789abcdef' \
  -H 'Content-Type: application/json' \
  -d '{
    "redirect_uris": ["http://localhost:3002/callback"],
    "post_logout_redirect_uris": ["http://localhost:3002/"],
    "scopes": ["openid", "profile", "email"]
  }'
```

### ユーザーの roles を更新

```bash
curl -X PUT http://localhost:8080/v1/admin/users/<IDENTITY_ID>/roles \
  -H 'Authorization: Bearer dev-bootstrap-token-0123456789abcdef0123456789abcdef' \
  -H 'Content-Type: application/json' \
  -d '{"roles": ["admin"]}'
```

### state と roles を同時に更新

```bash
curl -X PATCH http://localhost:8080/v1/admin/users/<IDENTITY_ID> \
  -H 'Authorization: Bearer dev-bootstrap-token-0123456789abcdef0123456789abcdef' \
  -H 'Content-Type: application/json' \
  -d '{"state": "active", "roles": ["admin"]}'
```

### 監査ログ取得

```bash
curl 'http://localhost:8080/v1/admin/audit-logs?event_type=oidc_client.created&limit=20' \
  -H 'Authorization: Bearer dev-bootstrap-token-0123456789abcdef0123456789abcdef'
```
