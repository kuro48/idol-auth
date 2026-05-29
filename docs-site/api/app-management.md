# App 管理 API

Management Token を使って、自アプリに紐づくユーザーを管理するエンドポイントです。

## 認証

全エンドポイントに `Authorization: Bearer <management_token>` ヘッダーが必要です。

Management Token はアプリ登録時に発行されます。紛失した場合は管理者に再発行を依頼してください。

::: warning
Management Token はアプリの全ユーザーデータにアクセスできる強力なトークンです。サーバーサイドで安全に管理し、フロントエンドに露出させないでください。
:::

---

## エンドポイント一覧

### `GET /v1/apps/self/users`

自アプリに紐づくユーザー一覧を取得します。

**レスポンス**

```json
{
  "app": {
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "name": "My App",
    "slug": "my-app"
  },
  "items": [
    {
      "identity_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "email": "user@example.com",
      "joined_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### `POST /v1/apps/self/users`

ユーザーを事前登録（招待）します。アカウントが存在しない場合は作成します。

**リクエスト（JSON）**

```json
{
  "email": "newuser@example.com",
  "password": "TemporaryPass1!",
  "display_name": "新しいユーザー"
}
```

**レスポンス**

```json
{
  "identity_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "email": "newuser@example.com"
}
```

---

### `DELETE /v1/apps/self/users/{identityID}`

ユーザーとアプリの連携を解除します。

**注意**: Kratos identity（共有アカウント本体）は削除されません。このアプリとの紐づけのみ解除されます。

**レスポンス**: `204 No Content`

---

### `GET /v1/apps/self/users/{identityID}/profile`

ユーザーの公開プロフィールを取得します。

**レスポンス**

```json
{
  "identity_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "email": "user@example.com",
  "display_name": "Alice",
  "avatar_url": "https://<AUTH_HOST>/uploads/avatars/xxx.png",
  "locale": "ja-JP",
  "timezone": "Asia/Tokyo"
}
```

---

## curl での確認例

```bash
AUTH_HOST="https://auth.example.com"
MGT_TOKEN="mgt_..."

# ユーザー一覧
curl "$AUTH_HOST/v1/apps/self/users" \
  -H "Authorization: Bearer $MGT_TOKEN"

# ユーザー事前登録
curl -X POST "$AUTH_HOST/v1/apps/self/users" \
  -H "Authorization: Bearer $MGT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"new@example.com","password":"Temp1234!"}'

# 連携解除
curl -X DELETE "$AUTH_HOST/v1/apps/self/users/<IDENTITY_ID>" \
  -H "Authorization: Bearer $MGT_TOKEN"

# プロフィール取得
curl "$AUTH_HOST/v1/apps/self/users/<IDENTITY_ID>/profile" \
  -H "Authorization: Bearer $MGT_TOKEN"
```
