import styles from '../docs/DocsLayout.module.css'

export function DocsConceptsPage() {
  return (
    <div className={styles.prose}>
      <h1>Core Concepts — 共有アカウントとアプリ連携</h1>
      <p>
        idol-auth では、ユーザーの本人アカウントと、各アプリでの利用関係を分けて扱います。
        アプリは必要な範囲の認可を受け、ユーザー本人はアカウントセンターから連携を管理できます。
      </p>
      <h2>主要な用語</h2>
      <table>
        <thead><tr><th>用語</th><th>説明</th></tr></thead>
        <tbody>
          <tr><td><strong>共有アカウント</strong></td><td>ユーザーが idol-auth 上で持つ本人アカウント。メール、パスワード、MFA、基本プロフィールを管理します。</td></tr>
          <tr><td><strong>アプリ</strong></td><td>idol-auth に接続する第三者サービス。アプリごとに Client ID と redirect URI を持ちます。</td></tr>
          <tr><td><strong>連携 (Membership)</strong></td><td>ユーザーとアプリの接続関係。ユーザーがアプリにログインすると作成されます。</td></tr>
          <tr><td><strong>Management Token</strong></td><td>アプリが自分のユーザー連携を管理するためのサーバー側トークン。他アプリには影響しません。</td></tr>
          <tr><td><strong>PKCE</strong></td><td>コードインターセプト攻撃を防ぐための拡張。SPA・ネイティブアプリでは必須です。</td></tr>
        </tbody>
      </table>
      <h2>認証フローの概要</h2>
      <ol>
        <li>ユーザーがアプリの「ログイン」ボタンを押す</li>
        <li>アプリが Authorization URL を生成し、idol-auth にリダイレクト</li>
        <li>ユーザーが idol-auth でログイン・同意</li>
        <li>idol-auth がアプリの redirect URI にコードを返す</li>
        <li>アプリがコードをトークンに交換</li>
        <li>アプリが ID Token・Access Token を使用</li>
      </ol>
      <h2>スコープ</h2>
      <p>アプリが要求できるスコープは登録時に指定します。最小限必要なスコープのみ要求してください。</p>
      <ul>
        <li><code>openid</code> — 認証。ID Token を取得します（必須）</li>
        <li><code>profile</code> — 表示名・アバター</li>
        <li><code>email</code> — メールアドレス</li>
      </ul>
    </div>
  )
}
