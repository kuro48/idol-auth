# idol-auth 連携ガイド

このドキュメントは idol-auth に OAuth2/OIDC でアプリを接続する開発者向けです。

## 概要

idol-auth は複数のアプリが 1 つの ID プールを共有できる認証基盤です。アプリ側の実装は 2 ステップです。

1. **Admin API** でアプリ登録 → OIDC クライアント発行（初回のみ）
2. **Public API / TypeScript SDK** で認証フローを実装

---

## ステップ 1: アプリ登録

Admin API を叩くには `ADMIN_BOOTSTRAP_TOKEN` が必要です。サービス管理者から受け取ってください。

### アプリを作成する

```bash
curl -X POST https://<AUTH_HOST>/v1/admin/apps \
  -H "Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My App",
    "slug": "my-app",
    "type": "web"
  }'
```

`type` は `web` / `spa` / `native` / `m2m` から選んでください。

レスポンス例:

```json
{
  "app": {
    "id": "01234567-89ab-cdef-0123-456789abcdef",
    "name": "My App",
    "slug": "my-app",
    "type": "web",
    "party_type": "third_party",
    "status": "active"
  },
  "management_token": "mgt_..."
}
```

`management_token` は安全に保管してください。後から `/v1/apps/self/*` で自アプリのユーザー管理に使います。

### OIDC クライアントを発行する

```bash
curl -X POST https://<AUTH_HOST>/v1/admin/apps/<APP_ID>/clients \
  -H "Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "client_type": "public",
    "redirect_uris": ["https://myapp.example.com/callback"],
    "scopes": ["openid", "email", "profile", "offline_access"]
  }'
```

`client_type` は `public`（PKCE 必須・SPA/ネイティブ向け）または `confidential`（サーバーサイド向け）です。

レスポンス例:

```json
{
  "client": {
    "hydra_client_id": "abc123",
    "client_type": "public",
    "pkce_required": true
  }
}
```

---

## ステップ 2: 認証フローの実装

### TypeScript SDK のインストール

```bash
npm install @idol-auth/client
```

### パターン A: ブラウザ OAuth2 フロー（推奨）

SPA・Web アプリはこちらを使います。

```typescript
import { IdolAuthClient } from "@idol-auth/client";

const client = new IdolAuthClient({
  baseUrl: "https://<AUTH_HOST>",
});

// 1. PKCE パラメータを生成
const codeVerifier = generateCodeVerifier();     // 43〜128文字のランダム文字列
const codeChallenge = await generateCodeChallenge(codeVerifier); // SHA-256 → base64url

// 2. 認可画面にリダイレクト
const loginUrl = client.browserLoginUrl({
  clientId: "abc123",
  redirectUri: "https://myapp.example.com/callback",
  scope: "openid email profile offline_access",
  state: crypto.randomUUID(),
  codeChallenge,
  codeChallengeMethod: "S256",
});
window.location.href = loginUrl;
```

```typescript
// 3. コールバックでトークンを取得
const params = new URLSearchParams(window.location.search);
const tokens = await client.token({
  grant_type: "authorization_code",
  code: params.get("code")!,
  redirect_uri: "https://myapp.example.com/callback",
  client_id: "abc123",
  code_verifier: codeVerifier, // PKCE
});

// tokens.access_token, tokens.id_token, tokens.refresh_token
```

### パターン B: Headless 認証（モバイル / バックエンド向け）

ブラウザリダイレクトを使わずに直接 session_token を取得できます。

```typescript
// 新規登録
const result = await client.register({
  email: "user@example.com",
  password: "secure-password",
  display_name: "Alice",
});
// result.session_token, result.identity_id, result.email

// ログイン
const result = await client.login({
  identifier: "user@example.com",
  password: "secure-password",
});
```

### セッション確認

```typescript
const session = await client.session(sessionToken);
// session.active, session.email, session.roles
```

### トークンのリフレッシュ

```typescript
const tokens = await client.token({
  grant_type: "refresh_token",
  refresh_token: storedRefreshToken,
  client_id: "abc123",
});
```

### ログアウト

```typescript
const logoutUrl = client.browserLogoutUrl({
  idTokenHint: idToken,
  postLogoutRedirectUri: "https://myapp.example.com/",
});
window.location.href = logoutUrl;
```

---

## ステップ 3: ユーザー管理（任意）

Management Token を使うと、自アプリに紐づくユーザーの一覧取得・連携解除ができます。

```bash
# ユーザー一覧
curl https://<AUTH_HOST>/v1/apps/self/users \
  -H "Authorization: Bearer <MANAGEMENT_TOKEN>"

# 特定ユーザーの連携解除
curl -X DELETE https://<AUTH_HOST>/v1/apps/self/users/<IDENTITY_ID> \
  -H "Authorization: Bearer <MANAGEMENT_TOKEN>"
```

Management Token は Kratos identity 本体を削除する権限を持ちません。自アプリの membership のみ操作できます。

---

## エラーハンドリング

SDK は HTTP エラー時に `IdolAuthError` をスローします。

```typescript
import { IdolAuthClient, IdolAuthError } from "@idol-auth/client";

try {
  const result = await client.login({ identifier, password });
} catch (err) {
  if (err instanceof IdolAuthError) {
    console.error(`HTTP ${err.status}: ${err.message}`);
    // 401 → 認証失敗
    // 400 → リクエスト不正
    // 429 → レート制限超過
  }
}
```

---

## API リファレンス

詳細なリクエスト/レスポンス仕様は Swagger UI を参照してください。

- `https://<AUTH_HOST>/docs`

主要エンドポイント一覧:

| エンドポイント | 用途 |
|---|---|
| `GET /v1/public/browser/login` | OAuth2 認可画面へリダイレクト |
| `GET /v1/public/browser/registration` | Kratos 登録画面へリダイレクト |
| `GET /v1/public/browser/logout` | OIDC ログアウトフロー開始 |
| `POST /v1/public/api/token` | トークン取得・リフレッシュ |
| `POST /v1/public/api/token/revoke` | トークン失効 |
| `POST /v1/public/api/token/introspect` | トークン検証 |
| `GET /v1/public/api/session` | セッション確認 |
| `POST /v1/public/api/register` | Headless 登録 |
| `POST /v1/public/api/login` | Headless ログイン |
| `GET /v1/apps/self/users` | 自アプリのユーザー一覧 |
| `DELETE /v1/apps/self/users/{id}` | 自アプリのユーザー連携解除 |
