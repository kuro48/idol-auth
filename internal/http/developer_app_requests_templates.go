package http

import "html/template"

const developerPageCSS = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>アプリ申請 | OshiLink Developer</title>
  <style>
    :root{
      --oshi:{{.OshiColor}};
      --oshi-weak:#fff1e8;
      --oshi-soft:#ffd9c2;
      --bg:#f8f6f3;
      --surface:#ffffff;
      --surface-2:#fffaf6;
      --text:#26211f;
      --muted:#776d67;
      --border:#eadfd7;
      --shadow:0 16px 40px rgba(48,35,28,.1);
      --radius-lg:28px;
      --radius-md:18px;
      --radius-sm:12px;
      --danger:#e85d75;
      --success:#35a67b;
      --warning:#e9a23b;
    }
    @media(prefers-color-scheme:dark){
      :root{
        --bg:#181412;--surface:#221d1a;--surface-2:#2b241f;--text:#fff8f1;--muted:#c8b8ad;
        --border:#42362f;--shadow:0 16px 40px rgba(0,0,0,.35);--oshi-weak:#372219;--oshi-soft:#5a3829;
      }
    }
    *,*::before,*::after{box-sizing:border-box}
    html,body{margin:0;padding:0}
    body{
      min-height:100vh;
      font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Hiragino Sans","Yu Gothic","Noto Sans JP",sans-serif;
      color:var(--text);background:var(--bg);letter-spacing:.01em;
    }
    button,input,select,textarea{font:inherit;color:inherit}
    button{cursor:pointer}
    a{color:var(--oshi);text-decoration:none}
    a:hover{text-decoration:underline}
    .site-footer{padding:24px 28px 40px;display:flex;flex-wrap:wrap;gap:8px 24px;border-top:1px solid var(--border);margin-top:8px}
    .site-footer a{font-size:12px;color:var(--muted);transition:color .15s}
    .site-footer a:hover{color:var(--oshi)}
    .topbar{
      position:sticky;top:0;z-index:20;
      display:flex;align-items:center;justify-content:space-between;
      gap:16px;padding:14px 28px;
      background:color-mix(in srgb,var(--surface) 88%,transparent);
      backdrop-filter:saturate(160%) blur(14px);
      -webkit-backdrop-filter:saturate(160%) blur(14px);
      border-bottom:1px solid var(--border);
    }
    .brand{display:flex;align-items:center;gap:12px}
    .brand-mark{
      width:42px;height:42px;border-radius:14px;
      background:var(--oshi);color:#fff;
      display:grid;place-items:center;
      font-weight:900;font-size:20px;letter-spacing:-.02em;
      box-shadow:0 10px 24px color-mix(in srgb,var(--oshi) 32%,transparent);
    }
    .brand-text strong{display:block;font-size:16px;line-height:1.1;letter-spacing:-.01em}
    .brand-text span{font-size:11px;color:var(--muted);font-weight:700;text-transform:uppercase;letter-spacing:.12em}
    .top-actions{display:flex;align-items:center;gap:12px}
    .user-chip{
      display:flex;align-items:center;gap:10px;
      padding:6px 12px 6px 6px;border-radius:999px;
      background:var(--surface-2);border:1px solid var(--border);
      font-size:13px;color:var(--text);max-width:280px;
    }
    .user-chip .dot{
      width:32px;height:32px;border-radius:50%;
      background:var(--oshi);color:#fff;
      display:grid;place-items:center;font-weight:900;font-size:13px;
    }
    .user-chip .email{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:200px;color:var(--muted)}
    .logout-btn{
      border:1px solid var(--border);background:var(--surface);
      color:var(--text);padding:9px 14px;border-radius:999px;
      font-weight:800;font-size:13px;
      transition:transform 160ms ease,background 160ms ease;
    }
    .logout-btn:hover{transform:translateY(-1px);background:var(--surface-2)}
    .shell{max-width:860px;margin:0 auto;padding:28px 28px 120px}
    .breadcrumb{display:flex;align-items:center;gap:8px;font-size:13px;color:var(--muted);margin-bottom:20px;flex-wrap:wrap}
    .breadcrumb a{color:var(--muted)}
    .breadcrumb a:hover{color:var(--oshi)}
    .breadcrumb .sep{opacity:.45}
    .card{
      background:var(--surface);border:1px solid var(--border);
      border-radius:var(--radius-lg);box-shadow:var(--shadow);padding:24px;margin-bottom:20px;
    }
    .card-head{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:18px;flex-wrap:wrap}
    .card-head h2{margin:0;font-size:20px;font-weight:700;letter-spacing:-.01em}
    .card-head h1{margin:0;font-size:24px;font-weight:800;letter-spacing:-.02em}
    .page-actions{display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-bottom:28px}
    .field{display:grid;gap:6px;margin-bottom:18px}
    .field label{font-size:12px;font-weight:800;color:var(--muted);letter-spacing:.05em}
    .field input[type=text],.field input[type=email],.field select,.field textarea{
      width:100%;border:1px solid var(--border);background:var(--surface-2);
      color:var(--text);padding:14px 16px;border-radius:16px;
      outline:none;transition:border-color 160ms ease,box-shadow 160ms ease;
      resize:vertical;
    }
    .field input:focus,.field select:focus,.field textarea:focus{
      border-color:var(--oshi);
      box-shadow:0 0 0 4px color-mix(in srgb,var(--oshi) 18%,transparent);
    }
    .field-note{font-size:11px;color:var(--muted);line-height:1.5;margin-top:4px}
    .form-error{
      background:color-mix(in srgb,var(--danger) 10%,transparent);
      border:1px solid color-mix(in srgb,var(--danger) 30%,transparent);
      color:var(--danger);border-radius:var(--radius-sm);
      padding:12px 16px;font-size:13px;font-weight:700;margin-bottom:18px;
    }
    .form-actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:6px}
    .btn{
      border:0;border-radius:16px;padding:14px 20px;
      font-weight:900;font-size:14px;line-height:1;
      display:inline-flex;align-items:center;gap:8px;
      transition:transform 160ms ease,box-shadow 160ms ease,opacity 160ms ease;
      cursor:pointer;
    }
    .btn:hover{transform:translateY(-1px);text-decoration:none}
    .btn-primary{background:var(--oshi);color:#fff;box-shadow:0 10px 24px color-mix(in srgb,var(--oshi) 32%,transparent)}
    .btn-primary:hover{opacity:.88}
    .btn-ghost{background:var(--surface-2);color:var(--text);border:1px solid var(--border)}
    .btn-ghost:hover{background:var(--surface)}
    .btn-danger{background:color-mix(in srgb,var(--danger) 12%,transparent);color:var(--danger);border:1px solid color-mix(in srgb,var(--danger) 25%,transparent)}
    .btn-danger:hover{background:color-mix(in srgb,var(--danger) 18%,transparent)}
    .btn-sm{padding:9px 14px;font-size:13px;border-radius:12px}
    .status-badge{
      display:inline-flex;align-items:center;gap:6px;
      padding:5px 12px;border-radius:999px;
      font-size:11px;font-weight:800;letter-spacing:.06em;flex-shrink:0;
    }
    .status-pending{background:color-mix(in srgb,var(--warning) 14%,transparent);color:var(--warning)}
    .status-under_review{background:color-mix(in srgb,#6d82ff 14%,transparent);color:#4a5fd9}
    .status-approved{background:color-mix(in srgb,var(--success) 14%,transparent);color:var(--success)}
    .status-rejected{background:color-mix(in srgb,var(--danger) 12%,transparent);color:var(--danger)}
    .status-withdrawn{background:color-mix(in srgb,var(--muted) 14%,transparent);color:var(--muted)}
    .status-changes_requested{background:color-mix(in srgb,#e96c2e 12%,transparent);color:#c0501c}
    .req-row{
      display:flex;align-items:center;justify-content:space-between;
      gap:14px;padding:18px;border-radius:var(--radius-md);
      background:var(--surface-2);border:1px solid var(--border);margin-bottom:12px;
      color:var(--text);
      transition:box-shadow 160ms ease,transform 160ms ease;
      text-decoration:none;
    }
    .req-row:hover{box-shadow:0 8px 24px rgba(48,35,28,.12);transform:translateY(-1px);text-decoration:none}
    .req-row-left{flex:1;min-width:0}
    .req-row-name{font-size:16px;font-weight:700;letter-spacing:-.01em;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
    .req-row-meta{font-size:12px;color:var(--muted);margin-top:4px}
    .empty-state{text-align:center;padding:60px 28px;color:var(--muted)}
    .empty-state p{font-size:15px;margin:0 0 20px}
    .info-list{display:grid;gap:10px}
    .info-item{
      display:flex;justify-content:space-between;align-items:flex-start;gap:14px;
      padding:14px;border-radius:16px;background:var(--surface-2);border:1px solid var(--border);
    }
    .info-item .k{font-size:11px;font-weight:800;color:var(--muted);text-transform:uppercase;letter-spacing:.1em;white-space:nowrap;padding-top:2px}
    .info-item .v{font-size:14px;color:var(--text);font-weight:600;text-align:right;word-break:break-all;max-width:68%}
    .reviewer-note{
      background:color-mix(in srgb,var(--warning) 8%,transparent);
      border:1px solid color-mix(in srgb,var(--warning) 25%,transparent);
      border-radius:var(--radius-md);padding:18px;margin-bottom:20px;
    }
    .reviewer-note h3{margin:0 0 8px;font-size:13px;font-weight:800;color:var(--warning);letter-spacing:.05em;text-transform:uppercase}
    .reviewer-note p{margin:0;font-size:14px;line-height:1.7;color:var(--text)}
    @media(max-width:640px){
      .shell{padding:16px 16px 80px}
      .topbar{padding:12px 16px}
      .card{padding:18px}
      .req-row{flex-direction:column;align-items:flex-start}
    }
  </style>
</head>
<body>
  <header class="topbar">
    <div class="brand">
      <a href="/account/" class="brand-mark">O</a>
      <div class="brand-text">
        <strong>OshiLink</strong>
        <span>Developer</span>
      </div>
    </div>
    <div class="top-actions">
      <div class="user-chip">
        <div class="dot">{{.Initials}}</div>
        <span class="email">{{.Email}}</span>
      </div>
      <a href="{{.LogoutURL}}" class="logout-btn">ログアウト</a>
    </div>
  </header>`

const developerListBody = `
  <main class="shell">
    <nav class="breadcrumb">
      <a href="/account/">アカウント</a>
      <span class="sep">›</span>
      <span>アプリ申請</span>
    </nav>
    <div class="page-actions">
      <h1 style="margin:0;font-size:24px;font-weight:800;letter-spacing:-.02em;flex:1">アプリ申請一覧</h1>
      <a href="/developer/app-requests/new" class="btn btn-primary">＋ 新規申請</a>
    </div>
    {{if .Requests}}
      {{range .Requests}}
      <a href="/developer/app-requests/{{.ID}}" class="req-row">
        <div class="req-row-left">
          <div class="req-row-name">{{.Name}}</div>
          <div class="req-row-meta">{{.Type}} · 申請日 {{.CreatedAt.Format "2006-01-02"}}</div>
        </div>
        <span class="status-badge status-{{.Status}}">{{.Status}}</span>
      </a>
      {{end}}
    {{else}}
      <div class="empty-state">
        <p>申請はまだありません</p>
        <a href="/developer/app-requests/new" class="btn btn-primary">最初の申請を作成する</a>
      </div>
    {{end}}
  </main>
  <footer class="site-footer">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
    <a href="/legal/contact">お問い合わせ</a>
  </footer>
</body>
</html>`

const developerFormBody = `
  <main class="shell">
    <nav class="breadcrumb">
      <a href="/account/">アカウント</a>
      <span class="sep">›</span>
      <a href="/developer/app-requests">アプリ申請</a>
      <span class="sep">›</span>
      {{if .Req}}修正・再申請{{else}}新規申請{{end}}
    </nav>
    <div class="card">
      <div class="card-head">
        <h1>{{if .Req}}修正・再申請{{else}}新規アプリ申請{{end}}</h1>
      </div>
      {{if .Error}}<div class="form-error">{{.Error}}</div>{{end}}
      <form method="post" action="{{if .Req}}/developer/app-requests/{{.Req.ID}}/resubmit{{else}}/developer/app-requests{{end}}">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <div class="field">
          <label for="name">アプリ名 <span style="color:var(--danger)">*</span></label>
          <input type="text" id="name" name="name" required maxlength="100"
            value="{{if .Req}}{{.Req.Name}}{{end}}" placeholder="例：My Awesome App">
        </div>
        <div class="field">
          <label for="type">アプリ種別 <span style="color:var(--danger)">*</span></label>
          <select id="type" name="type" required>
            <option value="">選択してください</option>
            {{$t := ""}}{{if .Req}}{{$t = .Req.Type}}{{end}}
            <option value="web"{{if eq $t "web"}} selected{{end}}>web — サーバーサイドWebアプリ</option>
            <option value="spa"{{if eq $t "spa"}} selected{{end}}>spa — シングルページアプリ（JS）</option>
            <option value="native"{{if eq $t "native"}} selected{{end}}>native — モバイル・デスクトップアプリ</option>
            <option value="m2m"{{if eq $t "m2m"}} selected{{end}}>m2m — マシン間通信（ユーザーなし）</option>
          </select>
        </div>
        <div class="field">
          <label for="description">アプリ説明 <span style="color:var(--danger)">*</span></label>
          <textarea id="description" name="description" required rows="3" maxlength="1000"
            placeholder="アプリの概要を1〜1000文字で記入してください">{{if .Req}}{{.Req.Description}}{{end}}</textarea>
        </div>
        <div class="field">
          <label for="purpose">利用目的 <span style="color:var(--danger)">*</span></label>
          <textarea id="purpose" name="purpose" required rows="6" minlength="200" maxlength="2000"
            placeholder="OshiLinkアカウントをどのように利用するかを200〜2000文字で詳しく説明してください">{{if .Req}}{{.Req.Purpose}}{{end}}</textarea>
          <div class="field-note">最低200文字・最大2000文字</div>
        </div>
        <div class="field">
          <label for="contact_email">連絡先メール <span style="color:var(--danger)">*</span></label>
          <input type="email" id="contact_email" name="contact_email" required
            value="{{if .Req}}{{.Req.ContactEmail}}{{end}}" placeholder="developer@example.com">
        </div>
        <div class="field">
          <label for="organization">組織名</label>
          <input type="text" id="organization" name="organization" maxlength="200"
            value="{{if .Req}}{{.Req.Organization}}{{end}}" placeholder="例：株式会社Example（任意）">
        </div>
        <div class="field">
          <label for="homepage_url">ホームページURL</label>
          <input type="text" id="homepage_url" name="homepage_url"
            value="{{if .Req}}{{.Req.HomepageURL}}{{end}}" placeholder="https://example.com（任意）">
        </div>
        <div class="field">
          <label for="privacy_policy_url">プライバシーポリシーURL</label>
          <input type="text" id="privacy_policy_url" name="privacy_policy_url"
            value="{{if .Req}}{{.Req.PrivacyPolicyURL}}{{end}}" placeholder="https://example.com/privacy（任意）">
        </div>
        <div class="field">
          <label for="terms_url">利用規約URL</label>
          <input type="text" id="terms_url" name="terms_url"
            value="{{if .Req}}{{.Req.TermsURL}}{{end}}" placeholder="https://example.com/terms（任意）">
        </div>
        <div class="field">
          <label for="redirect_uris">リダイレクトURI（1行に1つ）</label>
          <textarea id="redirect_uris" name="redirect_uris" rows="4"
            placeholder="https://example.com/callback">{{if .Req}}{{range $i,$u := .Req.RedirectURIs}}{{if $i}}&#10;{{end}}{{$u}}{{end}}{{end}}</textarea>
          <div class="field-note">m2m以外は必須。https または http://localhost のみ使用可</div>
        </div>
        <div class="field">
          <label for="post_logout_redirect_uris">ログアウト後リダイレクトURI（1行に1つ）</label>
          <textarea id="post_logout_redirect_uris" name="post_logout_redirect_uris" rows="3"
            placeholder="https://example.com/（任意）">{{if .Req}}{{range $i,$u := .Req.PostLogoutRedirectURIs}}{{if $i}}&#10;{{end}}{{$u}}{{end}}{{end}}</textarea>
        </div>
        <div class="field">
          <label for="scopes">要求スコープ（1行に1つ）</label>
          <textarea id="scopes" name="scopes" rows="3"
            placeholder="openid&#10;profile&#10;email">{{if .Req}}{{range $i,$s := .Req.Scopes}}{{if $i}}&#10;{{end}}{{$s}}{{end}}{{end}}</textarea>
          <div class="field-note">任意。空白の場合はデフォルトスコープが付与されます</div>
        </div>
        <div class="form-actions">
          <button type="submit" class="btn btn-primary">{{if .Req}}再申請する{{else}}申請を提出する{{end}}</button>
          <a href="{{if .Req}}/developer/app-requests/{{.Req.ID}}{{else}}/developer/app-requests{{end}}" class="btn btn-ghost">キャンセル</a>
        </div>
      </form>
    </div>
  </main>
  <footer class="site-footer">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
    <a href="/legal/contact">お問い合わせ</a>
  </footer>
</body>
</html>`

const developerDetailBody = `
  <main class="shell">
    <nav class="breadcrumb">
      <a href="/account/">アカウント</a>
      <span class="sep">›</span>
      <a href="/developer/app-requests">アプリ申請</a>
      <span class="sep">›</span>
      {{.Req.Name}}
    </nav>
    <div class="card">
      <div class="card-head">
        <h1>{{.Req.Name}}</h1>
        <span class="status-badge status-{{.Req.Status}}">{{.Req.Status}}</span>
      </div>
      {{if .Req.ReviewerNote}}
      <div class="reviewer-note">
        <h3>レビュアーからのコメント</h3>
        <p>{{.Req.ReviewerNote}}</p>
      </div>
      {{end}}
      <div class="info-list" style="margin-bottom:20px">
        <div class="info-item"><span class="k">種別</span><span class="v">{{.Req.Type}}</span></div>
        <div class="info-item"><span class="k">説明</span><span class="v">{{.Req.Description}}</span></div>
        <div class="info-item"><span class="k">連絡先メール</span><span class="v">{{.Req.ContactEmail}}</span></div>
        {{if .Req.Organization}}<div class="info-item"><span class="k">組織名</span><span class="v">{{.Req.Organization}}</span></div>{{end}}
        {{if .Req.HomepageURL}}<div class="info-item"><span class="k">HP URL</span><span class="v"><a href="{{.Req.HomepageURL}}" target="_blank" rel="noopener">{{.Req.HomepageURL}}</a></span></div>{{end}}
        {{if .Req.PrivacyPolicyURL}}<div class="info-item"><span class="k">プライバシーポリシー</span><span class="v"><a href="{{.Req.PrivacyPolicyURL}}" target="_blank" rel="noopener">{{.Req.PrivacyPolicyURL}}</a></span></div>{{end}}
        {{if .Req.TermsURL}}<div class="info-item"><span class="k">利用規約</span><span class="v"><a href="{{.Req.TermsURL}}" target="_blank" rel="noopener">{{.Req.TermsURL}}</a></span></div>{{end}}
        {{if .Req.RedirectURIs}}<div class="info-item"><span class="k">リダイレクトURI</span><span class="v">{{range .Req.RedirectURIs}}<div>{{.}}</div>{{end}}</span></div>{{end}}
        {{if .Req.PostLogoutRedirectURIs}}<div class="info-item"><span class="k">ログアウトURI</span><span class="v">{{range .Req.PostLogoutRedirectURIs}}<div>{{.}}</div>{{end}}</span></div>{{end}}
        {{if .Req.Scopes}}<div class="info-item"><span class="k">スコープ</span><span class="v">{{range .Req.Scopes}}<div>{{.}}</div>{{end}}</span></div>{{end}}
        <div class="info-item"><span class="k">申請日</span><span class="v">{{.Req.CreatedAt.Format "2006-01-02 15:04"}}</span></div>
        <div class="info-item"><span class="k">更新日</span><span class="v">{{.Req.UpdatedAt.Format "2006-01-02 15:04"}}</span></div>
        {{if .Req.DecidedAt}}<div class="info-item"><span class="k">決定日</span><span class="v">{{.Req.DecidedAt.Format "2006-01-02 15:04"}}</span></div>{{end}}
      </div>
      <div style="display:flex;gap:10px;flex-wrap:wrap">
        {{if .Req.CanWithdraw}}
        <a href="/developer/app-requests/{{.Req.ID}}/edit" class="btn btn-ghost">修正して再申請</a>
        <form method="post" action="/developer/app-requests/{{.Req.ID}}/withdraw" style="display:inline">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <button type="submit" class="btn btn-danger btn-sm"
            onclick="return confirm('この申請を取り下げますか？')">取り下げる</button>
        </form>
        {{end}}
        <a href="/developer/app-requests" class="btn btn-ghost btn-sm">一覧に戻る</a>
      </div>
    </div>
    {{if .Req.Purpose}}
    <div class="card">
      <div class="card-head"><h2>利用目的</h2></div>
      <p style="line-height:1.8;color:var(--muted);margin:0">{{.Req.Purpose}}</p>
    </div>
    {{end}}
  </main>
  <footer class="site-footer">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
    <a href="/legal/contact">お問い合わせ</a>
  </footer>
</body>
</html>`

var developerRequestListTpl = template.Must(template.New("dev-list").Parse(developerPageCSS + developerListBody))
var developerRequestFormTpl = template.Must(template.New("dev-form").Parse(developerPageCSS + developerFormBody))
var developerRequestDetailTpl = template.Must(template.New("dev-detail").Parse(developerPageCSS + developerDetailBody))
