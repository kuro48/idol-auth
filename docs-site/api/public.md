# Public API

認証なし（または Bearer トークン）でアクセスできる公開エンドポイントです。CORS が許可されており、ブラウザから直接呼び出せます。

::: tip インタラクティブに試す
詳細なリクエスト/レスポンス仕様とインタラクティブな try-it-out は **[API リファレンス](/docs/index.html)** を参照してください。
:::

## ブラウザリダイレクト

### `GET /v1/public/browser/login`

OAuth2 認可画面（Hydra）へのリダイレクト URL です。

**クエリパラメータ**

| パラメータ | 必須 | 説明 |
|---|---|---|
| `client_id` | ✅ | OIDC クライアント ID |
| `redirect_uri` | ✅ | 認可後のコールバック URL |
| `response_type` | ✅ | `code` 固定 |
| `scope` | | スペース区切りのスコープ |
| `state` | | CSRF 対策用のランダム文字列 |
| `nonce` | | リプレイ攻撃対策用のランダム文字列 |
| `code_challenge` | | PKCE チャレンジ（公開クライアントは必須）|
| `code_challenge_method` | | `S256` 固定 |

SDK での使い方:
```typescript
const url = client.browserLoginUrl({
  clientId: 'abc123',
  redirectUri: 'https://myapp.example.com/callback',
  scope: 'openid email profile',
  state: crypto.randomUUID(),
  codeChallenge,
  codeChallengeMethod: 'S256',
})
window.location.href = url
```

---

### `GET /v1/public/browser/registration`

Kratos の新規登録画面へのリダイレクト URL です。

**クエリパラメータ**

| パラメータ | 必須 | 説明 |
|---|---|---|
| `return_to` | | 登録完了後のリダイレクト先 URL |

---

### `GET /v1/public/browser/logout`

OIDC ログアウトフロー開始へのリダイレクト URL です。

**クエリパラメータ**

| パラメータ | 必須 | 説明 |
|---|---|---|
| `id_token_hint` | | ログアウト対象の ID トークン |
| `post_logout_redirect_uri` | | ログアウト完了後のリダイレクト先 |
| `state` | | CSRF 対策用のランダム文字列 |

---

## トークン操作

### `POST /v1/public/api/token`

アクセストークンの取得・リフレッシュ。OAuth2 標準の Token Endpoint です。

**リクエスト（`application/x-www-form-urlencoded`）**

Authorization Code フロー:

| フィールド | 必須 | 説明 |
|---|---|---|
| `grant_type` | ✅ | `authorization_code` |
| `code` | ✅ | 認可コード |
| `redirect_uri` | ✅ | 認可リクエスト時と同一の URL |
| `client_id` | ✅ | OIDC クライアント ID |
| `code_verifier` | △ | PKCE 必須クライアントでは必須 |
| `client_secret` | △ | Confidential クライアントでは必須 |

Refresh Token フロー:

| フィールド | 必須 | 説明 |
|---|---|---|
| `grant_type` | ✅ | `refresh_token` |
| `refresh_token` | ✅ | リフレッシュトークン |
| `client_id` | ✅ | OIDC クライアント ID |

**レスポンス**

```json
{
  "access_token": "eyJ...",
  "token_type": "bearer",
  "expires_in": 3600,
  "refresh_token": "ory_rt_...",
  "id_token": "eyJ..."
}
```

---

### `POST /v1/public/api/token/revoke`

トークンを失効させます。

**リクエスト**

| フィールド | 必須 | 説明 |
|---|---|---|
| `token` | ✅ | 失効させるトークン |
| `client_id` | ✅ | OIDC クライアント ID |
| `client_secret` | △ | Confidential クライアントでは必須 |

---

### `POST /v1/public/api/token/introspect`

トークンが有効かどうかを検証します。

**リクエスト**

| フィールド | 必須 | 説明 |
|---|---|---|
| `token` | ✅ | 検証するトークン |
| `client_id` | ✅ | OIDC クライアント ID |
| `client_secret` | △ | Confidential クライアントでは必須 |

**レスポンス**

```json
{
  "active": true,
  "sub": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "client_id": "abc123",
  "scope": "openid email profile",
  "exp": 1700000000
}
```

---

## セッション・Headless

### `GET /v1/public/api/session`

セッショントークンからユーザー情報を取得します。

**認証**: `Authorization: Bearer <session_token>`

**レスポンス**

```json
{
  "active": true,
  "subject": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "email": "user@example.com",
  "display_name": "Alice",
  "roles": ["user"]
}
```

---

### `POST /v1/public/api/register`

新規アカウントを作成してセッショントークンを返します。

**リクエスト（JSON）**

```json
{
  "email": "user@example.com",
  "password": "SecurePass1!",
  "display_name": "Alice"
}
```

**レスポンス**

```json
{
  "session_token": "ory_st_...",
  "identity_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "email": "user@example.com"
}
```

---

### `POST /v1/public/api/login`

メールアドレスとパスワードでログインしてセッショントークンを返します。

**リクエスト（JSON）**

```json
{
  "identifier": "user@example.com",
  "password": "SecurePass1!"
}
```

**レスポンス** — `POST /v1/public/api/register` と同じ形式
