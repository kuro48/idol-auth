package http

// Callers: admin_ui.go — adminUITpl.ExecuteTemplate for each admin UI handler.
// User instruction: 「管理者用のダッシュボードって作れる？」「操作系も最初から含めて」

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
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
}).Parse(adminUITemplates))

const adminUITemplates = `
{{define "head"}}
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --primary:#0017c1;--primary-dark:#00106e;--bg:#f0f2f5;--card:#fff;
  --border:#d4d6db;--text:#1a1a1a;--sub:#595959;
  --ok:#007b50;--ng:#bf0000;--warn:#e85500;--sw:220px
}
body{font-family:"Hiragino Sans","Yu Gothic","Noto Sans JP",system-ui,sans-serif;font-size:14px;background:var(--bg);color:var(--text);min-height:100vh}
a{color:var(--primary);text-decoration:none}a:hover{text-decoration:underline}
.topnav{position:fixed;top:0;left:0;right:0;z-index:100;height:52px;background:var(--primary-dark);color:#fff;display:flex;align-items:center;padding:0 24px;gap:16px}
.topnav-brand{font-size:15px;font-weight:700;letter-spacing:.06em}
.topnav-spacer{flex:1}
.topnav-email{font-size:12px;opacity:.8}
.layout{display:flex;padding-top:52px;min-height:100vh}
.sidebar{position:fixed;top:52px;left:0;bottom:0;width:var(--sw);background:#fff;border-right:1px solid var(--border);padding:12px 0;overflow-y:auto}
.sidebar-section{padding:10px 18px 4px;font-size:11px;font-weight:700;letter-spacing:.1em;text-transform:uppercase;color:#aaa}
.nav-link{display:flex;align-items:center;gap:10px;padding:10px 18px;color:var(--sub);font-size:14px;font-weight:500;transition:background .1s;text-decoration:none}
.nav-link:hover{background:#f0f2f5;color:var(--text);text-decoration:none}
.nav-link.active{background:#e8ecff;color:var(--primary);font-weight:700;text-decoration:none}
.main{margin-left:var(--sw);flex:1;padding:28px 32px}
.page-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:24px}
.page-title{font-size:20px;font-weight:700}
.stats-row{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin-bottom:24px}
.stat-card{background:var(--card);border:1px solid var(--border);border-radius:3px;padding:16px 20px}
.stat-val{font-size:32px;font-weight:700;color:var(--primary);line-height:1}
.stat-label{font-size:12px;color:var(--sub);margin-top:6px}
.card{background:var(--card);border:1px solid var(--border);border-radius:3px;padding:20px 24px;margin-bottom:20px}
.card-title{font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:var(--sub);margin-bottom:14px}
.table-wrap{overflow-x:auto}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;padding:9px 12px;background:#f7f8fb;border-bottom:2px solid var(--border);font-weight:600;color:var(--sub);white-space:nowrap}
td{padding:9px 12px;border-bottom:1px solid #ececec;vertical-align:middle}
tr:hover td{background:#f7f8fb}
.badge{display:inline-flex;align-items:center;padding:2px 8px;border-radius:2px;font-size:11px;font-weight:600}
.badge-active{background:#e6f4ef;color:var(--ok)}
.badge-inactive{background:#fce8e8;color:var(--ng)}
.badge-success{background:#e6f4ef;color:var(--ok)}
.badge-failure{background:#fce8e8;color:var(--ng)}
.badge-rotated{background:#fff3e0;color:var(--warn)}
.btn{display:inline-flex;align-items:center;gap:5px;padding:7px 14px;border-radius:3px;font-size:13px;font-weight:600;cursor:pointer;border:1px solid transparent;transition:opacity .12s;text-decoration:none}
.btn:hover{opacity:.84}
.btn-primary{background:var(--primary);color:#fff;border-color:var(--primary)}
.btn-secondary{background:#fff;color:var(--text);border-color:var(--border)}
.btn-danger{background:var(--ng);color:#fff;border-color:var(--ng)}
.btn-sm{padding:4px 9px;font-size:12px}
.form-row{display:flex;gap:10px;align-items:flex-end;flex-wrap:wrap;margin-bottom:18px}
.form-group{display:flex;flex-direction:column;gap:3px}
.form-group label{font-size:12px;font-weight:600;color:var(--sub)}
input[type=text],input[type=password],select{padding:7px 10px;border:1px solid var(--border);border-radius:3px;font-size:13px;color:var(--text);background:#fff;min-width:160px}
input:focus,select:focus{outline:2px solid var(--primary);outline-offset:-1px}
.modal-overlay{display:none;position:fixed;inset:0;background:rgba(0,0,0,.4);z-index:200;align-items:center;justify-content:center}
.modal-overlay.open{display:flex}
.modal{background:#fff;border-radius:3px;padding:28px;width:520px;max-width:94vw;box-shadow:0 8px 40px rgba(0,0,0,.18)}
.modal-title{font-size:16px;font-weight:700;margin-bottom:10px}
.modal-desc{margin-bottom:16px;color:var(--sub);font-size:13px;line-height:1.6}
.modal-actions{display:flex;gap:10px;justify-content:flex-end;margin-top:20px}
.modal .form-group{margin-bottom:12px}
.modal input[type=text],.modal input[type=password],.modal select{width:100%;min-width:0}
.toast-area{position:fixed;bottom:20px;right:20px;z-index:300;display:flex;flex-direction:column;gap:8px}
.toast{padding:11px 16px;border-radius:3px;font-size:13px;font-weight:600;color:#fff;box-shadow:0 4px 16px rgba(0,0,0,.2);animation:fi .18s ease}
.toast-success{background:var(--ok)}.toast-error{background:var(--ng)}
@keyframes fi{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:none}}
.empty-row td{text-align:center;color:#aaa;padding:24px}
.actions-cell{display:flex;gap:6px;flex-wrap:wrap}
</style>
{{end}}

{{define "topnav"}}
<header class="topnav">
  <span class="topnav-brand">idol-auth</span>
  <span style="font-size:11px;opacity:.45;letter-spacing:.08em">ADMIN</span>
  <span class="topnav-spacer"></span>
  <span class="topnav-email">{{.Email}}</span>
</header>
{{end}}

{{define "sidebar"}}
<nav class="sidebar">
  <div class="sidebar-section">メニュー</div>
  <a class="nav-link{{if eq .Nav "overview"}} active{{end}}" href="/admin-ui/">概要</a>
  <a class="nav-link{{if eq .Nav "apps"}} active{{end}}" href="/admin-ui/apps">アプリ</a>
  <a class="nav-link{{if eq .Nav "users"}} active{{end}}" href="/admin-ui/users">ユーザー</a>
  <a class="nav-link{{if eq .Nav "audit-logs"}} active{{end}}" href="/admin-ui/audit-logs">監査ログ</a>
</nav>
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
var _tokenCb=null;
function getToken(){return sessionStorage.getItem(TOKEN_KEY);}
function clearToken(){sessionStorage.removeItem(TOKEN_KEY);}
function closeModal(id){document.getElementById(id).classList.remove('open');}
function openModal(id){document.getElementById(id).classList.add('open');}
function confirmToken(){
  var t=document.getElementById('token-input').value.trim();
  if(!t)return;
  sessionStorage.setItem(TOKEN_KEY,t);
  closeModal('token-modal');
  if(_tokenCb){var cb=_tokenCb;_tokenCb=null;cb(t);}
}
function showToast(msg,type){
  var area=document.getElementById('toast-area');
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
</script>
{{end}}

{{define "overview"}}<!DOCTYPE html>
<html lang="ja">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>概要 — 管理ダッシュボード</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header"><h1 class="page-title">概要</h1></div>
  <div class="stats-row">
    <div class="stat-card">
      <div class="stat-val">{{.AppCount}}</div>
      <div class="stat-label">登録アプリ数</div>
    </div>
    <div class="stat-card">
      <div class="stat-val">{{len .RecentLogs}}</div>
      <div class="stat-label">直近イベント数（最大20件）</div>
    </div>
    <div class="stat-card">
      <div class="stat-val" style="font-size:18px">{{firstLogTime .RecentLogs}}</div>
      <div class="stat-label">最終監査イベント</div>
    </div>
  </div>
  <div class="card">
    <div class="card-title">最近の監査ログ</div>
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
<html lang="ja">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>アプリ — 管理ダッシュボード</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header">
    <h1 class="page-title">アプリ</h1>
    <button class="btn btn-primary" onclick="openModal('app-modal')">+ アプリ作成</button>
  </div>
  <div class="card">
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
<html lang="ja">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>ユーザー — 管理ダッシュボード</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header"><h1 class="page-title">ユーザー</h1></div>
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
  <div class="card">
    <div class="table-wrap">
      <table>
        <thead><tr><th>メール / Phone</th><th>ID</th><th>ステータス</th><th>ロール</th><th>操作</th></tr></thead>
        <tbody>
          {{range .Users}}
          <tr>
            <td>{{if .Email}}{{.Email}}{{else}}{{.Phone}}{{end}}</td>
            <td><code style="font-size:11px">{{truncate .ID 18}}</code></td>
            <td><span class="badge {{badgeClass .State}}">{{.State}}</span></td>
            <td>{{if .Roles}}<span style="font-size:12px;color:var(--sub)">{{join .Roles ", "}}</span>{{else}}<span style="color:#ccc">—</span>{{end}}</td>
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
<html lang="ja">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>監査ログ — 管理ダッシュボード</title>{{template "head" .}}</head>
<body>
{{template "topnav" .}}
<div class="layout">
{{template "sidebar" .}}
<main class="main">
  <div class="page-header"><h1 class="page-title">監査ログ</h1></div>
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
  <div class="card">
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
            <td style="font-size:11px;color:var(--sub)">{{.IPAddress}}</td>
          </tr>
          {{else}}<tr class="empty-row"><td colspan="6">ログはありません</td></tr>{{end}}
        </tbody>
      </table>
    </div>
    {{if .HasMore}}<p style="margin-top:12px;font-size:12px;color:var(--sub)">表示上限（50件）に達しました。絞り込みで件数を減らしてください。</p>{{end}}
  </div>
</main>
</div>
{{template "token-modal" .}}
{{template "base-js" .}}
</body></html>
{{end}}

{{define "error"}}<!DOCTYPE html>
<html lang="ja">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>エラー — 管理ダッシュボード</title>{{template "head" .}}</head>
<body style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--bg)">
  <div style="background:#fff;border:1px solid var(--border);border-radius:3px;padding:40px 48px;max-width:440px;text-align:center">
    <div style="font-size:18px;font-weight:700;margin-bottom:10px">{{.Title}}</div>
    <p style="color:var(--sub);font-size:14px;line-height:1.6">{{.Msg}}</p>
  </div>
</body></html>
{{end}}
`
