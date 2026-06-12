import styles from '../docs/DocsLayout.module.css'

export function DocsSecurityPage() {
  return (
    <div className={styles.prose}>
      <h1>セキュリティチェックリスト</h1>
      <p>
        OAuth2/OIDC 連携では、redirect URI、state、PKCE、token 保管、secret 管理の扱いが重要です。
        本番デプロイ前に以下の項目を確認してください。
      </p>
      <h2>OAuth2/OIDC</h2>
      <ul>
        <li>Authorization Code + PKCE を使用する</li>
        <li>callback では必ず <code>state</code> を検証する</li>
        <li>redirect URI は完全一致で登録し、任意の外部 URL へ転送できる設計を避ける</li>
        <li>必要最小限のスコープのみ要求する</li>
        <li><code>nonce</code> を使用して ID Token のリプレイ攻撃を防ぐ</li>
      </ul>
      <h2>Token 管理</h2>
      <ul>
        <li>Access Token はメモリ内に保持し、localStorage には保存しない</li>
        <li>Client Secret は環境変数で管理し、フロントエンドコードに含めない</li>
        <li>Management Token はサーバー側でのみ使用する</li>
        <li>Token の有効期限切れを適切にハンドリングする</li>
      </ul>
      <h2>通信</h2>
      <ul>
        <li>本番環境では HTTPS のみで動作させる</li>
        <li>redirect URI に <code>http://</code> を使用しない（<code>localhost</code> は開発環境のみ可）</li>
      </ul>
      <h2>実装</h2>
      <ul>
        <li>サードパーティライブラリの依存関係を定期的に更新する</li>
        <li>エラーメッセージで内部情報をユーザーに露出しない</li>
        <li>認証・認可エラーを適切にログに記録する</li>
      </ul>
    </div>
  )
}
