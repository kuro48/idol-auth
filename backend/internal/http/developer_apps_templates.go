package http

import "html/template"

// developerCredentialsData feeds the one-time credential display page shown
// right after instant self-service registration.
type developerCredentialsData struct {
	devPageBase
	AppName         string
	ClientID        string
	ClientSecret    string
	ManagementToken string
	DetailURL       string
}

// #nosec G101 -- Template labels and placeholders only; credential values are injected at render time.
const developerCredentialsBody = `
  <main class="shell">
    <nav class="breadcrumb">
      <a href="/account/">アカウント</a>
      <span class="sep">›</span>
      <a href="/developer/app-requests">アプリ</a>
      <span class="sep">›</span>
      登録完了
    </nav>
    <div class="card">
      <div class="card-head">
        <h1>「{{.AppName}}」を登録しました</h1>
        <span class="status-badge status-approved">approved</span>
      </div>
      <div class="reviewer-note">
        <h3>このページは一度しか表示されません</h3>
        <p>以下のクレデンシャルは再表示できません。今すぐ安全な場所にコピーして保管してください。</p>
      </div>
      <div class="info-list">
        <div class="info-item"><span class="k">Client ID</span><span class="v"><code>{{.ClientID}}</code></span></div>
        {{if .ClientSecret}}
        <div class="info-item"><span class="k">Client Secret</span><span class="v"><code>{{.ClientSecret}}</code></span></div>
        {{end}}
        <div class="info-item"><span class="k">Management Token</span><span class="v"><code>{{.ManagementToken}}</code></span></div>
      </div>
      <div style="display:flex;gap:10px;flex-wrap:wrap;margin-top:20px">
        <a href="{{.DetailURL}}" class="btn btn-primary">アプリ詳細へ</a>
        <a href="/developer/app-requests" class="btn btn-ghost">一覧に戻る</a>
      </div>
    </div>
    <div class="card">
      <div class="card-head"><h2>次のステップ</h2></div>
      <p style="line-height:1.8;color:var(--muted);margin:0">
        TypeScript SDK（@idol-auth/client）をインストールして、Client ID で認証フローを実装できます。
        詳しくは連携ガイドをご覧ください。
      </p>
    </div>
  </main>
  <footer class="site-footer">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
    <a href="/legal/contact">お問い合わせ</a>
  </footer>
</body>
</html>`

var developerCredentialsTpl = template.Must(template.New("dev-credentials").Parse(developerPageCSS + developerCredentialsBody))
