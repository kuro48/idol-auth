import styles from '../docs/DocsLayout.module.css'

export function DocsOverviewPage() {
  return (
    <div className={styles.prose}>
      <h1>idol-auth — 外部開発者向けドキュメント</h1>
      <p>
        idol-auth は、複数サービスで共通のユーザーアカウントを利用するための認証基盤です。
        このドキュメントは、ソースコードや運用環境にアクセスしない第三者アプリ開発者向けに、
        導入の流れと実装上の判断材料をまとめています。
      </p>
      <h2>外部開発者が扱う範囲</h2>
      <ul>
        <li><strong>アプリ登録</strong> — 開発者画面でアプリを登録し、OAuth2 クライアントを取得します。</li>
        <li><strong>ログイン導線</strong> — Authorization Code + PKCE フローで認証を実装します。</li>
        <li><strong>ユーザー管理</strong> — Management Token を使い、自アプリの連携ユーザーを管理します。</li>
        <li><strong>アカウントセンター</strong> — ユーザー本人が連携解除や設定変更を行う画面の URL を把握します。</li>
      </ul>
      <h2>クイックリンク</h2>
      <ul>
        <li><a href="/docs/start">最初の連携を作る</a> — アプリ登録からコールバック処理まで</li>
        <li><a href="/docs/sdk">TypeScript SDK</a> — ブラウザ向け OAuth2 フロー実装</li>
        <li><a href="/docs/api">API リファレンス</a> — 全エンドポイントの詳細仕様</li>
        <li><a href="/docs/security">セキュリティチェックリスト</a> — 本番前に確認すべき実装ポイント</li>
      </ul>
    </div>
  )
}
