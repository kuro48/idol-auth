import styles from '../docs/DocsLayout.module.css'

export function DocsSdkPage() {
  return (
    <div className={styles.prose}>
      <h1>TypeScript SDK</h1>
      <p>
        <code>@idol-auth/client</code> はブラウザ OAuth2 フロー、PKCE helper、token refresh、
        logout URL 生成、headless auth helper を提供します。
      </p>
      <h2>Install</h2>
      <pre><code>npm install @idol-auth/client</code></pre>
      <h2>Client を初期化する</h2>
      <pre><code>{`import { IdolAuthClient } from "@idol-auth/client";

const auth = new IdolAuthClient({
  clientId: "YOUR_CLIENT_ID",
  redirectUri: "https://your-app.example.com/callback",
  issuer: "https://auth.example.com",
});`}</code></pre>
      <h2>ログインフロー</h2>
      <pre><code>{`// ログイン開始 (PKCE自動生成)
const { url, state, codeVerifier } = await auth.startLogin({
  scopes: ["openid", "profile"],
});
// state と codeVerifier をセッションストレージに保存
window.location.href = url;

// コールバック処理
const storedState = sessionStorage.getItem("state");
const codeVerifier = sessionStorage.getItem("codeVerifier");
const tokens = await auth.handleCallback(window.location.href, {
  expectedState: storedState,
  codeVerifier,
});`}</code></pre>
      <h2>PKCE helpers</h2>
      <pre><code>{`import { generateCodeVerifier, generateCodeChallenge } from "@idol-auth/client";

const verifier = await generateCodeVerifier();
const challenge = await generateCodeChallenge(verifier);`}</code></pre>
      <h2>ログアウト</h2>
      <pre><code>{`const logoutUrl = auth.logoutUrl({
  postLogoutRedirectUri: "https://your-app.example.com/",
});
window.location.href = logoutUrl;`}</code></pre>
      <h2>ESM / CJS</h2>
      <p>
        パッケージは ESM と CJS のデュアルビルドを提供します。
        バンドラー環境では自動的に ESM が使用されます。
      </p>
    </div>
  )
}
