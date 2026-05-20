package http

import (
	"html/template"
	"strings"
	"time"

	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

var developerUITpl = template.Must(template.New("developer-ui").Funcs(template.FuncMap{
	"formatTime": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("2006-01-02 15:04")
	},
	"statusLabel": func(s appreg.Status) string {
		switch s {
		case appreg.StatusPending:
			return "審査待ち"
		case appreg.StatusUnderReview:
			return "審査中"
		case appreg.StatusChangesRequested:
			return "修正依頼"
		case appreg.StatusApproved:
			return "承認済み"
		case appreg.StatusRejected:
			return "却下"
		case appreg.StatusWithdrawn:
			return "取り下げ"
		}
		return string(s)
	},
	"statusClass": func(s appreg.Status) string {
		switch s {
		case appreg.StatusPending:
			return "st-pending"
		case appreg.StatusUnderReview:
			return "st-review"
		case appreg.StatusChangesRequested:
			return "st-changes"
		case appreg.StatusApproved:
			return "st-approved"
		case appreg.StatusRejected:
			return "st-rejected"
		case appreg.StatusWithdrawn:
			return "st-withdrawn"
		}
		return ""
	},
	"joinLines": func(ss []string) string {
		return strings.Join(ss, "\n")
	},
	"canWithdraw": func(r appreg.Request) bool {
		return r.CanWithdraw()
	},
	"isChangesRequested": func(r appreg.Request) bool {
		return r.Status == appreg.StatusChangesRequested
	},
	"isApproved": func(r appreg.Request) bool {
		return r.Status == appreg.StatusApproved
	},
	"join": func(ss []string, sep string) string {
		return strings.Join(ss, sep)
	},
}).Parse(developerUITemplates))

const developerUITemplates = `
{{define "dev-css"}}
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{--oshi:#ff8a3d;--oshi-weak:#fff1e8;--oshi-soft:#ffd9c2;--bg:#f8f6f3;--surface:#fff;--surface-2:#fffaf6;--text:#26211f;--muted:#776d67;--border:#eadfd7;--shadow:0 16px 40px rgba(48,35,28,.1);--radius-lg:28px;--radius-md:18px;--radius-sm:12px;--ok:#35a67b;--ng:#e85d75;--warn:#e9a23b}
html[data-theme="dark"]{--bg:#181412;--surface:#221d1a;--surface-2:#2b241f;--text:#fff8f1;--muted:#c8b8ad;--border:#42362f;--shadow:0 16px 40px rgba(0,0,0,.35);--oshi-weak:#372219;--oshi-soft:#5a3829}
body{font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Hiragino Sans","Yu Gothic","Noto Sans JP",sans-serif;font-size:14px;background:var(--bg);color:var(--text);min-height:100vh;letter-spacing:.01em}
a{color:var(--oshi);text-decoration:none}a:hover{text-decoration:underline}
.topnav{position:sticky;top:0;z-index:100;background:color-mix(in srgb,var(--surface) 88%,transparent);backdrop-filter:saturate(160%) blur(14px);-webkit-backdrop-filter:saturate(160%) blur(14px);border-bottom:1px solid var(--border);display:flex;align-items:center;padding:14px 28px;gap:12px}
.brand-mark{width:42px;height:42px;border-radius:14px;background:var(--oshi);color:#fff;display:grid;place-items:center;font-weight:900;font-size:20px;box-shadow:0 10px 24px color-mix(in srgb,var(--oshi) 32%,transparent)}
.topnav-brand{font-size:15px;font-weight:900;letter-spacing:-.01em}
.topnav-spacer{flex:1}
.topnav-email{font-size:12px;color:var(--muted);font-weight:800;padding:8px 14px;background:var(--surface-2);border:1px solid var(--border);border-radius:999px;max-width:280px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.topnav a{color:var(--text);font-size:13px;font-weight:800;padding:9px 14px;background:var(--surface);border:1px solid var(--border);border-radius:999px}
.topnav a:hover{background:var(--surface-2);text-decoration:none}
.main{padding:28px 28px 110px;max-width:980px;margin:0 auto}
.page-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:24px;gap:16px;flex-wrap:wrap}
.page-title{font-size:28px;font-weight:900;letter-spacing:-.03em}
.page-sub{color:var(--muted);font-size:13px;margin-top:6px;line-height:1.6}
.back-link{display:inline-flex;align-items:center;gap:6px;font-size:13px;color:var(--muted);margin-bottom:14px;font-weight:800}
.back-link:hover{color:var(--oshi)}
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-lg);box-shadow:var(--shadow);padding:24px 26px;margin-bottom:20px}
.card-title{font-size:11px;font-weight:900;letter-spacing:.1em;text-transform:uppercase;color:var(--muted);margin-bottom:14px}
.card-section{margin-bottom:18px;padding-bottom:18px;border-bottom:1px solid var(--border)}
.card-section:last-child{margin-bottom:0;padding-bottom:0;border-bottom:none}
.card-section-title{font-size:14px;font-weight:900;margin-bottom:12px;color:var(--text)}
.kv{display:grid;grid-template-columns:160px 1fr;gap:10px 16px;font-size:13px}
.kv dt{color:var(--muted);font-weight:800}
.kv dd{color:var(--text);word-break:break-word;white-space:pre-wrap}
.table-wrap{overflow-x:auto}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;padding:12px 14px;background:var(--surface-2);border-top:1px solid var(--border);border-bottom:1px solid var(--border);font-weight:900;color:var(--muted);font-size:11px;text-transform:uppercase;letter-spacing:.08em;white-space:nowrap}
td{padding:14px;border-bottom:1px solid var(--border);vertical-align:middle}
tr:hover td{background:var(--surface-2)}
.empty-row td{text-align:center;color:var(--muted);padding:36px 12px}
.badge{display:inline-flex;align-items:center;padding:5px 11px;border-radius:999px;font-size:11px;font-weight:900;letter-spacing:.03em}
.st-pending{background:color-mix(in srgb,var(--oshi) 14%,transparent);color:var(--oshi)}
.st-review{background:color-mix(in srgb,#8e6be8 16%,transparent);color:#8e6be8}
.st-changes{background:color-mix(in srgb,var(--warn) 16%,transparent);color:var(--warn)}
.st-approved{background:color-mix(in srgb,var(--ok) 16%,transparent);color:var(--ok)}
.st-rejected{background:color-mix(in srgb,var(--ng) 16%,transparent);color:var(--ng)}
.st-withdrawn{background:var(--surface-2);color:var(--muted)}
.btn{display:inline-flex;align-items:center;justify-content:center;gap:8px;padding:11px 16px;border-radius:16px;font-size:13px;font-weight:900;cursor:pointer;border:1px solid transparent;transition:transform 160ms ease,box-shadow 160ms ease,background 160ms ease;text-decoration:none;font-family:inherit;line-height:1.2}
.btn:hover{transform:translateY(-1px);text-decoration:none}
.btn:active{transform:translateY(1px)}
.btn-primary{background:var(--oshi);color:#fff;border-color:var(--oshi);box-shadow:0 12px 24px color-mix(in srgb,var(--oshi) 28%,transparent)}
.btn-secondary{background:var(--surface-2);color:var(--text);border-color:var(--border)}
.btn-danger{background:var(--ng);color:#fff;border-color:var(--ng)}
.btn-sm{padding:8px 12px;font-size:12px;border-radius:12px}
.form-grid{display:grid;gap:14px}
.form-group{display:flex;flex-direction:column;gap:5px}
.form-group label{font-size:12px;font-weight:900;color:var(--muted);letter-spacing:.05em}
.form-group .hint{font-size:11px;color:var(--muted);margin-top:2px}
input[type=text],input[type=email],input[type=url],textarea,select{padding:13px 15px;border:1px solid var(--border);border-radius:16px;font-size:13px;color:var(--text);background:var(--surface-2);font-family:inherit;width:100%;outline:none}
textarea{resize:vertical;min-height:88px;line-height:1.55}
input:focus,select:focus,textarea:focus{border-color:var(--oshi);box-shadow:0 0 0 4px color-mix(in srgb,var(--oshi) 18%,transparent)}
.form-actions{display:flex;gap:10px;margin-top:18px;justify-content:flex-end}
.note-block{background:var(--oshi-weak);border:1px solid var(--border);border-radius:16px;padding:14px 16px;color:var(--text);font-size:13px;line-height:1.6;white-space:pre-wrap}
.toast-area{position:fixed;bottom:20px;right:20px;z-index:300;display:flex;flex-direction:column;gap:8px}
.toast{padding:14px 18px;border-radius:18px;font-size:13px;font-weight:800;color:#fff;box-shadow:var(--shadow);animation:fi .18s ease;max-width:360px}
.toast-success{background:var(--ok)}
.toast-error{background:var(--ng)}
.float-tools{position:fixed;right:22px;bottom:22px;z-index:150;display:flex;align-items:center;gap:10px}
.theme-toggle{width:48px;height:48px;border-radius:50%;background:var(--surface);border:1px solid var(--border);color:var(--text);box-shadow:var(--shadow);display:grid;place-items:center;font-size:20px;cursor:pointer}
.color-dots{display:flex;gap:8px;padding:10px 12px;background:var(--surface);border:1px solid var(--border);border-radius:999px;box-shadow:var(--shadow)}
.color-dot{width:24px;height:24px;border-radius:50%;border:3px solid var(--surface);box-shadow:0 0 0 1px var(--border);padding:0;cursor:pointer}
@keyframes fi{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:none}}
@media (max-width:640px){
.topnav{padding:12px 18px}
.topnav-email{max-width:120px}
.main{padding:22px 18px 110px}
.page-title{font-size:24px}
.kv{grid-template-columns:1fr}
.kv dt{margin-bottom:-6px}
}
</style>
{{end}}

{{define "dev-nav"}}
<nav class="topnav">
  <div class="brand-mark">推</div>
  <div class="topnav-brand">開発者ポータル</div>
  <div class="topnav-spacer"></div>
  <div class="topnav-email">{{.Email}}</div>
  <a href="{{.LogoutURL}}">ログアウト</a>
</nav>
{{end}}

{{define "dev-js"}}
<div class="float-tools" aria-label="推し色を選ぶ">
  <div class="color-dots" role="group" aria-label="推し色を選ぶ">
    <button class="color-dot" type="button" data-color="#ff8a3d" style="background:#ff8a3d" title="オレンジ"></button>
    <button class="color-dot" type="button" data-color="#ff5fa2" style="background:#ff5fa2" title="ピンク"></button>
    <button class="color-dot" type="button" data-color="#7c5cff" style="background:#7c5cff" title="パープル"></button>
    <button class="color-dot" type="button" data-color="#39b58a" style="background:#39b58a" title="グリーン"></button>
    <button class="color-dot" type="button" data-color="#4b7bec" style="background:#4b7bec" title="ブルー"></button>
  </div>
  <button class="theme-toggle" id="theme-toggle" type="button" aria-label="テーマを切り替える" title="テーマ切替">☾</button>
</div>
<div class="toast-area" id="toast-area"></div>
<script>
document.addEventListener('click', function(e){
  var root = document.documentElement;
  var theme = e.target.closest('#theme-toggle');
  if(theme){
    var next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
    root.setAttribute('data-theme', next);
    theme.textContent = next === 'dark' ? '☀' : '☾';
    return;
  }
  var dot = e.target.closest('.color-dot');
  if(dot && dot.dataset.color){
    root.style.setProperty('--oshi', dot.dataset.color);
  }
});
function showToast(msg, type){
  var area = document.getElementById('toast-area');
  if(!area) return;
  var t = document.createElement('div');
  t.className = 'toast toast-' + (type === 'error' ? 'error' : 'success');
  t.textContent = msg;
  area.appendChild(t);
  setTimeout(function(){ t.remove(); }, 3600);
}
function splitLines(v){
  return (v || '').split('\n').map(function(s){ return s.trim(); }).filter(function(s){ return s.length > 0; });
}
function splitScopes(v){
  return (v || '').split(/\s+/).map(function(s){ return s.trim(); }).filter(function(s){ return s.length > 0; });
}
function collectForm(prefix){
  prefix = prefix || '';
  function val(id){ var el = document.getElementById(prefix + id); return el ? el.value.trim() : ''; }
  return {
    name: val('name'),
    type: val('type'),
    description: val('description'),
    homepage_url: val('homepage_url'),
    privacy_policy_url: val('privacy_policy_url'),
    terms_url: val('terms_url'),
    contact_email: val('contact_email'),
    organization: val('organization'),
    purpose: val('purpose'),
    redirect_uris: splitLines(val('redirect_uris')),
    post_logout_redirect_uris: splitLines(val('post_logout_redirect_uris')),
    scopes: splitScopes(val('scopes'))
  };
}
function submitNew(){
  var body = collectForm('');
  fetch('/v1/developer/applications', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    credentials: 'include',
    body: JSON.stringify(body)
  }).then(function(res){
    return res.json().then(function(data){ return {status: res.status, data: data}; });
  }).then(function(r){
    if(r.status === 201 && r.data && r.data.id){
      window.location.href = '/developer/applications/' + r.data.id;
      return;
    }
    var msg = (r.data && r.data.error) ? r.data.error : '申請の送信に失敗しました';
    showToast(msg, 'error');
  }).catch(function(){
    showToast('通信エラーが発生しました', 'error');
  });
}
function withdraw(id){
  if(!window.confirm('この申請を取り下げますか？この操作は元に戻せません。')) return;
  fetch('/v1/developer/applications/' + id, {
    method: 'DELETE',
    credentials: 'include'
  }).then(function(res){
    if(res.status === 200 || res.status === 204){
      window.location.href = '/developer/';
      return;
    }
    return res.json().then(function(data){
      showToast((data && data.error) ? data.error : '取り下げに失敗しました', 'error');
    });
  }).catch(function(){
    showToast('通信エラーが発生しました', 'error');
  });
}
function submitResubmit(id){
  var body = collectForm('rs-');
  fetch('/v1/developer/applications/' + id, {
    method: 'PATCH',
    headers: {'Content-Type': 'application/json'},
    credentials: 'include',
    body: JSON.stringify(body)
  }).then(function(res){
    return res.json().then(function(data){ return {status: res.status, data: data}; });
  }).then(function(r){
    if(r.status === 200){
      showToast('再申請を送信しました', 'success');
      setTimeout(function(){ window.location.reload(); }, 600);
      return;
    }
    var msg = (r.data && r.data.error) ? r.data.error : '再申請に失敗しました';
    showToast(msg, 'error');
  }).catch(function(){
    showToast('通信エラーが発生しました', 'error');
  });
}
</script>
{{end}}

{{define "dev-list"}}
<!DOCTYPE html>
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>申請一覧 | 開発者ポータル</title>{{template "dev-css" .}}</head>
<body>
{{template "dev-nav" .}}
<main class="main">
  <div class="page-header">
    <div>
      <div class="page-title">アプリ申請一覧</div>
      <div class="page-sub">登録したアプリの審査ステータスを確認できます</div>
    </div>
    <a href="/developer/new" class="btn btn-primary">＋ 新規申請</a>
  </div>
  <div class="card">
    <div class="table-wrap">
      <table>
        <thead><tr><th>アプリ名</th><th>タイプ</th><th>ステータス</th><th>申請日</th><th>操作</th></tr></thead>
        <tbody>
        {{if .Requests}}
          {{range .Requests}}
          <tr>
            <td><strong>{{.Name}}</strong></td>
            <td>{{.Type}}</td>
            <td><span class="badge {{statusClass .Status}}">{{statusLabel .Status}}</span></td>
            <td>{{formatTime .CreatedAt}}</td>
            <td><a href="/developer/applications/{{.ID}}" class="btn btn-secondary btn-sm">詳細</a></td>
          </tr>
          {{end}}
        {{else}}
          <tr class="empty-row"><td colspan="5">まだ申請がありません。「＋ 新規申請」から最初のアプリを登録してください。</td></tr>
        {{end}}
        </tbody>
      </table>
    </div>
  </div>
</main>
{{template "dev-js" .}}
</body></html>
{{end}}

{{define "dev-new"}}
<!DOCTYPE html>
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>新規申請 | 開発者ポータル</title>{{template "dev-css" .}}</head>
<body>
{{template "dev-nav" .}}
<main class="main">
  <a href="/developer/" class="back-link">← 一覧へ戻る</a>
  <div class="page-header"><div><div class="page-title">新規アプリ申請</div><div class="page-sub">審査担当者がレビューしやすいよう、各項目を具体的に記入してください</div></div></div>
  <form onsubmit="event.preventDefault(); submitNew();">
    <div class="card">
      <div class="card-section">
        <div class="card-section-title">基本情報</div>
        <div class="form-grid">
          <div class="form-group"><label for="name">アプリ名</label><input type="text" id="name" required></div>
          <div class="form-group"><label for="type">タイプ</label>
            <select id="type"><option value="web">Web</option><option value="spa">SPA</option><option value="native">Native</option><option value="m2m">M2M</option></select>
          </div>
          <div class="form-group"><label for="description">概要</label><textarea id="description" required></textarea></div>
          <div class="form-group"><label for="purpose">利用目的</label><textarea id="purpose" required></textarea><span class="hint">200文字以上で具体的にご記入ください</span></div>
        </div>
      </div>
      <div class="card-section">
        <div class="card-section-title">連絡先</div>
        <div class="form-grid">
          <div class="form-group"><label for="contact_email">連絡先メール</label><input type="email" id="contact_email" required></div>
          <div class="form-group"><label for="organization">組織名（任意）</label><input type="text" id="organization"></div>
        </div>
      </div>
      <div class="card-section">
        <div class="card-section-title">URL情報</div>
        <div class="form-grid">
          <div class="form-group"><label for="homepage_url">ホームページURL（任意）</label><input type="text" id="homepage_url"><span class="hint">https:// から始まるURLを入力</span></div>
          <div class="form-group"><label for="privacy_policy_url">プライバシーポリシーURL（任意）</label><input type="text" id="privacy_policy_url"></div>
          <div class="form-group"><label for="terms_url">利用規約URL（任意）</label><input type="text" id="terms_url"></div>
        </div>
      </div>
      <div class="card-section">
        <div class="card-section-title">OAuth設定</div>
        <div class="form-grid">
          <div class="form-group"><label for="redirect_uris">リダイレクトURI</label><textarea id="redirect_uris" placeholder="https://example.com/callback" required></textarea><span class="hint">1行に1つずつ入力</span></div>
          <div class="form-group"><label for="post_logout_redirect_uris">ログアウト後リダイレクトURI（任意）</label><textarea id="post_logout_redirect_uris"></textarea></div>
          <div class="form-group"><label for="scopes">スコープ</label><input type="text" id="scopes" value="openid"><span class="hint">スペース区切り。デフォルト: openid</span></div>
        </div>
      </div>
    </div>
    <div class="form-actions">
      <a href="/developer/" class="btn btn-secondary">キャンセル</a>
      <button type="submit" class="btn btn-primary">申請を送信</button>
    </div>
  </form>
</main>
{{template "dev-js" .}}
</body></html>
{{end}}

{{define "dev-detail"}}
<!DOCTYPE html>
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>{{.Request.Name}} | 開発者ポータル</title>{{template "dev-css" .}}</head>
<body>
{{template "dev-nav" .}}
<main class="main">
  <a href="/developer/" class="back-link">← 一覧へ戻る</a>
  <div class="page-header">
    <div>
      <div class="page-title">{{.Request.Name}} <span class="badge {{statusClass .Request.Status}}" style="margin-left:8px;vertical-align:middle">{{statusLabel .Request.Status}}</span></div>
      <div class="page-sub">申請ID: {{.Request.ID}} ・ Version {{.Request.Version}}</div>
    </div>
  </div>

  <div class="card">
    <div class="card-title">申請内容</div>
    <dl class="kv">
      <dt>タイプ</dt><dd>{{.Request.Type}}</dd>
      <dt>概要</dt><dd>{{.Request.Description}}</dd>
      <dt>利用目的</dt><dd>{{.Request.Purpose}}</dd>
      <dt>連絡先メール</dt><dd>{{.Request.ContactEmail}}</dd>
      <dt>組織</dt><dd>{{if .Request.Organization}}{{.Request.Organization}}{{else}}—{{end}}</dd>
      <dt>ホームページ</dt><dd>{{if .Request.HomepageURL}}<a href="{{.Request.HomepageURL}}" target="_blank" rel="noopener">{{.Request.HomepageURL}}</a>{{else}}—{{end}}</dd>
      <dt>プライバシーポリシー</dt><dd>{{if .Request.PrivacyPolicyURL}}<a href="{{.Request.PrivacyPolicyURL}}" target="_blank" rel="noopener">{{.Request.PrivacyPolicyURL}}</a>{{else}}—{{end}}</dd>
      <dt>利用規約</dt><dd>{{if .Request.TermsURL}}<a href="{{.Request.TermsURL}}" target="_blank" rel="noopener">{{.Request.TermsURL}}</a>{{else}}—{{end}}</dd>
      <dt>リダイレクトURI</dt><dd>{{joinLines .Request.RedirectURIs}}</dd>
      <dt>ログアウトURI</dt><dd>{{if .Request.PostLogoutRedirectURIs}}{{joinLines .Request.PostLogoutRedirectURIs}}{{else}}—{{end}}</dd>
      <dt>スコープ</dt><dd>{{join .Request.Scopes " "}}</dd>
      <dt>申請日時</dt><dd>{{formatTime .Request.CreatedAt}}</dd>
      <dt>更新日時</dt><dd>{{formatTime .Request.UpdatedAt}}</dd>
    </dl>
  </div>

  {{if .Request.ReviewerNote}}
  <div class="card">
    <div class="card-title">レビュアーコメント</div>
    <div class="note-block">{{.Request.ReviewerNote}}</div>
  </div>
  {{end}}

  {{if isApproved .Request}}
  <div class="card">
    <div class="card-title">発行された認可情報</div>
    <dl class="kv">
      <dt>App ID</dt><dd>{{if .Request.CreatedAppID}}{{.Request.CreatedAppID}}{{else}}—{{end}}</dd>
      <dt>Client ID</dt><dd>{{if .Request.CreatedClientID}}{{.Request.CreatedClientID}}{{else}}—{{end}}</dd>
    </dl>
  </div>
  {{end}}

  {{if canWithdraw .Request}}
  <div class="card">
    <div class="card-title">操作</div>
    <p style="font-size:13px;color:var(--sub);margin-bottom:14px">審査前または修正依頼中の申請は取り下げることができます。</p>
    <button type="button" class="btn btn-danger" onclick="withdraw('{{.Request.ID}}')">申請を取り下げる</button>
  </div>
  {{end}}

  {{if isChangesRequested .Request}}
  <div class="card">
    <div class="card-title">修正して再申請</div>
    <p style="font-size:13px;color:var(--sub);margin-bottom:14px">レビュアーからの指摘を反映したうえで、再申請してください。</p>
    <form onsubmit="event.preventDefault(); submitResubmit('{{.Request.ID}}');">
      <div class="card-section">
        <div class="card-section-title">基本情報</div>
        <div class="form-grid">
          <div class="form-group"><label for="rs-name">アプリ名</label><input type="text" id="rs-name" value="{{.Request.Name}}" required></div>
          <div class="form-group"><label for="rs-type">タイプ</label>
            <select id="rs-type">
              <option value="web"{{if eq .Request.Type "web"}} selected{{end}}>Web</option>
              <option value="spa"{{if eq .Request.Type "spa"}} selected{{end}}>SPA</option>
              <option value="native"{{if eq .Request.Type "native"}} selected{{end}}>Native</option>
              <option value="m2m"{{if eq .Request.Type "m2m"}} selected{{end}}>M2M</option>
            </select>
          </div>
          <div class="form-group"><label for="rs-description">概要</label><textarea id="rs-description" required>{{.Request.Description}}</textarea></div>
          <div class="form-group"><label for="rs-purpose">利用目的</label><textarea id="rs-purpose" required>{{.Request.Purpose}}</textarea><span class="hint">200文字以上で具体的にご記入ください</span></div>
        </div>
      </div>
      <div class="card-section">
        <div class="card-section-title">連絡先</div>
        <div class="form-grid">
          <div class="form-group"><label for="rs-contact_email">連絡先メール</label><input type="email" id="rs-contact_email" value="{{.Request.ContactEmail}}" required></div>
          <div class="form-group"><label for="rs-organization">組織名（任意）</label><input type="text" id="rs-organization" value="{{.Request.Organization}}"></div>
        </div>
      </div>
      <div class="card-section">
        <div class="card-section-title">URL情報</div>
        <div class="form-grid">
          <div class="form-group"><label for="rs-homepage_url">ホームページURL（任意）</label><input type="text" id="rs-homepage_url" value="{{.Request.HomepageURL}}"></div>
          <div class="form-group"><label for="rs-privacy_policy_url">プライバシーポリシーURL（任意）</label><input type="text" id="rs-privacy_policy_url" value="{{.Request.PrivacyPolicyURL}}"></div>
          <div class="form-group"><label for="rs-terms_url">利用規約URL（任意）</label><input type="text" id="rs-terms_url" value="{{.Request.TermsURL}}"></div>
        </div>
      </div>
      <div class="card-section">
        <div class="card-section-title">OAuth設定</div>
        <div class="form-grid">
          <div class="form-group"><label for="rs-redirect_uris">リダイレクトURI</label><textarea id="rs-redirect_uris" required>{{joinLines .Request.RedirectURIs}}</textarea><span class="hint">1行に1つずつ入力</span></div>
          <div class="form-group"><label for="rs-post_logout_redirect_uris">ログアウト後リダイレクトURI（任意）</label><textarea id="rs-post_logout_redirect_uris">{{joinLines .Request.PostLogoutRedirectURIs}}</textarea></div>
          <div class="form-group"><label for="rs-scopes">スコープ</label><input type="text" id="rs-scopes" value="{{join .Request.Scopes " "}}"><span class="hint">スペース区切り</span></div>
        </div>
      </div>
      <div class="form-actions">
        <button type="submit" class="btn btn-primary">再申請を送信</button>
      </div>
    </form>
  </div>
  {{end}}
</main>
{{template "dev-js" .}}
</body></html>
{{end}}

{{define "dev-error"}}
<!DOCTYPE html>
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>エラー | 開発者ポータル</title>
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
body{font-family:"Hiragino Sans","Yu Gothic","Noto Sans JP",system-ui,sans-serif;background:#f0f2f5;color:#1a1a1a;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px}
.err-card{background:#fff;border:1px solid #d4d6db;border-radius:4px;padding:36px 40px;max-width:520px;text-align:center}
.err-title{font-size:18px;font-weight:700;margin-bottom:12px;color:#bf0000}
.err-msg{color:#595959;font-size:14px;line-height:1.7;margin-bottom:20px}
.err-link{color:#1740c9;font-size:13px;text-decoration:none}
.err-link:hover{text-decoration:underline}
</style>
</head>
<body>
<div class="err-card">
  <div class="err-title">エラー</div>
  <div class="err-msg">{{.Message}}</div>
  <a href="/developer/" class="err-link">← 開発者ポータルへ戻る</a>
</div>
</body></html>
{{end}}
`
