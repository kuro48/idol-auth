# アプリ申請の流れ

idol-auth に新しいアプリを登録する 2 つの方法を説明します。

## 方法 A: 管理者に直接依頼する（推奨）

サービス管理者に連絡し、以下の情報を伝えてください。

- アプリ名
- アプリの種類（`web` / `spa` / `native` / `m2m`）
- コールバック URL（OAuth2 認可後のリダイレクト先）
- 必要なスコープ（`openid`, `email`, `profile`, `offline_access`）

管理者が `ADMIN_BOOTSTRAP_TOKEN` を使って登録作業を行い、以下を発行してくれます。

- `client_id`（Hydra クライアント ID）
- `management_token`

## 方法 B: 開発者ポータルから申請する

idol-auth のサービスに開発者アカウントがある場合、ポータルから申請できます。

1. `/developer/app-requests/new` にアクセス
2. アプリ名・種類・説明・希望スコープを入力して提出
3. 管理者が承認 → メール通知

申請の状態は `/developer/app-requests/` で確認できます。

## アプリの種類

| type | 説明 | 推奨クライアント |
|---|---|---|
| `web` | サーバーサイドレンダリングの Web アプリ | `confidential`（client_secret あり）|
| `spa` | React / Vue 等のシングルページアプリ | `public`（PKCE 必須）|
| `native` | iOS / Android ネイティブアプリ | `public`（PKCE 必須）|
| `m2m` | マシン間通信・バックエンドサービス | `confidential`（Client Credentials）|

## OIDC クライアントの追加発行

既存アプリに対して追加のクライアント（例: 本番用・ステージング用）を発行できます。

```bash
curl -X POST https://<AUTH_HOST>/v1/admin/apps/<APP_ID>/clients \
  -H "Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "client_type": "public",
    "redirect_uris": ["https://staging.myapp.example.com/callback"],
    "scopes": ["openid", "email", "profile"]
  }'
```

## Management Token の再発行

`management_token` を紛失した場合は再発行できます。

```bash
curl -X POST https://<AUTH_HOST>/v1/admin/apps/<APP_ID>/management-token \
  -H "Authorization: Bearer <ADMIN_BOOTSTRAP_TOKEN>"
```

::: warning
再発行すると古いトークンは即時無効になります。
:::
