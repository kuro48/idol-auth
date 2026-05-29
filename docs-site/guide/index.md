# クイックスタート

idol-auth を使ってアプリに OAuth2/OIDC 認証を組み込む最短手順です。

## 前提

- サービス管理者から `AUTH_HOST`（idol-auth のベース URL）と `ADMIN_BOOTSTRAP_TOKEN` を受け取っていること
- Node.js 18 以上（TypeScript SDK を使う場合）

## ステップ 1: アプリを登録する

Admin API でアプリと OIDC クライアントを一括作成します。

```bash
curl -X POST https://<AUTH_HOST>/v1/admin/apps \
  -H "Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My App",
    "slug": "my-app",
    "type": "spa",
    "client": {
      "client_type": "public",
      "redirect_uris": ["https://myapp.example.com/callback"],
      "post_logout_redirect_uris": ["https://myapp.example.com/"],
      "scopes": ["openid", "email", "profile", "offline_access"]
    }
  }'
```

レスポンスから `client.hydra_client_id` と `management_token` を保存してください。

```json
{
  "app": { "id": "...", "name": "My App", "slug": "my-app" },
  "client": { "hydra_client_id": "abc123", "pkce_required": true },
  "management_token": "mgt_..."
}
```

::: tip
`management_token` はアプリのユーザー管理（一覧・連携解除）に使います。安全な場所に保管してください。
:::

## ステップ 2: SDK をインストールする

```bash
npm install @idol-auth/client
```

## ステップ 3: 認証フローを実装する

```typescript
import { IdolAuthClient } from '@idol-auth/client'

const client = new IdolAuthClient({
  baseUrl: 'https://<AUTH_HOST>',
})

// PKCE パラメータを生成
const codeVerifier = generateCodeVerifier()
const codeChallenge = await generateCodeChallenge(codeVerifier)

// 認可画面にリダイレクト
const loginUrl = client.browserLoginUrl({
  clientId: 'abc123',
  redirectUri: 'https://myapp.example.com/callback',
  scope: 'openid email profile offline_access',
  state: crypto.randomUUID(),
  codeChallenge,
  codeChallengeMethod: 'S256',
})
window.location.href = loginUrl
```

コールバックページでトークンを取得します。

```typescript
const params = new URLSearchParams(window.location.search)
const tokens = await client.token({
  grant_type: 'authorization_code',
  code: params.get('code')!,
  redirect_uri: 'https://myapp.example.com/callback',
  client_id: 'abc123',
  code_verifier: codeVerifier,
})
// tokens.access_token, tokens.id_token, tokens.refresh_token
```

## 次のステップ

- [OIDCフロー詳細](./auth-flows) — ログイン・同意・ログアウトの全フロー解説
- [Headless 認証](./headless-auth) — ブラウザなしでのAPI認証
- [App 管理 API](../api/app-management) — ユーザー管理・プロフィール取得
