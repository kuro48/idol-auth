# OIDCフロー（ブラウザ）

ブラウザアプリ・SPA・ネイティブアプリで使う OAuth2 Authorization Code + PKCE フローの詳細解説です。

## 全体像

```
ユーザー → クライアントアプリ → Hydra → idol-auth → Kratos
```

idol-auth は Hydra（OAuth2サーバー）と Kratos（IDサーバー）の間を仲介するブリッジとして動きます。

## 1. ログインフロー

```mermaid
sequenceDiagram
    actor User as ユーザー
    participant App as クライアントアプリ
    participant Hydra as Ory Hydra
    participant Server as idol-auth
    participant Kratos as Ory Kratos

    User->>App: ログインボタン押下
    App->>Hydra: GET /oauth2/auth?response_type=code&...
    Hydra-->>Server: リダイレクト /v1/auth/login?login_challenge=xxx

    alt 既存セッションあり
        Server->>Hydra: AcceptLoginRequest
        Hydra-->>Server: redirect_to
    else 新規ログイン
        Server-->>User: Kratos ログイン画面へリダイレクト
        User->>Kratos: メール・パスワード入力
        Kratos-->>User: セッションCookie発行
        User->>Server: リダイレクト /v1/auth/login?login_challenge=xxx
        Server->>Hydra: AcceptLoginRequest
        Hydra-->>Server: redirect_to
    end
    Server-->>User: 同意フローへ
```

## 2. 同意フロー（Consent）

first_party アプリはスキップされます。third_party アプリは同意画面が表示されます。

```mermaid
sequenceDiagram
    actor User as ユーザー
    participant Hydra as Ory Hydra
    participant Server as idol-auth

    Hydra-->>Server: リダイレクト /v1/auth/consent?consent_challenge=xxx

    alt first_party または同意スキップ
        Server->>Hydra: AcceptConsentRequest（自動）
        Hydra-->>Server: redirect_to
        Server-->>User: コールバックへリダイレクト
    else third_party（同意画面表示）
        Server-->>User: スコープ一覧の同意画面
        User->>Server: POST /v1/auth/consent（許可 or 拒否）
        Server->>Hydra: AcceptConsentRequest or RejectConsentRequest
        Hydra-->>Server: redirect_to
        Server-->>User: コールバックへリダイレクト
    end
```

## 3. ログアウトフロー

```mermaid
sequenceDiagram
    actor User as ユーザー
    participant App as クライアントアプリ
    participant Hydra as Ory Hydra
    participant Server as idol-auth

    User->>App: ログアウトボタン押下
    App->>Server: POST /v1/auth/logout
    Server-->>App: { logout_url }
    App->>Hydra: GET /oauth2/sessions/logout（id_token_hint付き）
    Hydra-->>Server: リダイレクト /v1/auth/logout?logout_challenge=xxx
    Server->>Hydra: AcceptLogoutRequest
    Hydra-->>Server: redirect_to
    Server-->>User: ログアウト完了
```

SDK でのログアウト実装:

```typescript
const logoutUrl = client.browserLogoutUrl({
  idTokenHint: idToken,
  postLogoutRedirectUri: 'https://myapp.example.com/',
})
window.location.href = logoutUrl
```

## PKCE の実装

公開クライアント（SPA・ネイティブ）では PKCE が必須です。

```typescript
function generateCodeVerifier(): string {
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

async function generateCodeChallenge(verifier: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(verifier)
  const hash = await crypto.subtle.digest('SHA-256', data)
  return btoa(String.fromCharCode(...new Uint8Array(hash)))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}
```

## スコープ一覧

| スコープ | 内容 |
|---|---|
| `openid` | OIDC 認証（必須）|
| `email` | メールアドレス |
| `profile` | 表示名・アバター等 |
| `offline_access` | リフレッシュトークン発行 |

## トークンのリフレッシュ

```typescript
const tokens = await client.token({
  grant_type: 'refresh_token',
  refresh_token: storedRefreshToken,
  client_id: 'abc123',
})
```
