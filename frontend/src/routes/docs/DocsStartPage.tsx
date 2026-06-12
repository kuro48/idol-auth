import styles from '../docs/DocsLayout.module.css'

export function DocsStartPage() {
  return (
    <div className={styles.prose}>
      <h1>最初の連携を作る</h1>
      <p>
        このページでは、アプリ登録、クレデンシャル保管、ログイン導線、callback 処理、
        本番前チェックまでの流れを説明します。
      </p>
      <h2>1. アプリを登録する</h2>
      <p>
        <a href="/developer/app-requests/new">開発者画面</a>からアプリ名、種別、redirect URI、
        logout 後の戻り先を登録します。承認されると Client ID が発行されます。
      </p>
      <h2>2. クレデンシャルを保管する</h2>
      <p>
        Web サーバー型のアプリでは Client Secret も保管します。
        SPA やネイティブアプリに Secret を埋め込んではいけません。
        Secret を持てないクライアントは PKCE を必須として運用します。
      </p>
      <h2>3. Authorization Code + PKCE フローを実装する</h2>
      <pre><code>{`import { IdolAuthClient } from "@idol-auth/client";

const auth = new IdolAuthClient({
  clientId: "YOUR_CLIENT_ID",
  redirectUri: "https://your-app.example.com/callback",
  issuer: "https://auth.example.com",
});

// ログイン開始
const { url } = await auth.startLogin({ scopes: ["openid", "profile"] });
window.location.href = url;

// コールバック処理
const session = await auth.handleCallback(window.location.href);`}</code></pre>
      <h2>4. state・nonce を必ず検証する</h2>
      <p>
        CSRF 攻撃を防ぐため、callback では <code>state</code> パラメータを検証してください。
        <code>nonce</code> は ID Token の <code>nonce</code> クレームと照合します。
      </p>
      <h2>5. 本番前チェック</h2>
      <ul>
        <li>redirect URI が完全一致で登録されているか</li>
        <li>Client Secret が環境変数で管理されているか</li>
        <li>HTTPS のみで動作するか</li>
        <li><a href="/docs/security">セキュリティチェックリスト</a>を確認する</li>
      </ul>
    </div>
  )
}
