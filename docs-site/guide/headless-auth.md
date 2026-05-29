# Headless 認証（API）

ブラウザリダイレクトを使わず、API 直接呼び出しで認証する方式です。モバイルアプリやサーバーサイドのバックエンドに向いています。

::: warning
Headless 認証ではレート制限が設けられています（5回/分/IP）。過度なリクエストは `429 Too Many Requests` が返ります。
:::

## 新規登録

```typescript
import { IdolAuthClient, IdolAuthError } from '@idol-auth/client'

const client = new IdolAuthClient({ baseUrl: 'https://<AUTH_HOST>' })

try {
  const result = await client.register({
    email: 'user@example.com',
    password: 'secure-password',
    display_name: 'Alice',  // 省略可
  })

  // result.session_token  — 以降のリクエストに使用
  // result.identity_id    — ユーザーの一意 ID
  // result.email          — 登録したメールアドレス
} catch (err) {
  if (err instanceof IdolAuthError) {
    // 400: バリデーションエラー（パスワードが弱い等）
    // 409: メールアドレス重複
    console.error(err.status, err.message)
  }
}
```

## ログイン

```typescript
const result = await client.login({
  identifier: 'user@example.com',  // メールアドレス または電話番号
  password: 'secure-password',
})

// result.session_token を保存して以降のリクエストに使用
```

## セッション確認

取得した `session_token` でユーザー情報を取得します。

```typescript
const session = await client.session(result.session_token)

// session.active       — セッションが有効かどうか
// session.email        — メールアドレス
// session.subject      — ユーザー ID
// session.roles        — ロール配列
// session.display_name — 表示名
```

::: tip
Headless モードは主に **独立したモバイルアプリ** や **バックエンド間通信** で使います。ブラウザ系のアプリは [OIDCフロー](./auth-flows) を使ってください。
:::

## curl での確認例

```bash
# 登録
curl -X POST https://<AUTH_HOST>/v1/public/api/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test1234!"}'

# ログイン
curl -X POST https://<AUTH_HOST>/v1/public/api/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"test@example.com","password":"Test1234!"}'

# セッション確認
curl https://<AUTH_HOST>/v1/public/api/session \
  -H "Authorization: Bearer <session_token>"
```
