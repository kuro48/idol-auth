import styles from '../docs/DocsLayout.module.css'

export function DocsAccountPage() {
  return (
    <div className={styles.prose}>
      <h1>アカウントセンター — ユーザー本人の管理画面</h1>
      <p>
        アカウントセンターは、ユーザー本人が共有アカウントと連携アプリを管理する場所です。
        第三者アプリは、ユーザーがこの画面で連携解除や認証設定変更を行えることを前提に設計します。
      </p>
      <h2>ユーザーができること</h2>
      <ul>
        <li><strong>プロフィール確認</strong> — 表示名、メール、電話番号、公開プロフィール情報を確認</li>
        <li><strong>連携アプリ管理</strong> — 接続済みアプリを確認し、不要な連携を解除</li>
        <li><strong>認証設定</strong> — パスワード、MFA、バックアップコードを管理</li>
        <li><strong>セッション管理</strong> — アクティブなセッションを確認・無効化</li>
        <li><strong>アカウント削除</strong> — アカウント削除のスケジュール設定・キャンセル</li>
      </ul>
      <h2>アプリ側の設計ポイント</h2>
      <ul>
        <li>連携解除後にユーザーがログインしようとした場合の動作を定義してください</li>
        <li>ユーザー向けに「アカウント設定」へのリンクを提供することを推奨します</li>
        <li>Access Token の有効期限切れやスコープ変更に対応した設計が必要です</li>
      </ul>
      <h2>設定ページへのリンク</h2>
      <p>ユーザーを認証設定ページに誘導する場合、以下の URL を使用します：</p>
      <pre><code>{`GET /v1/auth/providers
→ レスポンスの settings_url を参照`}</code></pre>
    </div>
  )
}
