import styles from '../docs/DocsLayout.module.css'

export function DocsManagementPage() {
  return (
    <div className={styles.prose}>
      <h1>Management Token — 連携ユーザーを管理する</h1>
      <p>
        Management Token は、あなたのアプリに紐づくユーザー連携だけを扱うためのサーバー側トークンです。
        共有アカウント本体や他アプリの連携にはアクセスできません。
      </p>
      <h2>できること</h2>
      <ul>
        <li><strong>連携ユーザーの確認</strong> — 自アプリに接続済みのユーザーを一覧取得できます</li>
        <li><strong>連携解除</strong> — 自アプリから見たユーザーの membership を無効化できます</li>
        <li><strong>公開プロフィール参照</strong> — 許可された範囲の公開プロフィールを参照できます</li>
      </ul>
      <h2>できないこと</h2>
      <ul>
        <li>共有アカウント本体の完全削除</li>
        <li>他アプリの連携へのアクセス</li>
        <li>admin 権限が必要な操作</li>
      </ul>
      <h2>Management Token の取得</h2>
      <p>
        Management Token はアプリ登録が承認されると発行されます。
        開発者画面の App Request 詳細ページで確認できます（一度しか表示されません）。
      </p>
      <h2>API の使い方</h2>
      <pre><code>{`# 連携ユーザー一覧
GET /v1/apps/self/users
Authorization: Bearer {management_token}

# プロフィール取得
GET /v1/apps/self/users/{identity_id}/profile
Authorization: Bearer {management_token}`}</code></pre>
      <h2>セキュリティ注意事項</h2>
      <ul>
        <li>Management Token はサーバー側でのみ使用してください</li>
        <li>クライアントサイドのコードや公開リポジトリに含めないでください</li>
        <li>環境変数または secrets manager で管理してください</li>
      </ul>
    </div>
  )
}
