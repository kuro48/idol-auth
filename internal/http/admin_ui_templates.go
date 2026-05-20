package http

// Callers: admin_ui.go — adminUITpl.ExecuteTemplate for each admin UI handler.
// CSS lives in admin_ui_styles.go (const adminUIStyles) and is concatenated below.
// User instruction: 「./idol_common_account_dashboard.htmlを参考にして、ログイン画面やアカウント登録画面、管理者ダッシュボードを作成してほしい」

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	admindomain "github.com/kuro48/idol-auth/internal/domain/admin"
)

var adminUITpl = template.Must(template.New("admin-ui").Funcs(template.FuncMap{
	"formatTime": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("2006-01-02 15:04")
	},
	"truncate": func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n] + "…"
	},
	"join": strings.Join,
	// badgeClass uses interface{} because named string types (audit.Result, AppStatus, etc.)
	// are not AssignableTo string in Go reflection; fmt.Sprint handles them correctly.
	"badgeClass": func(v interface{}) string {
		switch strings.ToLower(fmt.Sprint(v)) {
		case "active":
			return "badge-active"
		case "inactive":
			return "badge-inactive"
		case "success":
			return "badge-success"
		case "failure":
			return "badge-failure"
		case "rotated":
			return "badge-rotated"
		case "pending":
			return "badge-pending"
		case "under_review":
			return "badge-review"
		case "changes_requested":
			return "badge-changes"
		case "approved":
			return "badge-success"
		case "rejected":
			return "badge-failure"
		case "withdrawn":
			return "badge-inactive"
		default:
			return ""
		}
	},
	"firstLogTime": func(logs []admindomain.AuditLog) string {
		if len(logs) == 0 {
			return "—"
		}
		return logs[0].OccurredAt.Format("01/02 15:04")
	},
}).Parse(adminUIStyles + adminUITemplates))

const adminUITemplates = `
{{define "topnav"}}
<header class="topnav">
  <div class="topnav-mark">★</div>
  <div class="topnav-brand">
    <strong>OshiLink Admin</strong>
    <span>idol-auth console</span>
  </div>
  <span class="topnav-spacer"></span>
  <span class="topnav-email">{{.Email}}</span>
</header>
{{end}}

{{define "sidebar"}}
<aside class="sidebar">
  <div class="brand">
    <div class="brand-mark">★</div>
    <div class="brand-text">
      <strong>OshiLink Admin</strong>
      <span>Common Account Console</span>
    </div>
  </div>
  <div>
    <div class="nav-section-title">ダッシュボード</div>
    <div class="nav-list">
      <a class="nav-link{{if eq .Nav "overview"}} active{{end}}" href="/admin-ui/"><span class="nav-icon">📊</span>概要</a>
      <a class="nav-link{{if eq .Nav "apps"}} active{{end}}" href="/admin-ui/apps"><span class="nav-icon">🧩</span>アプリ</a>
      <a class="nav-link{{if eq .Nav "users"}} active{{end}}" href="/admin-ui/users"><span class="nav-icon">👥</span>ユーザー</a>
    </div>
  </div>
  <div>
    <div class="nav-section-title">運用</div>
    <div class="nav-list">
      <a class="nav-link{{if eq .Nav "audit-logs"}} active{{end}}" href="/admin-ui/audit-logs"><span class="nav-icon">📜</span>監査ログ</a>
      <a class="nav-link{{if eq .Nav "app-requests"}} active{{end}}" href="/admin-ui/app-requests"><span class="nav-icon">📝</span>申請管理</a>
    </div>
  </div>
  <div class="sidebar-footer">
    <div class="theme-row">
      <strong>表示設定</strong>
      <button type="button" class="theme-toggle" id="themeToggle">🌙 Dark</button>
    </div>
    <div class="color-dots" aria-label="推しメンカラー">
      <button type="button" class="color-dot" style="background:#ff8a3d" data-color="#ff8a3d" title="Orange"></button>
      <button type="button" class="color-dot" style="background:#ef6aa8" data-color="#ef6aa8" title="Pink"></button>
      <button type="button" class="color-dot" style="background:#5a8dee" data-color="#5a8dee" title="Blue"></button>
      <button type="button" class="color-dot" style="background:#35a67b" data-color="#35a67b" title="Green"></button>
      <button type="button" class="color-dot" style="background:#8e6be8" data-color="#8e6be8" title="Purple"></button>
    </div>
  </div>
</aside>
{{end}}

{{define "token-modal"}}
<div class="modal-overlay" id="token-modal">
  <div class="modal">
    <div class="modal-title">管理者トークンの確認</div>
    <div class="modal-desc">この操作には Bootstrap Token が必要です。入力したトークンはセッション中のみ保持されます。</div>
    <div class="form-group"><label>Bootstrap Token</label>
      <input type="password" id="token-input" placeholder="トークンを入力">
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeModal('token-modal')">キャンセル</button>
      <button class="btn btn-primary" onclick="confirmToken()">確認</button>
    </div>
  </div>
</div>
{{end}}

{{define "base-js"}}
<div class="toast-area" id="toast-area"></div>
<script>
var TOKEN_KEY='idol_auth_admin_token';
var THEME_KEY='idol_auth_admin_theme';
var OSHI_KEY='idol_auth_admin_oshi';
var _tokenCb=null;
function getToken(){return sessionStorage.getItem(TOKEN_KEY);}
function clearToken(){sessionStorage.removeItem(TOKEN_KEY);}
function closeModal(id){var el=document.getElementById(id);if(el)el.classList.remove('open');}
function openModal(id){var el=document.getElementById(id);if(el)el.classList.add('open');}
function confirmToken(){
  var t=document.getElementById('token-input').value.trim();
  if(!t)return;
  sessionStorage.setItem(TOKEN_KEY,t);
  closeModal('token-modal');
  if(_tokenCb){var cb=_tokenCb;_tokenCb=null;cb(t);}
}
function showToast(msg,type){
  var area=document.getElementById('toast-area');
  if(!area)return;
  var el=document.createElement('div');
  el.className='toast toast-'+(type||'success');
  el.textContent=msg;
  area.appendChild(el);
  setTimeout(function(){el.remove();},3500);
}
function adminFetch(method,path,body){
  var token=getToken();
  if(!token){
    return new Promise(function(resolve){
      _tokenCb=function(){resolve(adminFetch(method,path,body));};
      openModal('token-modal');
      document.getElementById('token-input').value='';
      setTimeout(function(){document.getElementById('token-input').focus();},80);
    });
  }
  var opts={method:method,headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},credentials:'same-origin'};
  if(body!=null)opts.body=JSON.stringify(body);
  return fetch(path,opts).then(function(res){
    if(res.status===401||res.status===403){
      clearToken();
      return res.json().then(function(d){throw new Error(d.error||'認証エラー ('+res.status+')');});
    }
    return res;
  });
}
(function(){
  var root=document.documentElement;
  var savedTheme=localStorage.getItem(THEME_KEY)||'light';
  root.setAttribute('data-theme',savedTheme);
  var savedOshi=localStorage.getItem(OSHI_KEY);
  if(savedOshi)root.style.setProperty('--oshi',savedOshi);
  function updateLabel(){
    var t=document.getElementById('themeToggle');
    if(!t)return;
    t.textContent=root.getAttribute('data-theme')==='dark'?'☀️ Light':'🌙 Dark';
  }
  document.addEventListener('click',function(e){
    var t=e.target.closest('#themeToggle');
    if(t){
      var next=root.getAttribute('data-theme')==='dark'?'light':'dark';
      root.setAttribute('data-theme',next);
      localStorage.setItem(THEME_KEY,next);
      updateLabel();
      return;
    }
    var dot=e.target.closest('.color-dot');
    if(dot&&dot.dataset.color){
      root.style.setProperty('--oshi',dot.dataset.color);
      localStorage.setItem(OSHI_KEY,dot.dataset.color);
    }
  });
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',updateLabel);
  else updateLabel();
})();
</script>
{{end}}

{{define "overview"}}<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>概要 — OshiLink Admin</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header">
    <div class="page-title-block">
      <h1>概要ダッシュボード</h1>
      <p>登録アプリ・監査イベント・利用状況をひと目で把握できます。</p>
    </div>
  </div>
  <div class="metric-grid">
    <article class="metric-card">
      <div class="metric-top"><span>登録アプリ</span><div class="metric-icon">🧩</div></div>
      <strong>{{.AppCount}}</strong><p>連携可能なアプリ総数</p>
    </article>
    <article class="metric-card">
      <div class="metric-top"><span>直近イベント</span><div class="metric-icon">⚡</div></div>
      <strong>{{len .RecentLogs}}</strong><p>過去20件までの監査</p>
    </article>
    <article class="metric-card">
      <div class="metric-top"><span>最終イベント</span><div class="metric-icon">🕒</div></div>
      <strong style="font-size:22px">{{firstLogTime .RecentLogs}}</strong><p>最新の監査ログ時刻</p>
    </article>
    <article class="metric-card">
      <div class="metric-top"><span>システム状態</span><div class="metric-icon">💚</div></div>
      <strong style="font-size:22px;color:var(--success)">Healthy</strong><p>稼働中サービス</p>
    </article>
  </div>
  <div class="table-card">
    <div class="table-head">
      <div>
        <h3>最近の監査ログ</h3>
        <p>直近のイベントを20件まで表示します。</p>
      </div>
      <a class="btn btn-secondary btn-sm" href="/admin-ui/audit-logs">すべて見る</a>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>日時</th><th>イベント</th><th>アクター</th><th>対象 ID</th><th>結果</th></tr></thead>
        <tbody>
          {{range .RecentLogs}}
          <tr>
            <td style="white-space:nowrap">{{formatTime .OccurredAt}}</td>
            <td>{{.EventType}}</td>
            <td>{{.ActorID}}</td>
            <td><code style="font-size:11px">{{truncate .TargetID 24}}</code></td>
            <td><span class="badge {{badgeClass .Result}}">{{.Result}}</span></td>
          </tr>
          {{else}}<tr class="empty-row"><td colspan="5">監査ログはありません</td></tr>{{end}}
        </tbody>
      </table>
    </div>
  </div>
</main>
</div>
{{template "token-modal" .}}
{{template "base-js" .}}
</body></html>
{{end}}

{{define "apps"}}<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>アプリ — OshiLink Admin</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header">
    <div class="page-title-block">
      <h1>アプリ管理</h1>
      <p>共通アカウントと連携するアプリと OIDC クライアントを作成・確認します。</p>
    </div>
    <button class="btn btn-primary" onclick="openModal('app-modal')">＋ アプリ作成</button>
  </div>
  <div class="table-card">
    <div class="table-wrap">
      <table>
        <thead><tr><th>名前</th><th>スラグ</th><th>タイプ</th><th>パーティ</th><th>ステータス</th><th>作成日</th><th>操作</th></tr></thead>
        <tbody>
          {{range .Apps}}
          <tr>
            <td><strong>{{.Name}}</strong></td>
            <td><code>{{.Slug}}</code></td>
            <td>{{.Type}}</td>
            <td>{{.PartyType}}</td>
            <td><span class="badge {{badgeClass .Status}}">{{.Status}}</span></td>
            <td>{{formatTime .CreatedAt}}</td>
            <td><button class="btn btn-secondary btn-sm" data-app-id="{{.ID}}" onclick="openClientModal(this)">クライアント作成</button></td>
          </tr>
          {{else}}<tr class="empty-row"><td colspan="7">アプリはありません</td></tr>{{end}}
        </tbody>
      </table>
    </div>
  </div>
</main>
</div>

<div class="modal-overlay" id="app-modal">
  <div class="modal">
    <div class="modal-title">アプリ作成</div>
    <div class="form-group"><label>名前 *</label><input type="text" id="app-name" placeholder="例: My App"></div>
    <div class="form-group"><label>スラグ *</label><input type="text" id="app-slug" placeholder="例: my-app"></div>
    <div class="form-group"><label>タイプ</label>
      <select id="app-type"><option value="web">web</option><option value="spa">spa</option><option value="native">native</option><option value="m2m">m2m</option></select>
    </div>
    <div class="form-group"><label>パーティタイプ</label>
      <select id="app-party"><option value="first_party">first_party</option><option value="third_party">third_party</option></select>
    </div>
    <div class="form-group"><label>リダイレクト URI（カンマ区切り、任意）</label>
      <input type="text" id="app-uris" placeholder="https://app.example.com/callback">
    </div>
    <div class="form-group"><label>スコープ（カンマ区切り、任意）</label>
      <input type="text" id="app-scopes" placeholder="openid profile email">
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeModal('app-modal')">キャンセル</button>
      <button class="btn btn-primary" onclick="submitCreateApp()">作成</button>
    </div>
  </div>
</div>

<div class="modal-overlay" id="client-modal">
  <input type="hidden" id="client-app-id">
  <div class="modal">
    <div class="modal-title">OIDC クライアント作成</div>
    <div class="form-group"><label>クライアントタイプ</label>
      <select id="client-type"><option value="public">public</option><option value="confidential">confidential</option></select>
    </div>
    <div class="form-group"><label>リダイレクト URI *（カンマ区切り）</label>
      <input type="text" id="client-uris" placeholder="https://app.example.com/callback">
    </div>
    <div class="form-group"><label>スコープ（カンマ区切り、任意）</label>
      <input type="text" id="client-scopes" placeholder="openid profile email">
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeModal('client-modal')">キャンセル</button>
      <button class="btn btn-primary" onclick="submitCreateClient()">作成</button>
    </div>
  </div>
</div>

{{template "token-modal" .}}
{{template "base-js" .}}
<script>
function openClientModal(btn){
  document.getElementById('client-app-id').value=btn.getAttribute('data-app-id');
  openModal('client-modal');
}
async function submitCreateApp(){
  var name=document.getElementById('app-name').value.trim();
  var slug=document.getElementById('app-slug').value.trim();
  if(!name||!slug){showToast('名前とスラグは必須です','error');return;}
  var uris=document.getElementById('app-uris').value.trim().split(/[\s,]+/).filter(Boolean);
  var scopes=document.getElementById('app-scopes').value.trim().split(/[\s,]+/).filter(Boolean);
  var body={name:name,slug:slug,type:document.getElementById('app-type').value,party_type:document.getElementById('app-party').value};
  if(uris.length>0){body.redirect_uris=uris;body.scopes=scopes.length>0?scopes:['openid'];}
  try{
    var res=await adminFetch('POST','/v1/admin/apps',body);
    var d=await res.json();
    if(!res.ok){showToast('エラー: '+(d.error||res.status),'error');return;}
    var token=d.management_token?' 管理用トークン: '+d.management_token:'';
    showToast('アプリを作成しました。'+token);
    closeModal('app-modal');
    setTimeout(function(){location.reload();},700);
  }catch(e){showToast(e.message,'error');}
}
async function submitCreateClient(){
  var appId=document.getElementById('client-app-id').value;
  var uris=document.getElementById('client-uris').value.trim().split(/[\s,]+/).filter(Boolean);
  if(uris.length===0){showToast('リダイレクト URI は必須です','error');return;}
  var scopes=document.getElementById('client-scopes').value.trim().split(/[\s,]+/).filter(Boolean);
  try{
    var res=await adminFetch('POST','/v1/admin/apps/'+appId+'/clients',{
      client_type:document.getElementById('client-type').value,
      redirect_uris:uris,
      scopes:scopes.length>0?scopes:['openid']
    });
    var d=await res.json();
    if(!res.ok){showToast('エラー: '+(d.error||res.status),'error');return;}
    var secret=d.client_secret?' シークレット: '+d.client_secret:'';
    showToast('クライアントを作成しました。'+secret);
    closeModal('client-modal');
  }catch(e){showToast(e.message,'error');}
}
</script>
</body></html>
{{end}}

{{define "users"}}<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>ユーザー — OshiLink Admin</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header">
    <div class="page-title-block">
      <h1>ユーザー管理</h1>
      <p>共通アカウントの利用者を検索し、状態・ロール・セッションを操作します。</p>
    </div>
  </div>
  <form method="get" action="/admin-ui/users" class="form-row">
    <div class="form-group"><label>メール / 識別子</label>
      <input type="text" name="q" value="{{.Query}}" placeholder="user@example.com">
    </div>
    <div class="form-group"><label>ステータス</label>
      <select name="state">
        <option value="">すべて</option>
        <option value="active"{{if eq .State "active"}} selected{{end}}>有効</option>
        <option value="inactive"{{if eq .State "inactive"}} selected{{end}}>無効</option>
      </select>
    </div>
    <button type="submit" class="btn btn-primary">検索</button>
    <a href="/admin-ui/users" class="btn btn-secondary">クリア</a>
  </form>
  <div class="table-card">
    <div class="table-wrap">
      <table>
        <thead><tr><th>メール / Phone</th><th>ID</th><th>ステータス</th><th>ロール</th><th>操作</th></tr></thead>
        <tbody>
          {{range .Users}}
          <tr>
            <td>{{if .Email}}{{.Email}}{{else}}{{.Phone}}{{end}}</td>
            <td><code style="font-size:11px">{{truncate .ID 18}}</code></td>
            <td><span class="badge {{badgeClass .State}}">{{.State}}</span></td>
            <td>{{if .Roles}}<span class="role-pill">{{join .Roles ", "}}</span>{{else}}<span style="color:var(--muted);opacity:.5">—</span>{{end}}</td>
            <td>
              <div class="actions-cell">
                {{if eq .State "active"}}
                <button class="btn btn-secondary btn-sm" data-id="{{.ID}}" onclick="userAction(this,'disable')">無効化</button>
                {{else}}
                <button class="btn btn-secondary btn-sm" data-id="{{.ID}}" onclick="userAction(this,'enable')">有効化</button>
                {{end}}
                <button class="btn btn-secondary btn-sm" data-id="{{.ID}}" onclick="userAction(this,'revoke')">セッション失効</button>
                <button class="btn btn-secondary btn-sm" data-id="{{.ID}}" data-roles="{{join .Roles ","}}" onclick="userAction(this,'roles')">ロール</button>
                <button class="btn btn-danger btn-sm" data-id="{{.ID}}" onclick="userAction(this,'delete')">削除</button>
              </div>
            </td>
          </tr>
          {{else}}<tr class="empty-row"><td colspan="5">ユーザーが見つかりません</td></tr>{{end}}
        </tbody>
      </table>
    </div>
  </div>
</main>
</div>

<div class="modal-overlay" id="roles-modal">
  <input type="hidden" id="roles-user-id">
  <div class="modal">
    <div class="modal-title">ロール設定</div>
    <div class="modal-desc">カンマ区切りでロールを入力してください。空欄にするとロールが削除されます。</div>
    <div class="form-group"><label>ロール</label>
      <input type="text" id="roles-input" placeholder="admin, viewer">
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeModal('roles-modal')">キャンセル</button>
      <button class="btn btn-primary" onclick="saveRoles()">保存</button>
    </div>
  </div>
</div>

{{template "token-modal" .}}
{{template "base-js" .}}
<script>
async function userAction(btn,action){
  var id=btn.getAttribute('data-id');
  if(action==='roles'){
    document.getElementById('roles-user-id').value=id;
    document.getElementById('roles-input').value=btn.getAttribute('data-roles')||'';
    openModal('roles-modal');
    return;
  }
  var confirms={
    disable:'このユーザーを無効化しますか？',
    delete:'このユーザーを完全に削除しますか？この操作は取り消せません。',
    revoke:'全セッションを失効させますか？'
  };
  if(confirms[action]&&!confirm(confirms[action]))return;
  try{
    var res;
    if(action==='disable')res=await adminFetch('PATCH','/v1/admin/users/'+id,{state:'inactive'});
    else if(action==='enable')res=await adminFetch('PATCH','/v1/admin/users/'+id,{state:'active'});
    else if(action==='revoke')res=await adminFetch('POST','/v1/admin/users/'+id+'/revoke-sessions',null);
    else if(action==='delete')res=await adminFetch('DELETE','/v1/admin/users/'+id,null);
    if(res.status!==204&&!res.ok){
      var d=await res.json().catch(function(){return {};});
      throw new Error(d.error||'HTTP '+res.status);
    }
    var msgs={disable:'無効化しました',enable:'再有効化しました',revoke:'セッションを失効させました',delete:'削除しました'};
    showToast(msgs[action]||'完了');
    if(action!=='revoke')setTimeout(function(){location.reload();},700);
  }catch(e){showToast(e.message,'error');}
}
async function saveRoles(){
  var id=document.getElementById('roles-user-id').value;
  var raw=document.getElementById('roles-input').value.trim();
  var roles=raw?raw.split(/[\s,]+/).filter(Boolean):[];
  try{
    var res=await adminFetch('PATCH','/v1/admin/users/'+id,{roles:roles});
    if(!res.ok){
      var d=await res.json().catch(function(){return {};});
      throw new Error(d.error||'HTTP '+res.status);
    }
    showToast('ロールを更新しました');
    closeModal('roles-modal');
    setTimeout(function(){location.reload();},700);
  }catch(e){showToast(e.message,'error');}
}
</script>
</body></html>
{{end}}

{{define "audit-logs"}}<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>監査ログ — OshiLink Admin</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header">
    <div class="page-title-block">
      <h1>監査ログ</h1>
      <p>システムで発生したイベントを絞り込みながら追跡します。</p>
    </div>
  </div>
  <form method="get" action="/admin-ui/audit-logs" class="form-row">
    <div class="form-group"><label>イベントタイプ</label>
      <select name="event_type">
        <option value="">すべて</option>
        <option value="app.created"{{if eq .EventType "app.created"}} selected{{end}}>app.created</option>
        <option value="oidc_client.created"{{if eq .EventType "oidc_client.created"}} selected{{end}}>oidc_client.created</option>
        <option value="identity.roles.updated"{{if eq .EventType "identity.roles.updated"}} selected{{end}}>identity.roles.updated</option>
        <option value="identity.disabled"{{if eq .EventType "identity.disabled"}} selected{{end}}>identity.disabled</option>
        <option value="identity.enabled"{{if eq .EventType "identity.enabled"}} selected{{end}}>identity.enabled</option>
        <option value="identity.deleted"{{if eq .EventType "identity.deleted"}} selected{{end}}>identity.deleted</option>
        <option value="identity.sessions.revoked"{{if eq .EventType "identity.sessions.revoked"}} selected{{end}}>identity.sessions.revoked</option>
      </select>
    </div>
    <div class="form-group"><label>アクター ID</label>
      <input type="text" name="actor_id" value="{{.ActorID}}" placeholder="admin@example.com">
    </div>
    <div class="form-group"><label>対象タイプ</label>
      <select name="target_type">
        <option value="">すべて</option>
        <option value="app"{{if eq .TargetType "app"}} selected{{end}}>app</option>
        <option value="client"{{if eq .TargetType "client"}} selected{{end}}>client</option>
        <option value="user"{{if eq .TargetType "user"}} selected{{end}}>user</option>
      </select>
    </div>
    <button type="submit" class="btn btn-primary">絞り込み</button>
    <a href="/admin-ui/audit-logs" class="btn btn-secondary">クリア</a>
  </form>
  <div class="table-card">
    <div class="table-wrap">
      <table>
        <thead><tr><th>日時</th><th>イベント</th><th>アクター</th><th>対象 ID</th><th>結果</th><th>IP</th></tr></thead>
        <tbody>
          {{range .Logs}}
          <tr>
            <td style="white-space:nowrap">{{formatTime .OccurredAt}}</td>
            <td>{{.EventType}}</td>
            <td>{{.ActorID}}</td>
            <td><code style="font-size:11px">{{truncate .TargetID 22}}</code></td>
            <td><span class="badge {{badgeClass .Result}}">{{.Result}}</span></td>
            <td style="font-size:11px;color:var(--muted)">{{.IPAddress}}</td>
          </tr>
          {{else}}<tr class="empty-row"><td colspan="6">ログはありません</td></tr>{{end}}
        </tbody>
      </table>
    </div>
    {{if .HasMore}}<p style="padding:14px 24px;font-size:12px;color:var(--muted);border-top:1px solid var(--border);background:var(--surface-2)">表示上限（50件）に達しました。絞り込みで件数を減らしてください。</p>{{end}}
  </div>
</main>
</div>
{{template "token-modal" .}}
{{template "base-js" .}}
</body></html>
{{end}}

{{define "error"}}<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>エラー — OshiLink Admin</title>{{template "head" .}}</head>
<body style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--bg);padding:24px">
  <div style="background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-lg);box-shadow:var(--shadow);padding:36px 40px;max-width:480px;text-align:center">
    <div style="width:56px;height:56px;border-radius:18px;background:var(--oshi-weak);color:var(--oshi);font-size:28px;display:grid;place-items:center;margin:0 auto 18px">⚠️</div>
    <div style="font-size:20px;font-weight:900;margin-bottom:10px;letter-spacing:-0.02em">{{.Title}}</div>
    <p style="color:var(--muted);font-size:14px;line-height:1.7">{{.Msg}}</p>
  </div>
</body></html>
{{end}}

{{define "app-requests"}}<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>申請管理 — OshiLink Admin</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header">
    <div class="page-title-block">
      <h1>アプリ申請管理</h1>
      <p>サードパーティ申請の審査・承認・修正依頼を行います。</p>
    </div>
  </div>
  <form method="get" action="/admin-ui/app-requests" class="form-row">
    <div class="form-group"><label>ステータス</label>
      <select name="status">
        <option value=""{{if eq .StatusFilter ""}} selected{{end}}>全て</option>
        <option value="pending"{{if eq .StatusFilter "pending"}} selected{{end}}>審査待ち</option>
        <option value="under_review"{{if eq .StatusFilter "under_review"}} selected{{end}}>審査中</option>
        <option value="changes_requested"{{if eq .StatusFilter "changes_requested"}} selected{{end}}>修正依頼</option>
        <option value="approved"{{if eq .StatusFilter "approved"}} selected{{end}}>承認済み</option>
        <option value="rejected"{{if eq .StatusFilter "rejected"}} selected{{end}}>却下</option>
      </select>
    </div>
    <button type="submit" class="btn btn-primary">絞り込み</button>
    <a href="/admin-ui/app-requests" class="btn btn-secondary">クリア</a>
  </form>
  <div class="table-card">
    <div class="table-wrap">
      <table>
        <thead><tr><th>申請名</th><th>タイプ</th><th>申請者ID</th><th>ステータス</th><th>申請日</th><th>操作</th></tr></thead>
        <tbody>
          {{range .Requests}}
          <tr>
            <td><strong>{{.Name}}</strong>{{if .Slug}} <span style="font-size:11px;color:var(--muted)">/ {{.Slug}}</span>{{end}}</td>
            <td>{{if .Type}}{{.Type}}{{else}}<span style="color:var(--muted);opacity:.5">—</span>{{end}}</td>
            <td><code style="font-size:11px">{{truncate .IdentityID 18}}</code></td>
            <td><span class="badge {{badgeClass .Status}}">{{.Status}}</span></td>
            <td style="white-space:nowrap">{{formatTime .CreatedAt}}</td>
            <td><a class="btn btn-secondary btn-sm" href="/admin-ui/app-requests/{{.ID}}">詳細</a></td>
          </tr>
          {{else}}<tr class="empty-row"><td colspan="6">申請はありません</td></tr>{{end}}
        </tbody>
      </table>
    </div>
  </div>
</main>
</div>
{{template "token-modal" .}}
{{template "base-js" .}}
</body></html>
{{end}}

{{define "app-request-detail"}}<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Request.Name}} — 申請管理</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <a class="back-link" href="/admin-ui/app-requests">← 申請一覧へ戻る</a>
  <div class="page-header">
    <div class="page-title-block">
      <div class="detail-title-row">
        <h1>{{.Request.Name}}</h1>
        <span class="badge {{badgeClass .Request.Status}}">{{.Request.Status}}</span>
      </div>
      <p>申請内容を確認し、承認・修正依頼・却下を行えます。</p>
    </div>
  </div>
  <div class="card">
    <div class="card-title">申請内容</div>
    <table class="kv-table">
      <tbody>
        <tr><th>申請ID</th><td><code style="font-size:11px">{{.Request.ID}}</code></td></tr>
        <tr><th>申請者ID</th><td><code style="font-size:11px">{{.Request.IdentityID}}</code></td></tr>
        <tr><th>名前</th><td>{{.Request.Name}}</td></tr>
        <tr><th>スラグ</th><td>{{if .Request.Slug}}<code>{{.Request.Slug}}</code>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>タイプ</th><td>{{if .Request.Type}}{{.Request.Type}}{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>説明</th><td>{{if .Request.Description}}{{.Request.Description}}{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>用途</th><td>{{if .Request.Purpose}}{{.Request.Purpose}}{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>組織</th><td>{{if .Request.Organization}}{{.Request.Organization}}{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>連絡先メール</th><td>{{if .Request.ContactEmail}}{{.Request.ContactEmail}}{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>ホームページ</th><td>{{if .Request.HomepageURL}}<a href="{{.Request.HomepageURL}}" target="_blank" rel="noopener noreferrer">{{.Request.HomepageURL}}</a>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>プライバシーポリシー</th><td>{{if .Request.PrivacyPolicyURL}}<a href="{{.Request.PrivacyPolicyURL}}" target="_blank" rel="noopener noreferrer">{{.Request.PrivacyPolicyURL}}</a>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>利用規約</th><td>{{if .Request.TermsURL}}<a href="{{.Request.TermsURL}}" target="_blank" rel="noopener noreferrer">{{.Request.TermsURL}}</a>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>リダイレクト URI</th><td>{{if .Request.RedirectURIs}}<ul class="uri-list">{{range .Request.RedirectURIs}}<li><code>{{.}}</code></li>{{end}}</ul>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>ログアウト後 URI</th><td>{{if .Request.PostLogoutRedirectURIs}}<ul class="uri-list">{{range .Request.PostLogoutRedirectURIs}}<li><code>{{.}}</code></li>{{end}}</ul>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>スコープ</th><td>{{if .Request.Scopes}}<code style="font-size:12px">{{join .Request.Scopes ", "}}</code>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>レビュー担当</th><td>{{if .Request.ReviewerID}}<code style="font-size:11px">{{.Request.ReviewerID}}</code>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>レビュー メモ</th><td>{{if .Request.ReviewerNote}}{{.Request.ReviewerNote}}{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>決定日時</th><td>{{if .Request.DecidedAt}}{{formatTime .Request.DecidedAt.UTC}}{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>作成済みアプリ ID</th><td>{{if .Request.CreatedAppID}}<code style="font-size:11px">{{.Request.CreatedAppID}}</code>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>作成済みクライアント ID</th><td>{{if .Request.CreatedClientID}}<code style="font-size:11px">{{.Request.CreatedClientID}}</code>{{else}}<span class="kv-empty">—</span>{{end}}</td></tr>
        <tr><th>バージョン</th><td>{{.Request.Version}}</td></tr>
        <tr><th>作成日時</th><td>{{formatTime .Request.CreatedAt}}</td></tr>
        <tr><th>更新日時</th><td>{{formatTime .Request.UpdatedAt}}</td></tr>
      </tbody>
    </table>
    <div class="detail-actions">
      <button class="btn btn-approve" onclick="openModal('approve-modal')">承認</button>
      <button class="btn btn-changes" onclick="openModal('changes-modal')">修正依頼</button>
      <button class="btn btn-danger" onclick="openModal('reject-modal')">却下</button>
    </div>
  </div>
</main>
</div>

<div class="modal-overlay" id="approve-modal">
  <div class="modal">
    <div class="modal-title">申請を承認</div>
    <div class="modal-desc">アプリと OIDC クライアントが作成されます。パーティタイプを選択してください。</div>
    <div class="form-group"><label>パーティタイプ</label>
      <select id="approve-party-type">
        <option value="first_party">first_party</option>
        <option value="third_party">third_party</option>
      </select>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeModal('approve-modal')">キャンセル</button>
      <button class="btn btn-approve" onclick="submitApprove()">承認する</button>
    </div>
  </div>
</div>

<div class="modal-overlay" id="reject-modal">
  <div class="modal">
    <div class="modal-title">申請を却下</div>
    <div class="modal-desc">却下理由を入力してください。申請者に通知されます。</div>
    <div class="form-group"><label>却下メモ</label>
      <textarea id="reject-note" placeholder="例: ホームページ URL が無効です"></textarea>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeModal('reject-modal')">キャンセル</button>
      <button class="btn btn-danger" onclick="submitReject()">却下する</button>
    </div>
  </div>
</div>

<div class="modal-overlay" id="changes-modal">
  <div class="modal">
    <div class="modal-title">修正を依頼</div>
    <div class="modal-desc">修正してほしい点を申請者に伝えてください（必須）。</div>
    <div class="form-group"><label>修正依頼メモ *</label>
      <textarea id="changes-note" placeholder="例: プライバシーポリシーの URL を追加してください"></textarea>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeModal('changes-modal')">キャンセル</button>
      <button class="btn btn-changes" onclick="submitRequestChanges()">送信</button>
    </div>
  </div>
</div>

{{template "token-modal" .}}
{{template "base-js" .}}
<script>
var REQ_ID={{printf "%q" .Request.ID.String}};
async function appReqSend(path, body, modalId, toastMsg, after){
  try{
    var res=await adminFetch('POST','/v1/admin/app-requests/'+REQ_ID+path, body);
    if(!res.ok){var d=await res.json().catch(function(){return {};});throw new Error(d.error||'HTTP '+res.status);}
    showToast(toastMsg);closeModal(modalId);setTimeout(after,700);
  }catch(e){showToast(e.message,'error');}
}
function submitApprove(){appReqSend('/approve',{party_type:document.getElementById('approve-party-type').value},'approve-modal','申請を承認しました',function(){location.href='/admin-ui/app-requests';});}
function submitReject(){appReqSend('/reject',{note:document.getElementById('reject-note').value.trim()},'reject-modal','申請を却下しました',function(){location.href='/admin-ui/app-requests';});}
function submitRequestChanges(){var note=document.getElementById('changes-note').value.trim();if(!note){showToast('修正依頼メモは必須です','error');return;}appReqSend('/request-changes',{note:note},'changes-modal','修正依頼を送信しました',function(){location.reload();});}
</script>
</body></html>
{{end}}
`
