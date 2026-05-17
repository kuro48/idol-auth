package http

import "html/template"

var apiReferenceTpl = template.Must(template.New("api-ref").Parse(apiReferenceHTML))

const apiReferenceHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>idol-auth API Reference</title>
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#0f1117;--surface:#1a1d27;--border:#2a2d3e;--text:#e2e4f0;--muted:#8b8fa8;
  --accent:#6366f1;--accent-dim:#312e81;--green:#22c55e;--blue:#3b82f6;--amber:#f59e0b;--red:#ef4444;
  --post:#16a34a;--get:#1d4ed8;--patch:#b45309;--delete:#b91c1c;
  --radius:8px;--sidebar:260px;--topbar:56px;
}
body{background:var(--bg);color:var(--text);font:14px/1.6 'Inter','Segoe UI',system-ui,sans-serif;display:flex;flex-direction:column;min-height:100vh}

/* ── TOP BAR ── */
#topbar{height:var(--topbar);background:var(--surface);border-bottom:1px solid var(--border);display:flex;align-items:center;gap:16px;padding:0 24px;position:sticky;top:0;z-index:100}
#topbar .logo{font-weight:700;font-size:15px;letter-spacing:-.3px;color:#fff}
#topbar .badge{background:var(--accent-dim);color:#a5b4fc;font-size:10px;font-weight:600;padding:2px 8px;border-radius:4px;text-transform:uppercase;letter-spacing:.5px}
#topbar a.swagger{margin-left:auto;font-size:12px;color:var(--muted);text-decoration:none;display:flex;align-items:center;gap:6px;padding:6px 12px;border:1px solid var(--border);border-radius:6px;transition:border-color .15s,color .15s}
#topbar a.swagger:hover{border-color:var(--accent);color:var(--text)}

/* ── LAYOUT ── */
#layout{display:flex;flex:1}

/* ── SIDEBAR ── */
#sidebar{width:var(--sidebar);flex-shrink:0;background:var(--surface);border-right:1px solid var(--border);position:sticky;top:var(--topbar);height:calc(100vh - var(--topbar));overflow-y:auto;padding:20px 0}
#sidebar::-webkit-scrollbar{width:4px}
#sidebar::-webkit-scrollbar-thumb{background:var(--border);border-radius:2px}
.nav-group{margin-bottom:8px}
.nav-group-label{font-size:10px;font-weight:700;text-transform:uppercase;letter-spacing:.8px;color:var(--muted);padding:8px 20px 4px}
.nav-link{display:flex;align-items:center;gap:8px;padding:6px 20px;text-decoration:none;color:var(--muted);font-size:13px;border-left:2px solid transparent;transition:color .12s,border-color .12s,background .12s}
.nav-link:hover{color:var(--text);background:rgba(99,102,241,.08)}
.nav-link.active{color:var(--accent);border-left-color:var(--accent);background:rgba(99,102,241,.1)}
.nav-link .method-pill{font-size:9px;font-weight:700;padding:1px 5px;border-radius:3px;text-transform:uppercase;letter-spacing:.3px;flex-shrink:0}
.pill-get{background:rgba(29,78,216,.3);color:#93c5fd}
.pill-post{background:rgba(22,163,74,.3);color:#86efac}
.pill-patch{background:rgba(180,83,9,.3);color:#fcd34d}
.pill-delete{background:rgba(185,28,28,.3);color:#fca5a5}

/* ── MAIN ── */
#main{flex:1;padding:40px 48px;max-width:900px}
h1{font-size:28px;font-weight:700;letter-spacing:-.5px;margin-bottom:6px}
.intro-sub{color:var(--muted);font-size:14px;margin-bottom:40px}

h2{font-size:18px;font-weight:700;margin:48px 0 16px;padding-top:8px;border-top:1px solid var(--border)}
h2:first-of-type{margin-top:0;border-top:none}

/* ── ENDPOINT CARD ── */
.endpoint{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);margin-bottom:12px;overflow:hidden}
.endpoint-header{display:flex;align-items:center;gap:12px;padding:14px 18px;cursor:pointer;user-select:none;transition:background .12s}
.endpoint-header:hover{background:rgba(255,255,255,.03)}
.method-tag{font-size:11px;font-weight:700;padding:3px 9px;border-radius:5px;text-transform:uppercase;letter-spacing:.4px;min-width:54px;text-align:center}
.tag-get{background:rgba(29,78,216,.25);color:#93c5fd;border:1px solid rgba(29,78,216,.5)}
.tag-post{background:rgba(22,163,74,.25);color:#86efac;border:1px solid rgba(22,163,74,.5)}
.tag-patch{background:rgba(180,83,9,.25);color:#fcd34d;border:1px solid rgba(180,83,9,.5)}
.tag-delete{background:rgba(185,28,28,.25);color:#fca5a5;border:1px solid rgba(185,28,28,.5)}
.endpoint-path{font-family:ui-monospace,'SFMono-Regular',Consolas,monospace;font-size:13px;color:#e2e4f0}
.endpoint-summary{margin-left:auto;font-size:12px;color:var(--muted)}
.chevron{margin-left:8px;color:var(--muted);transition:transform .2s;flex-shrink:0}
.endpoint-body{display:none;padding:0 18px 18px;border-top:1px solid var(--border)}
.endpoint.open .endpoint-body{display:block}
.endpoint.open .chevron{transform:rotate(180deg)}
.desc{color:var(--muted);font-size:13px;margin:14px 0 12px}

/* ── PARAMS TABLE ── */
.params-label{font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.5px;color:var(--muted);margin:16px 0 8px}
table{width:100%;border-collapse:collapse;font-size:12px}
th{text-align:left;color:var(--muted);font-weight:600;border-bottom:1px solid var(--border);padding:6px 10px}
td{padding:7px 10px;border-bottom:1px solid rgba(42,45,62,.6);vertical-align:top}
tr:last-child td{border-bottom:none}
code.inline{font-family:ui-monospace,'SFMono-Regular',Consolas,monospace;font-size:11px;background:rgba(99,102,241,.12);color:#a5b4fc;padding:1px 5px;border-radius:3px}
.req-badge{font-size:9px;font-weight:700;color:#f87171;background:rgba(248,113,113,.1);padding:1px 5px;border-radius:3px;text-transform:uppercase}
.opt-badge{font-size:9px;color:var(--muted)}

/* ── CODE BLOCK ── */
.code-block{background:#0a0c12;border:1px solid var(--border);border-radius:6px;padding:14px 16px;font-family:ui-monospace,'SFMono-Regular',Consolas,monospace;font-size:12px;line-height:1.7;overflow-x:auto;margin:10px 0}
.code-block .key{color:#7dd3fc}
.code-block .str{color:#86efac}
.code-block .num{color:#fb923c}
.code-block .url{color:#c4b5fd;text-decoration:underline dotted}
.code-block .cmt{color:#4b5563;font-style:italic}

/* ── RESPONSE CHIPS ── */
.responses{display:flex;flex-wrap:wrap;gap:6px;margin:12px 0 4px}
.r-chip{font-size:11px;font-weight:600;padding:3px 9px;border-radius:4px}
.r-2xx{background:rgba(34,197,94,.15);color:#86efac;border:1px solid rgba(34,197,94,.3)}
.r-3xx{background:rgba(59,130,246,.15);color:#93c5fd;border:1px solid rgba(59,130,246,.3)}
.r-4xx{background:rgba(251,191,36,.15);color:#fde68a;border:1px solid rgba(251,191,36,.3)}
.r-5xx{background:rgba(239,68,68,.15);color:#fca5a5;border:1px solid rgba(239,68,68,.3)}

/* ── OVERVIEW BOXES ── */
.info-box{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:18px 20px;margin-bottom:16px}
.info-box h3{font-size:13px;font-weight:700;margin-bottom:10px;color:var(--text)}
.info-box p{font-size:13px;color:var(--muted);line-height:1.7}

/* ── FLOW DIAGRAM ── */
.flow-wrap{overflow-x:auto}
.flow{font-family:ui-monospace,'SFMono-Regular',Consolas,monospace;font-size:11px;color:var(--muted);line-height:1.8;white-space:pre;background:#0a0c12;border:1px solid var(--border);border-radius:6px;padding:16px 20px}
.flow .hi{color:#a5b4fc}
.flow .arrow{color:#4b5563}

/* ── ERROR TABLE ── */
.error-table td:first-child{font-family:ui-monospace,'SFMono-Regular',Consolas,monospace;font-weight:600}

/* ── RESPONSIVE ── */
@media(max-width:768px){
  #sidebar{display:none}
  #main{padding:24px 20px}
}
</style>
</head>
<body>

<header id="topbar">
  <span class="logo">idol-auth</span>
  <span class="badge">API Reference</span>
  <a class="swagger" href="/docs/index.html">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
    Swagger UI
  </a>
</header>

<div id="layout">
<nav id="sidebar">
  <div class="nav-group">
    <div class="nav-group-label">概要</div>
    <a class="nav-link" href="#intro">はじめに</a>
    <a class="nav-link" href="#auth">認証方式</a>
    <a class="nav-link" href="#errors">エラー形式</a>
  </div>
  <div class="nav-group">
    <div class="nav-group-label">ブラウザフロー</div>
    <a class="nav-link" href="#ep-browser-login"><span class="method-pill pill-get">GET</span>/browser/login</a>
    <a class="nav-link" href="#ep-browser-reg"><span class="method-pill pill-get">GET</span>/browser/registration</a>
    <a class="nav-link" href="#ep-browser-logout"><span class="method-pill pill-get">GET</span>/browser/logout</a>
  </div>
  <div class="nav-group">
    <div class="nav-group-label">ヘッドレス API</div>
    <a class="nav-link" href="#ep-register"><span class="method-pill pill-post">POST</span>/api/register</a>
    <a class="nav-link" href="#ep-login"><span class="method-pill pill-post">POST</span>/api/login</a>
    <a class="nav-link" href="#ep-session"><span class="method-pill pill-get">GET</span>/api/session</a>
    <a class="nav-link" href="#ep-token"><span class="method-pill pill-post">POST</span>/api/token</a>
    <a class="nav-link" href="#ep-revoke"><span class="method-pill pill-post">POST</span>/api/token/revoke</a>
    <a class="nav-link" href="#ep-introspect"><span class="method-pill pill-post">POST</span>/api/token/introspect</a>
  </div>
</nav>

<main id="main">

  <h1>Public API Reference</h1>
  <p class="intro-sub">外部アプリ開発者向け。すべての Public エンドポイントのベース URL は <code class="inline">https://auth.example.com</code> です。</p>

  <!-- ═══════════ INTRO ═══════════ -->
  <section id="intro">
    <h2>はじめに</h2>
    <div class="info-box">
      <h3>2 種類の連携モード</h3>
      <p>
        <strong>ブラウザフロー</strong>（リダイレクト型）は、エンドユーザーを idol-auth の認証ページへリダイレクトして OIDC 認証コードを取得します。スマートフォンアプリや SPA でも PKCE を使えば安全に利用できます。<br><br>
        <strong>ヘッドレス API</strong>（直接呼び出し）は、独自 UI を持つアプリが email/password で直接ログイン・登録を行い、<code class="inline">session_token</code> を受け取るモードです。
      </p>
    </div>
    <div class="info-box">
      <h3>PKCE フロー概略</h3>
      <div class="flow-wrap"><pre class="flow"><span class="hi">アプリ</span>                    <span class="hi">idol-auth</span>                   <span class="hi">あなたのサーバー</span>
  <span class="arrow">│</span>                           <span class="arrow">│</span>                            <span class="arrow">│</span>
  <span class="arrow">│</span>─ GET /v1/public/browser/login ──────────────────────<span class="arrow">▶</span><span class="arrow">│</span>
  <span class="arrow">│</span>  (client_id, redirect_uri, code_challenge, ...)      <span class="arrow">│</span>
  <span class="arrow">│</span><span class="arrow">◀</span>── 302 ──────────────────────────────────────────────<span class="arrow">│</span>
  <span class="arrow">│</span>                           <span class="arrow">│</span>
  <span class="arrow">│</span>─ ユーザーがログイン完了 ────<span class="arrow">▶</span><span class="arrow">│</span>
  <span class="arrow">│</span><span class="arrow">◀</span>── redirect_uri?code=AUTH_CODE ─────────────────────<span class="arrow">│</span>
  <span class="arrow">│</span>                                                       <span class="arrow">│</span>
  <span class="arrow">│</span>─ POST /v1/public/api/token (code + code_verifier) ──<span class="arrow">▶</span><span class="arrow">│</span>
  <span class="arrow">│</span><span class="arrow">◀</span>── { access_token, id_token, refresh_token } ────────<span class="arrow">│</span></pre></div>
    </div>
  </section>

  <!-- ═══════════ AUTH ═══════════ -->
  <section id="auth">
    <h2>認証方式</h2>
    <div class="info-box">
      <h3>Bearer トークン</h3>
      <p>ヘッドレス API の保護エンドポイントは <code class="inline">Authorization: Bearer &lt;session_token&gt;</code> ヘッダーが必要です。トークンはログイン・登録の成功レスポンスの <code class="inline">session_token</code> フィールドから取得できます。</p>
    </div>
    <div class="code-block"><span class="cmt"># ヘッドレスログイン後のリクエスト例</span>
<span class="key">Authorization</span>: Bearer <span class="str">ory_st_xxxxxxxxxxxx</span>
<span class="key">Content-Type</span>: application/json</div>
  </section>

  <!-- ═══════════ ERRORS ═══════════ -->
  <section id="errors">
    <h2>エラー形式</h2>
    <div class="info-box">
      <h3>共通エラーレスポンス</h3>
      <p>エラー時は以下の JSON 形式を返します。</p>
    </div>
    <div class="code-block">{
  <span class="key">"error"</span>: <span class="str">"human readable message"</span>
}</div>
    <p class="params-label">HTTP ステータスコード</p>
    <table class="error-table">
      <thead><tr><th>コード</th><th>意味</th></tr></thead>
      <tbody>
        <tr><td>400</td><td>リクエストが不正（必須パラメータ欠落・形式エラーなど）</td></tr>
        <tr><td>401</td><td>認証が必要（セッションなし・トークン期限切れ）</td></tr>
        <tr><td>403</td><td>権限なし</td></tr>
        <tr><td>404</td><td>リソースが見つからない</td></tr>
        <tr><td>422</td><td>バリデーションエラー（登録時の重複など）</td></tr>
        <tr><td>502</td><td>上流サービス（Kratos/Hydra）との通信エラー</td></tr>
      </tbody>
    </table>
  </section>

  <!-- ═══════════ BROWSER FLOW ═══════════ -->
  <h2>ブラウザフロー</h2>

  <div class="endpoint" id="ep-browser-login">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-get">GET</span>
      <span class="endpoint-path">/v1/public/browser/login</span>
      <span class="endpoint-summary">Browser login URL</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">Hydra authorization request 用の browser login URL へリダイレクトします。PKCE フローの開始点です。</p>
      <p class="params-label">Query Parameters</p>
      <table>
        <thead><tr><th>Name</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
        <tbody>
          <tr><td><code class="inline">client_id</code></td><td>string</td><td><span class="req-badge">必須</span></td><td>OIDC クライアント ID</td></tr>
          <tr><td><code class="inline">redirect_uri</code></td><td>string</td><td><span class="req-badge">必須</span></td><td>認証完了後のリダイレクト先 URI</td></tr>
          <tr><td><code class="inline">response_type</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>デフォルト: <code class="inline">code</code></td></tr>
          <tr><td><code class="inline">scope</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>スペース区切りスコープ（例: <code class="inline">openid profile email</code>）</td></tr>
          <tr><td><code class="inline">state</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>CSRF 対策用ランダム文字列</td></tr>
          <tr><td><code class="inline">nonce</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>ID トークンのリプレイ攻撃防止用</td></tr>
          <tr><td><code class="inline">code_challenge</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>PKCE code challenge（推奨）</td></tr>
          <tr><td><code class="inline">code_challenge_method</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td><code class="inline">S256</code>（推奨）</td></tr>
        </tbody>
      </table>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-3xx">302 Found</span>
        <span class="r-chip r-4xx">400 Bad Request</span>
      </div>
    </div>
  </div>

  <div class="endpoint" id="ep-browser-reg">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-get">GET</span>
      <span class="endpoint-path">/v1/public/browser/registration</span>
      <span class="endpoint-summary">Browser registration URL</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">Kratos ブラウザ登録ページへリダイレクトします。</p>
      <p class="params-label">Query Parameters</p>
      <table>
        <thead><tr><th>Name</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
        <tbody>
          <tr><td><code class="inline">return_to</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>登録完了後のリダイレクト先 URL</td></tr>
        </tbody>
      </table>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-3xx">302 Found</span>
      </div>
    </div>
  </div>

  <div class="endpoint" id="ep-browser-logout">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-get">GET</span>
      <span class="endpoint-path">/v1/public/browser/logout</span>
      <span class="endpoint-summary">Browser logout URL</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">OIDC logout URL へリダイレクトします。</p>
      <p class="params-label">Query Parameters</p>
      <table>
        <thead><tr><th>Name</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
        <tbody>
          <tr><td><code class="inline">id_token_hint</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>ID トークン（セッション特定に使用）</td></tr>
          <tr><td><code class="inline">post_logout_redirect_uri</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>ログアウト後のリダイレクト先</td></tr>
          <tr><td><code class="inline">state</code></td><td>string</td><td><span class="opt-badge">任意</span></td><td>OIDC state</td></tr>
        </tbody>
      </table>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-3xx">302 Found</span>
      </div>
    </div>
  </div>

  <!-- ═══════════ HEADLESS API ═══════════ -->
  <h2>ヘッドレス API</h2>

  <div class="endpoint" id="ep-register">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-post">POST</span>
      <span class="endpoint-path">/v1/public/api/register</span>
      <span class="endpoint-summary">Headless registration</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">API モードで新規 identity を作成し、<code class="inline">session_token</code> を返します。独自ログイン UI を持つアプリ向けです。</p>
      <p class="params-label">Request Body <code class="inline">application/json</code></p>
      <div class="code-block">{
  <span class="key">"email"</span>: <span class="str">"user@example.com"</span>,      <span class="cmt">// メールアドレス</span>
  <span class="key">"password"</span>: <span class="str">"securepassword"</span>    <span class="cmt">// 8文字以上</span>
}</div>
      <p class="params-label">Response Body <code class="inline">201 Created</code></p>
      <div class="code-block">{
  <span class="key">"session_token"</span>: <span class="str">"ory_st_xxxx"</span>,
  <span class="key">"identity_id"</span>: <span class="str">"uuid"</span>
}</div>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-2xx">201 Created</span>
        <span class="r-chip r-4xx">400 Bad Request</span>
        <span class="r-chip r-4xx">422 Unprocessable</span>
      </div>
    </div>
  </div>

  <div class="endpoint" id="ep-login">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-post">POST</span>
      <span class="endpoint-path">/v1/public/api/login</span>
      <span class="endpoint-summary">Headless login</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">API モードで email/password 認証を行い、<code class="inline">session_token</code> を返します。</p>
      <p class="params-label">Request Body <code class="inline">application/json</code></p>
      <div class="code-block">{
  <span class="key">"email"</span>: <span class="str">"user@example.com"</span>,
  <span class="key">"password"</span>: <span class="str">"securepassword"</span>
}</div>
      <p class="params-label">Response Body <code class="inline">200 OK</code></p>
      <div class="code-block">{
  <span class="key">"session_token"</span>: <span class="str">"ory_st_xxxx"</span>,
  <span class="key">"identity_id"</span>: <span class="str">"uuid"</span>
}</div>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-2xx">200 OK</span>
        <span class="r-chip r-4xx">400 Bad Request</span>
        <span class="r-chip r-4xx">401 Unauthorized</span>
      </div>
    </div>
  </div>

  <div class="endpoint" id="ep-session">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-get">GET</span>
      <span class="endpoint-path">/v1/public/api/session</span>
      <span class="endpoint-summary">Current headless session</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc"><code class="inline">Authorization: Bearer &lt;session_token&gt;</code> から Kratos セッション情報を返します。</p>
      <p class="params-label">Response Body <code class="inline">200 OK</code></p>
      <div class="code-block">{
  <span class="key">"authenticated"</span>: <span class="num">true</span>,
  <span class="key">"identity_id"</span>: <span class="str">"uuid"</span>,
  <span class="key">"email"</span>: <span class="str">"user@example.com"</span>,
  <span class="key">"display_name"</span>: <span class="str">"username"</span>,
  <span class="key">"roles"</span>: [<span class="str">"member"</span>]
}</div>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-2xx">200 OK</span>
        <span class="r-chip r-4xx">401 Unauthorized</span>
        <span class="r-chip r-5xx">502 Bad Gateway</span>
      </div>
    </div>
  </div>

  <div class="endpoint" id="ep-token">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-post">POST</span>
      <span class="endpoint-path">/v1/public/api/token</span>
      <span class="endpoint-summary">Exchange OAuth2 token</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">Hydra <code class="inline">/oauth2/token</code> への form-encoded プロキシです。認証コード交換・リフレッシュトークン更新に使います。</p>
      <p class="params-label">Request Body <code class="inline">application/x-www-form-urlencoded</code></p>
      <table>
        <thead><tr><th>Field</th><th>Required</th><th>Description</th></tr></thead>
        <tbody>
          <tr><td><code class="inline">grant_type</code></td><td><span class="req-badge">必須</span></td><td><code class="inline">authorization_code</code> または <code class="inline">refresh_token</code></td></tr>
          <tr><td><code class="inline">code</code></td><td><span class="opt-badge">認可コード時</span></td><td>認可エンドポイントから受け取ったコード</td></tr>
          <tr><td><code class="inline">redirect_uri</code></td><td><span class="opt-badge">認可コード時</span></td><td>登録済みリダイレクト URI</td></tr>
          <tr><td><code class="inline">client_id</code></td><td><span class="opt-badge">任意</span></td><td>OIDC クライアント ID</td></tr>
          <tr><td><code class="inline">client_secret</code></td><td><span class="opt-badge">任意</span></td><td>OIDC クライアントシークレット</td></tr>
          <tr><td><code class="inline">code_verifier</code></td><td><span class="opt-badge">PKCE 時</span></td><td>PKCE code verifier</td></tr>
          <tr><td><code class="inline">refresh_token</code></td><td><span class="opt-badge">リフレッシュ時</span></td><td>リフレッシュトークン</td></tr>
        </tbody>
      </table>
      <p class="params-label">Response Body <code class="inline">200 OK</code></p>
      <div class="code-block">{
  <span class="key">"access_token"</span>: <span class="str">"ory_at_xxxx"</span>,
  <span class="key">"id_token"</span>: <span class="str">"eyJ..."</span>,
  <span class="key">"refresh_token"</span>: <span class="str">"ory_rt_xxxx"</span>,
  <span class="key">"token_type"</span>: <span class="str">"bearer"</span>,
  <span class="key">"expires_in"</span>: <span class="num">3600</span>
}</div>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-2xx">200 OK</span>
        <span class="r-chip r-4xx">400 Bad Request</span>
        <span class="r-chip r-5xx">502 Bad Gateway</span>
      </div>
    </div>
  </div>

  <div class="endpoint" id="ep-revoke">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-post">POST</span>
      <span class="endpoint-path">/v1/public/api/token/revoke</span>
      <span class="endpoint-summary">Revoke OAuth2 token</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">アクセストークンまたはリフレッシュトークンを無効化します。Hydra <code class="inline">/oauth2/revoke</code> へのプロキシです。</p>
      <p class="params-label">Request Body <code class="inline">application/x-www-form-urlencoded</code></p>
      <table>
        <thead><tr><th>Field</th><th>Required</th><th>Description</th></tr></thead>
        <tbody>
          <tr><td><code class="inline">token</code></td><td><span class="req-badge">必須</span></td><td>無効化するトークン</td></tr>
          <tr><td><code class="inline">client_id</code></td><td><span class="opt-badge">任意</span></td><td>OIDC クライアント ID</td></tr>
          <tr><td><code class="inline">client_secret</code></td><td><span class="opt-badge">任意</span></td><td>OIDC クライアントシークレット</td></tr>
        </tbody>
      </table>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-2xx">200 OK</span>
        <span class="r-chip r-4xx">400 Bad Request</span>
        <span class="r-chip r-5xx">502 Bad Gateway</span>
      </div>
    </div>
  </div>

  <div class="endpoint" id="ep-introspect">
    <div class="endpoint-header" onclick="toggle(this)">
      <span class="method-tag tag-post">POST</span>
      <span class="endpoint-path">/v1/public/api/token/introspect</span>
      <span class="endpoint-summary">Introspect OAuth2 token</span>
      <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="endpoint-body">
      <p class="desc">トークンの有効性・クレームを検証します。confidential client の credentials が必要です。Hydra <code class="inline">/oauth2/introspect</code> へのプロキシです。</p>
      <p class="params-label">Request Body <code class="inline">application/x-www-form-urlencoded</code></p>
      <table>
        <thead><tr><th>Field</th><th>Required</th><th>Description</th></tr></thead>
        <tbody>
          <tr><td><code class="inline">token</code></td><td><span class="req-badge">必須</span></td><td>検証するトークン</td></tr>
          <tr><td><code class="inline">client_id</code></td><td><span class="opt-badge">任意</span></td><td>OIDC クライアント ID</td></tr>
          <tr><td><code class="inline">client_secret</code></td><td><span class="opt-badge">任意</span></td><td>OIDC クライアントシークレット</td></tr>
        </tbody>
      </table>
      <p class="params-label">Response Body <code class="inline">200 OK</code></p>
      <div class="code-block">{
  <span class="key">"active"</span>: <span class="num">true</span>,
  <span class="key">"sub"</span>: <span class="str">"identity-uuid"</span>,
  <span class="key">"scope"</span>: <span class="str">"openid profile email"</span>,
  <span class="key">"exp"</span>: <span class="num">1716000000</span>,
  <span class="key">"iat"</span>: <span class="num">1715996400</span>,
  <span class="key">"client_id"</span>: <span class="str">"your-client-id"</span>
}</div>
      <p class="params-label">Responses</p>
      <div class="responses">
        <span class="r-chip r-2xx">200 OK</span>
        <span class="r-chip r-4xx">400 Bad Request</span>
        <span class="r-chip r-5xx">502 Bad Gateway</span>
      </div>
    </div>
  </div>

</main>
</div>

<script>
function toggle(header) {
  header.closest('.endpoint').classList.toggle('open');
}

// Scroll-spy
const observer = new IntersectionObserver(entries => {
  entries.forEach(e => {
    if (e.isIntersecting) {
      const id = e.target.id;
      document.querySelectorAll('.nav-link').forEach(l => {
        l.classList.toggle('active', l.getAttribute('href') === '#' + id);
      });
    }
  });
}, { rootMargin: '-30% 0px -60% 0px' });

document.querySelectorAll('section[id], .endpoint[id]').forEach(el => observer.observe(el));
</script>
</body>
</html>
`
