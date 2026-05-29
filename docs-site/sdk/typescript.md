# TypeScript / JavaScript SDK

`@idol-auth/client` は idol-auth の Public API をラップした公式 TypeScript クライアントです。

## インストール

```bash
npm install @idol-auth/client
```

## 初期化

```typescript
import { IdolAuthClient } from '@idol-auth/client'

const client = new IdolAuthClient({
  baseUrl: 'https://<AUTH_HOST>',
  // fetch: customFetch,  // カスタム fetch 実装（省略可）
})
```

### オプション

| オプション | 型 | 必須 | 説明 |
|---|---|---|---|
| `baseUrl` | `string` | ✅ | idol-auth サーバーのベース URL |
| `fetch` | `typeof fetch` | | カスタム fetch 実装（デフォルトは `globalThis.fetch`）|

---

## メソッド一覧

### `browserLoginUrl(params)`

OAuth2 認可画面へのリダイレクト URL を返します。

```typescript
const url: string = client.browserLoginUrl({
  clientId: 'abc123',
  redirectUri: 'https://...',
  responseType: 'code',
  scope: 'openid email',
  state: 'random-string',
  nonce: 'random-nonce',
  codeChallenge: 'xxx',
  codeChallengeMethod: 'S256',
})
```

---

### `browserRegistrationUrl(returnTo?)`

Kratos 登録画面へのリダイレクト URL を返します。

```typescript
const url: string = client.browserRegistrationUrl(
  'https://myapp.example.com/welcome'
)
```

---

### `browserLogoutUrl(params?)`

OIDC ログアウトフロー開始へのリダイレクト URL を返します。

```typescript
const url: string = client.browserLogoutUrl({
  idTokenHint: idToken,
  postLogoutRedirectUri: 'https://myapp.example.com/',
  state: 'random-state',
})
```

---

### `token(req)`

アクセストークンを取得・リフレッシュします。

```typescript
// Authorization Code フロー
const tokens: TokenResponse = await client.token({
  grant_type: 'authorization_code',
  code: 'auth-code',
  redirect_uri: 'https://myapp.example.com/callback',
  client_id: 'abc123',
  code_verifier: 'pkce-verifier',
})

// Refresh Token フロー
const tokens: TokenResponse = await client.token({
  grant_type: 'refresh_token',
  refresh_token: 'ory_rt_...',
  client_id: 'abc123',
})
```

---

### `revoke(req)`

トークンを失効させます。

```typescript
await client.revoke({ token: 'ory_rt_...', client_id: 'abc123' })
```

---

### `introspect(req)`

トークンが有効かどうかを検証します。

```typescript
const result: IntrospectResponse = await client.introspect({
  token: 'eyJ...',
  client_id: 'abc123',
})

if (result.active) {
  console.log('有効なトークン', result.sub)
}
```

---

### `session(sessionToken)`

セッショントークンからユーザー情報を取得します。

```typescript
const session: SessionResponse = await client.session('ory_st_...')
console.log(session.email, session.roles)
```

---

### `register(req)`

新規アカウントを作成してセッショントークンを返します。

```typescript
const result: AuthResult = await client.register({
  email: 'user@example.com',
  password: 'SecurePass1!',
  display_name: 'Alice',
})
```

---

### `login(req)`

ログインしてセッショントークンを返します。

```typescript
const result: AuthResult = await client.login({
  identifier: 'user@example.com',
  password: 'SecurePass1!',
})
// result.session_token, result.identity_id, result.email
```

---

## エラーハンドリング

全メソッドは HTTP エラー時に `IdolAuthError` をスローします。

```typescript
import { IdolAuthError } from '@idol-auth/client'

try {
  await client.login({ identifier: 'x', password: 'bad' })
} catch (err) {
  if (err instanceof IdolAuthError) {
    console.error(err.status)   // HTTP ステータスコード
    console.error(err.message)  // 'idol-auth: HTTP 401: ...'
  }
}
```

---

## 型定義

全型は named export で公開されています。

```typescript
import type {
  IdolAuthClientOptions,
  BrowserLoginParams,
  BrowserLogoutParams,
  TokenRequest,
  TokenResponse,
  RevokeRequest,
  IntrospectRequest,
  IntrospectResponse,
  SessionResponse,
  RegisterRequest,
  LoginRequest,
  AuthResult,
} from '@idol-auth/client'
```
