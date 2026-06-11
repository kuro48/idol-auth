# idol-auth 連携ガイド

このドキュメントは idol-auth に OAuth2/OIDC でアプリを接続する開発者向けです。

## 概要

idol-auth は複数のアプリが 1 つの ID プールを共有できる認証基盤です。アプリ側の実装は 2 ステップです。

1. **セルフサービス登録**でアプリ登録 → OIDC クライアントとクレデンシャルが即時発行（初回のみ）
2. **Public API / TypeScript SDK** で認証フローを実装

管理者への連絡やトークンの手渡しは不要です。

---

## ステップ 1: アプリ登録（セルフサービス・即時発行）

idol-auth アカウントでログインして登録するだけで、審査なしで OIDC クライアントとクレデンシャルが発行されます。

### 方法 A: ブラウザから登録（推奨）

1. `https://<AUTH_HOST>` でアカウントを作成（メール認証あり）
2. `https://<AUTH_HOST>/developer/app-requests/new` を開く
3. アプリ名・種別・説明・リダイレクト URI を入力して「登録する（即時発行）」
4. 表示された **Client ID / Client Secret / Management Token** を保管

クレデンシャルは**このページにしか表示されません**。必ずその場でコピーしてください。

### 方法 B: API から登録

ログインセッション（Kratos セッション Cookie）で `POST /v1/developer/apps` を呼びます。

```bash
curl -X POST https://<AUTH_HOST>/v1/developer/apps \
  -b "ory_kratos_session=<SESSION_COOKIE>" \
  -H "Content-Type: application/json" \
  -H "Sec-Fetch-Site: same-origin" \
  -d '{
    "name": "My App",
    "type": "web",
    "description": "My awesome app",
    "redirect_uris": ["https://myapp.example.com/callback"],
    "post_logout_redirect_uris": ["https://myapp.example.com/"],
    "scopes": ["openid", "email", "profile", "offline_access"]
  }'
```

`type` は `web` / `spa` / `native` / `m2m` から選んでください。

レスポンス例（`client_secret` と `management_token` は一度だけ返されます）:

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
  "client": {
    "hydra_client_id": "abc123",
    "client_type": "confidential",
    "pkce_required": false
  },
  "client_secret": "...",
  "management_token": "mgt_..."
}
```

`management_token` は安全に保管してください。後から `/v1/apps/self/*` で自アプリのユーザー管理に使います。

### セルフサービス登録の制限

| 項目 | 制限 |
|---|---|
| スコープ | `openid` / `email` / `profile` / `offline_access` のみ |
| party_type | `third_party` 固定 |
| アプリ数 | 1 アカウントあたり最大 5 件 |

これを超える要件（カスタムスコープ、first_party 扱いなど）は管理者へ相談してください（従来の Admin API での発行も引き続き可能です）。

### 管理者向け: 事後監視

事前審査の代わりに、登録は事後監視で運用します。

- 新規登録ごとに `ADMIN_ALLOWED_EMAILS` 宛に通知メールが送信されます
- 全登録は管理 UI（`/admin/apps`・`/admin/app-requests`）と監査ログで確認できます
- 不審なアプリは管理 API でいつでも無効化（`status: disabled`）できます

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

// 1. ログインボタンなどから呼ぶ。
// PKCE（verifier/challenge）と state の生成・保存・検証は SDK が行います。
await client.loginWithRedirect({
  clientId: "abc123",
  redirectUri: "https://myapp.example.com/callback",
});
```

```typescript
// 2. コールバックページで呼ぶ。state 検証とトークン交換まで自動。
const tokens = await client.handleRedirectCallback();

// tokens.access_token, tokens.id_token, tokens.refresh_token
```

PKCE を手動で制御したい場合は低レベルヘルパーも利用できます。

```typescript
import { generatePKCE } from "@idol-auth/client";

const { codeVerifier, codeChallenge } = await generatePKCE();
const loginUrl = client.browserLoginUrl({
  clientId: "abc123",
  redirectUri: "https://myapp.example.com/callback",
  scope: "openid email profile offline_access",
  state: crypto.randomUUID(),
  codeChallenge,
  codeChallengeMethod: "S256",
});
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

アプリ側で招待や事前作成が必要な場合は、Management Token で membership 付き identity を作成できます。

```bash
curl -X POST https://<AUTH_HOST>/v1/apps/self/users \
  -H "Authorization: Bearer <MANAGEMENT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "temporary-password",
    "display_name": "Alice"
  }'
```

アプリユーザー向けに公開できるプロフィールは次で取得します。

```bash
curl https://<AUTH_HOST>/v1/apps/self/users/<IDENTITY_ID>/profile \
  -H "Authorization: Bearer <MANAGEMENT_TOKEN>"
```

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

詳細なリクエスト/レスポンス仕様は `/docs/doc.json` の OpenAPI 仕様を参照してください。

- `https://<AUTH_HOST>/docs/index.html`
- `https://<AUTH_HOST>/docs/doc.json`

主要エンドポイント一覧:

| エンドポイント | 用途 |
|---|---|
| `GET /v1/public/browser/login` | OAuth2 認可画面へリダイレクト |
| `GET /v1/public/browser/registration` | Kratos 登録画面へリダイレクト |
| `GET /v1/public/browser/logout` | OIDC ログアウトフロー開始 |
| `GET /account/` | アカウントセンター（SNS-style profile page、要認証） |
| `GET /settings?flow=<flowID>` | Kratos 設定画面プロキシ（メール・パスワード・MFA 変更、要認証） |
| `POST /v1/public/api/token` | トークン取得・リフレッシュ |
| `POST /v1/public/api/token/revoke` | トークン失効 |
| `POST /v1/public/api/token/introspect` | トークン検証 |
| `GET /v1/public/api/session` | セッション確認 |
| `POST /v1/public/api/register` | Headless 登録 |
| `POST /v1/public/api/login` | Headless ログイン |
| `GET /v1/apps/self/users` | 自アプリのユーザー一覧 |
| `POST /v1/apps/self/users` | 自アプリのユーザー事前登録 |
| `DELETE /v1/apps/self/users/{id}` | 自アプリのユーザー連携解除 |
| `GET /v1/apps/self/users/{id}/profile` | 自アプリユーザーの公開プロフィール取得 |
| `POST /v1/developer/apps` | アプリ登録（即時発行・要セッション） |
| `GET /v1/developer/app-requests` | 自分のアプリ登録一覧 |
| `POST /v1/developer/app-requests` | アプリ申請を提出（審査フロー） |

---

## ステップ 4: アカウントセンターと設定（ユーザー向け画面）

ユーザーが自分のプロフィールを管理する UI を提供しています。

### アカウントセンター (`/account/`)

ユーザーがログインして訪問できるセンター画面です。以下をサポートしています:

- **プロフィール表示** — display_name, email, phone, oshi_color など
- **連携アプリ一覧** — 現在接続している third-party app の表示・連携解除
- **共有アカウント削除予約** — アカウント完全削除を申請（管理 API で後続処理）

セッション認証が必須です。未認証ユーザーは Kratos ログイン画面へリダイレクトされます。

### 設定画面 (`/settings?flow=<flowID>`)

Kratos self-service settings flow をプロキシした画面です。ユーザーは以下を変更できます:

- **プロフィール** — メールアドレス、電話番号、表示名
- **パスワード変更**
- **TOTP 設定** — 二段階認証（TOTP）の有効化・無効化
- **バックアップコード** — MFA 認証失敗時の回復用コード

**アクセス方法:**

1. Kratos browser settings flow で flow_id を取得
2. `/settings?flow=<flow_id>` へリダイレクト
3. ユーザーが値を変更
4. POST submit → Kratos へリレー → success 時は `/account/` へ自動リダイレクト

セッション認証が必須です。
