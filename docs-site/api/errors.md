# エラーコード

## エラーレスポンス形式

```json
{
  "error": "エラーメッセージ"
}
```

## HTTP ステータスコード

| コード | 意味 | よくある原因 |
|---|---|---|
| `400 Bad Request` | リクエスト不正 | 必須パラメータが欠けている、パスワードが弱すぎる |
| `401 Unauthorized` | 認証失敗 | トークンが無効・期限切れ、認証情報が誤り |
| `403 Forbidden` | 権限不足 | アクセス権限がない操作を試みた |
| `404 Not Found` | リソース不存在 | 指定したユーザー ID やアプリ ID が存在しない |
| `409 Conflict` | 重複 | メールアドレスが既に登録済み |
| `429 Too Many Requests` | レート制限超過 | 短時間に多数のリクエストを送信した |
| `500 Internal Server Error` | サーバーエラー | サービス側の問題 |

## レート制限

エンドポイントごとにレート制限があります。

| エンドポイント | 制限 |
|---|---|
| `POST /v1/public/api/register` | 5 回 / 分 / IP |
| `POST /v1/public/api/login` | 5 回 / 分 / IP |
| その他の Public API | 緩やかな制限（IP 単位）|

制限を超えた場合は `429 Too Many Requests` が返ります。`Retry-After` ヘッダーで再試行可能な時刻を確認してください。

## SDK でのエラーハンドリング

TypeScript SDK は HTTP エラー時に `IdolAuthError` をスローします。

```typescript
import { IdolAuthClient, IdolAuthError } from '@idol-auth/client'

const client = new IdolAuthClient({ baseUrl: 'https://<AUTH_HOST>' })

try {
  const result = await client.login({
    identifier: 'user@example.com',
    password: 'wrong-password',
  })
} catch (err) {
  if (err instanceof IdolAuthError) {
    switch (err.status) {
      case 400:
        console.error('リクエストが不正です:', err.message)
        break
      case 401:
        console.error('認証に失敗しました')
        break
      case 429:
        console.error('しばらく待ってから再試行してください')
        break
      default:
        console.error(`エラー ${err.status}:`, err.message)
    }
  }
}
```

`IdolAuthError` のプロパティ:

| プロパティ | 型 | 説明 |
|---|---|---|
| `status` | `number` | HTTP ステータスコード |
| `message` | `string` | `idol-auth: HTTP {status}: {body}` 形式のメッセージ |
