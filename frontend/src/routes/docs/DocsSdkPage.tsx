import styles from '../docs/DocsLayout.module.css'

export function DocsSdkPage() {
  return (
    <div className={styles.prose}>
      <h1>TypeScript SDK</h1>
      <p>
        <code>@idol-auth/client</code> はブラウザ向けの OAuth2 / OIDC クライアントライブラリです。
        Authorization Code + PKCE フロー、トークン管理、セッション検証、ログアウトをシンプルな API で提供します。
      </p>

      <h2>インストール</h2>
      <pre><code>{`npm install @idol-auth/client
# または
yarn add @idol-auth/client
pnpm add @idol-auth/client`}</code></pre>

      <h2>Client を初期化する</h2>
      <p>
        アプリケーション起動時に一度だけ <code>IdolAuthClient</code> を生成し、モジュールスコープで保持します。
      </p>
      <pre><code>{`import { IdolAuthClient } from "@idol-auth/client";

export const auth = new IdolAuthClient({
  // 開発者ポータルで発行された Client ID
  clientId: "YOUR_CLIENT_ID",

  // 承認済みの redirect URI と完全一致する必要があります
  redirectUri: "https://your-app.example.com/callback",

  // Issuer URL（/.well-known/openid-configuration を提供する URL）
  issuer: "https://auth.example.com",

  // (オプション) デフォルトスコープ。startLogin で上書き可能
  scopes: ["openid", "profile"],
});`}</code></pre>

      <h2>Authorization Code + PKCE フロー</h2>
      <p>
        SPA・ネイティブアプリは Client Secret を安全に保持できないため、必ず PKCE を使用してください。
        SDK は code_verifier / code_challenge の生成を自動で行います。
      </p>

      <h3>1. ログインを開始する</h3>
      <pre><code>{`// login.ts
import { auth } from "./auth";

export async function startLogin() {
  const { url, state, codeVerifier } = await auth.startLogin({
    scopes: ["openid", "profile"],
  });

  // CSRF 防止のため state と codeVerifier をセッションストレージに保存
  sessionStorage.setItem("oauth_state", state);
  sessionStorage.setItem("code_verifier", codeVerifier);

  window.location.href = url;
}`}</code></pre>

      <h3>2. コールバックを処理する</h3>
      <pre><code>{`// callback.ts
import { auth } from "./auth";

export async function handleCallback(): Promise<void> {
  const storedState = sessionStorage.getItem("oauth_state");
  const codeVerifier = sessionStorage.getItem("code_verifier");

  // state が一致しない場合は認証エラーとして扱う
  if (!storedState || !codeVerifier) {
    throw new Error("Missing OAuth state. Please retry login.");
  }

  const tokens = await auth.handleCallback(window.location.href, {
    expectedState: storedState,
    codeVerifier,
  });

  // 使用済みの state を削除
  sessionStorage.removeItem("oauth_state");
  sessionStorage.removeItem("code_verifier");

  // アクセストークンをメモリに保持（localStorage には保存しない）
  storeTokens(tokens);

  // ログイン後のページへリダイレクト
  window.location.replace("/dashboard");
}`}</code></pre>

      <h2>セッション確認</h2>
      <p>
        バックエンドの <code>/api/session</code> エンドポイントで現在のセッション状態を取得できます。
        Cookie ベースのセッションを使用するため、<code>credentials: "include"</code> が必要です。
      </p>
      <pre><code>{`import type { SessionView } from "@idol-auth/client";

async function getSession(): Promise<SessionView | null> {
  const res = await fetch("/api/session", { credentials: "include" });
  if (!res.ok) return null;
  return res.json();
}

// SessionView の型定義
interface SessionView {
  authenticated: boolean;
  subject?: string;          // Hydra subject（= Kratos identity ID）
  identity_id?: string;
  email?: string;
  display_name?: string;
  roles?: string[];
  oshi_color?: string;
  authenticator_assurance_level?: string; // "aal1" | "aal2"
}`}</code></pre>

      <h2>トークンのリフレッシュ</h2>
      <p>
        アクセストークンの有効期限が切れた場合、<code>refreshTokens</code> で新しいトークンを取得します。
        リフレッシュトークンは httpOnly Cookie で管理されるため、JS から直接アクセスできません。
      </p>
      <pre><code>{`import { auth } from "./auth";

async function getAccessToken(): Promise<string> {
  if (auth.isTokenExpired()) {
    await auth.refreshTokens();
  }
  return auth.getAccessToken();
}

// API リクエスト時の使用例
async function fetchProtectedResource(path: string) {
  const token = await getAccessToken();
  const res = await fetch(path, {
    headers: { Authorization: \`Bearer \${token}\` },
    credentials: "include",
  });
  if (!res.ok) throw new Error(\`\${res.status}\`);
  return res.json();
}`}</code></pre>

      <h2>ログアウト</h2>
      <p>
        Hydra のログアウトエンドポイントへリダイレクトし、SSO セッションを終了します。
        <code>postLogoutRedirectUri</code> には承認済みの URI を指定してください。
      </p>
      <pre><code>{`import { auth } from "./auth";

export function logout() {
  const logoutUrl = auth.logoutUrl({
    postLogoutRedirectUri: "https://your-app.example.com/",
  });
  window.location.href = logoutUrl;
}`}</code></pre>

      <h2>エラーハンドリング</h2>
      <p>
        SDK のメソッドは失敗時に <code>IdolAuthError</code> をスローします。
        <code>code</code> プロパティでエラー種別を判別できます。
      </p>
      <pre><code>{`import { IdolAuthError, auth } from "./auth";

try {
  const tokens = await auth.handleCallback(window.location.href, {
    expectedState,
    codeVerifier,
  });
} catch (err) {
  if (err instanceof IdolAuthError) {
    switch (err.code) {
      case "state_mismatch":
        // CSRF の可能性。ログインページへ戻す
        window.location.replace("/login");
        break;
      case "token_expired":
        // トークンが期限切れ
        await auth.refreshTokens();
        break;
      case "access_denied":
        // ユーザーが認可を拒否した
        showError("認可がキャンセルされました");
        break;
      default:
        console.error("Auth error:", err.message);
    }
  } else {
    throw err;
  }
}`}</code></pre>

      <h2>React での使い方</h2>
      <p>
        セッション状態を React で管理するカスタムフックの実装例です。
      </p>
      <pre><code>{`// hooks/useSession.ts
import { useEffect, useState } from "react";
import type { SessionView } from "@idol-auth/client";

interface UseSessionResult {
  session: SessionView | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

export function useSession(): UseSessionResult {
  const [session, setSession] = useState<SessionView | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetch("/api/session", { credentials: "include" })
      .then((res) => (res.ok ? res.json() : null))
      .then((data: SessionView | null) => {
        setSession(data?.authenticated ? data : null);
      })
      .catch(() => setSession(null))
      .finally(() => setIsLoading(false));
  }, []);

  return {
    session,
    isLoading,
    isAuthenticated: session?.authenticated ?? false,
  };
}

// 使用例
function ProfilePage() {
  const { session, isLoading, isAuthenticated } = useSession();

  if (isLoading) return <p>読み込み中...</p>;
  if (!isAuthenticated) return <p>ログインが必要です</p>;

  return <p>こんにちは、{session!.display_name}</p>;
}`}</code></pre>

      <h2>PKCE ヘルパー（低レベル API）</h2>
      <p>
        SDK 内部で使用している PKCE ヘルパーを直接インポートできます。
        独自フローを実装する場合や、既存の OAuth ライブラリと組み合わせる場合に利用してください。
      </p>
      <pre><code>{`import {
  generateCodeVerifier,
  generateCodeChallenge,
  generateState,
} from "@idol-auth/client";

const verifier = await generateCodeVerifier(); // 43〜128 文字のランダム文字列
const challenge = await generateCodeChallenge(verifier); // S256 ハッシュ
const state = generateState(); // 32 バイトの乱数を base64url エンコード`}</code></pre>

      <h2>対応スコープ</h2>
      <table>
        <thead>
          <tr>
            <th>スコープ</th>
            <th>含まれるクレーム</th>
            <th>用途</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><code>openid</code></td>
            <td><code>sub</code></td>
            <td>必須。ID Token を発行する</td>
          </tr>
          <tr>
            <td><code>profile</code></td>
            <td><code>name</code>, <code>display_name</code>, <code>oshi_color</code></td>
            <td>プロフィール情報</td>
          </tr>
          <tr>
            <td><code>email</code></td>
            <td><code>email</code>, <code>email_verified</code></td>
            <td>メールアドレス</td>
          </tr>
          <tr>
            <td><code>offline_access</code></td>
            <td>—</td>
            <td>リフレッシュトークンを取得する</td>
          </tr>
        </tbody>
      </table>

      <h2>ESM / CJS</h2>
      <p>
        パッケージは ESM と CJS のデュアルビルドを提供します。
        バンドラー（Vite, webpack, esbuild）環境では自動的に ESM が使用されます。
        Node.js 環境では <code>require()</code> でも <code>import</code> でも読み込めます。
      </p>
    </div>
  )
}
