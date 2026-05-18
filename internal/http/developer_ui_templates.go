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
:root{--primary:#1740c9;--primary-dark:#0017c1;--bg:#f0f2f5;--card:#fff;--border:#d4d6db;--text:#1a1a1a;--sub:#595959;--ok:#007b50;--ng:#bf0000;--warn:#e85500;--sw:56px}
body{font-family:"Hiragino Sans","Yu Gothic","Noto Sans JP",system-ui,sans-serif;font-size:14px;background:var(--bg);color:var(--text);min-height:100vh}
a{color:var(--primary);text-decoration:none}a:hover{text-decoration:underline}
.topnav{position:fixed;top:0;left:0;right:0;z-index:100;height:52px;background:var(--primary-dark);color:#fff;display:flex;align-items:center;padding:0 24px;gap:16px}
.topnav-brand{font-size:15px;font-weight:700;letter-spacing:.06em}
.topnav-spacer{flex:1}
.topnav-email{font-size:12px;opacity:.85}
.topnav a{color:#fff;font-size:12px;opacity:.85}
.topnav a:hover{opacity:1;text-decoration:underline}
.main{padding:80px 24px 64px;max-width:900px;margin:0 auto}
.page-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:24px;gap:16px;flex-wrap:wrap}
.page-title{font-size:22px;font-weight:700;letter-spacing:-.01em}
.page-sub{color:var(--sub);font-size:13px;margin-top:4px}
.back-link{display:inline-flex;align-items:center;gap:6px;font-size:13px;color:var(--sub);margin-bottom:14px}
.back-link:hover{color:var(--primary)}
.card{background:var(--card);border:1px solid var(--border);border-radius:4px;padding:22px 26px;margin-bottom:18px}
.card-title{font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:var(--sub);margin-bottom:14px}
.card-section{margin-bottom:16px;padding-bottom:16px;border-bottom:1px solid #eef0f4}
.card-section:last-child{margin-bottom:0;padding-bottom:0;border-bottom:none}
.card-section-title{font-size:13px;font-weight:700;margin-bottom:10px;color:var(--text)}
.kv{display:grid;grid-template-columns:160px 1fr;gap:10px 16px;font-size:13px}
.kv dt{color:var(--sub);font-weight:600}
.kv dd{color:var(--text);word-break:break-word;white-space:pre-wrap}
.table-wrap{overflow-x:auto}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;padding:10px 12px;background:#f7f8fb;border-bottom:2px solid var(--border);font-weight:600;color:var(--sub);white-space:nowrap}
td{padding:10px 12px;border-bottom:1px solid #ececec;vertical-align:middle}
tr:hover td{background:#fafbfd}
.empty-row td{text-align:center;color:#999;padding:36px 12px}
.badge{display:inline-flex;align-items:center;padding:3px 10px;border-radius:999px;font-size:11px;font-weight:700;letter-spacing:.02em}
.st-pending{background:#e8ecff;color:#1740c9}
.st-review{background:#efe8ff;color:#5a26c0}
.st-changes{background:#fff1e0;color:#c25500}
.st-approved{background:#e6f4ef;color:#007b50}
.st-rejected{background:#fce8e8;color:#bf0000}
.st-withdrawn{background:#ececec;color:#595959}
.btn{display:inline-flex;align-items:center;justify-content:center;gap:6px;padding:8px 16px;border-radius:4px;font-size:13px;font-weight:600;cursor:pointer;border:1px solid transparent;transition:opacity .12s,transform .06s;text-decoration:none;font-family:inherit;line-height:1.2}
.btn:hover{opacity:.86;text-decoration:none}
.btn:active{transform:translateY(1px)}
.btn-primary{background:var(--primary);color:#fff;border-color:var(--primary)}
.btn-secondary{background:#fff;color:var(--text);border-color:var(--border)}
.btn-danger{background:var(--ng);color:#fff;border-color:var(--ng)}
.btn-sm{padding:4px 10px;font-size:12px}
.form-grid{display:grid;gap:14px}
.form-group{display:flex;flex-direction:column;gap:5px}
.form-group label{font-size:12px;font-weight:700;color:var(--sub);letter-spacing:.02em}
.form-group .hint{font-size:11px;color:#888;margin-top:2px}
input[type=text],input[type=email],input[type=url],textarea,select{padding:9px 11px;border:1px solid var(--border);border-radius:4px;font-size:13px;color:var(--text);background:#fff;font-family:inherit;width:100%}
textarea{resize:vertical;min-height:88px;line-height:1.55}
input:focus,select:focus,textarea:focus{outline:2px solid var(--primary);outline-offset:-1px}
.form-actions{display:flex;gap:10px;margin-top:18px;justify-content:flex-end}
.note-block{background:#fff8ec;border:1px solid #f0d9a5;border-radius:4px;padding:14px 16px;color:#7a4a00;font-size:13px;line-height:1.6;white-space:pre-wrap}
.toast-area{position:fixed;bottom:20px;right:20px;z-index:300;display:flex;flex-direction:column;gap:8px}
.toast{padding:11px 16px;border-radius:3px;font-size:13px;font-weight:600;color:#fff;box-shadow:0 4px 16px rgba(0,0,0,.2);animation:fi .18s ease;max-width:360px}
.toast-success{background:var(--ok)}
.toast-error{background:var(--ng)}
@keyframes fi{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:none}}
@media (max-width:640px){
.main{padding:72px 14px 48px}
.kv{grid-template-columns:1fr}
.kv dt{margin-bottom:-6px}
}
</style>
{{end}}

{{define "dev-nav"}}
<nav class="topnav">
  <div class="topnav-brand">開発者ポータル</div>
  <div class="topnav-spacer"></div>
  <div class="topnav-email">{{.Email}}</div>
  <a href="{{.LogoutURL}}">ログアウト</a>
</nav>
{{end}}

{{define "dev-js"}}
<div class="toast-area" id="toast-area"></div>
<script>
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
