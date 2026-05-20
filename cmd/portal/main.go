package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kuro48/idol-auth/internal/demo"
	kratosinfra "github.com/kuro48/idol-auth/internal/infra/kratos"
)

func main() {
	if err := run(); err != nil {
		slog.Error("portal server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := demo.LoadPortalConfig()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	kratosClient := demo.NewKratosFlowClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)
	sessionClient := kratosinfra.NewFrontendClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)
	themeUpdater := kratosinfra.NewAdminClient(cfg.KratosAdminURL)

	kratosPublicURL, err := url.Parse(cfg.KratosPublicURL)
	if err != nil {
		return fmt.Errorf("parse kratos public url: %w", err)
	}
	kratosProxy := httputil.NewSingleHostReverseProxy(kratosPublicURL)

	mux := http.NewServeMux()
	mux.Handle("/self-service/", kratosProxy)
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handleLogout(w, r, cfg, http.DefaultClient)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderHome(w, demo.ResolveSessionOshiColor(r.Context(), sessionClient, r))
	})
	mux.HandleFunc("/ui/theme", func(w http.ResponseWriter, r *http.Request) {
		demo.HandleThemePreference(w, r, sessionClient, themeUpdater)
	})

	registerFlow(mux, kratosClient, sessionClient, "login", "Login", "Sign in with the shared account.")
	registerFlow(mux, kratosClient, sessionClient, "registration", "Registration", "Create a shared account.")
	registerFlow(mux, kratosClient, sessionClient, "recovery", "Recovery", "Recover your account.")
	registerFlow(mux, kratosClient, sessionClient, "verification", "Verification", "Verify your identifier.")
	registerFlow(mux, kratosClient, sessionClient, "settings", "Settings", "Manage security settings and MFA.")
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		demo.HandleKratosError(w, r, kratosClient)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("portal server starting", "addr", server.Addr)
	return server.ListenAndServe()
}

func registerFlow(mux *http.ServeMux, kratosClient *demo.KratosFlowClient, sessionClient demo.SessionReader, flowType, title, description string) {
	mux.HandleFunc("/"+flowType, func(w http.ResponseWriter, r *http.Request) {
		flowID := r.URL.Query().Get("flow")
		if flowID == "" {
			http.Redirect(w, r, kratosClient.BrowserInitURL(flowType), http.StatusFound)
			return
		}
		flow, err := kratosClient.GetFlow(r.Context(), r, flowType, flowID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err := demo.RenderPage(w, demo.PageData{
			Title:       title,
			Description: description,
			FlowType:    flowType,
			OshiColor:   demo.ResolveSessionOshiColor(r.Context(), sessionClient, r),
			Flow:        flow,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request, cfg *demo.PortalConfig, client *http.Client) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	loginURL := strings.TrimRight(cfg.AppURL, "/") + "/login"
	logoutBrowserURL, err := url.Parse(strings.TrimRight(cfg.KratosPublicURL, "/") + "/self-service/logout/browser")
	if err != nil {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	q := logoutBrowserURL.Query()
	q.Set("return_to", loginURL)
	logoutBrowserURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, logoutBrowserURL.String(), nil)
	if err != nil {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	if cookie := filterOryCookies(r.Header.Get("Cookie")); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var decoded struct {
		LogoutURL string `json:"logout_url"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil || strings.TrimSpace(decoded.LogoutURL) == "" {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, decoded.LogoutURL, http.StatusSeeOther)
}

func filterOryCookies(cookieHeader string) string {
	parts := strings.Split(cookieHeader, ";")
	ory := parts[:0]
	for _, part := range parts {
		if strings.HasPrefix(strings.TrimSpace(part), "ory_") {
			ory = append(ory, strings.TrimSpace(part))
		}
	}
	return strings.Join(ory, "; ")
}

func renderHome(w http.ResponseWriter, oshiColor string) {
	const tpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Idol Auth Portal</title>
  <style>
    :root {
      --oshi:#ff8a3d;
      --oshi-weak:#fff1e8;
      --oshi-soft:#ffd9c2;
      --bg:#f8f6f3;
      --surface:#ffffff;
      --surface-2:#fffaf6;
      --text:#26211f;
      --muted:#776d67;
      --border:#eadfd7;
      --shadow:none;
      --radius-lg:8px;
      --radius-md:6px;
      --radius-sm:4px;
      --oshi-deep:color-mix(in srgb,var(--oshi) 72%,#26211f);
      --oshi-line:color-mix(in srgb,var(--oshi) 34%,transparent);
    }
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
      padding: 0 0 96px;
      position: relative;
      overflow-x: hidden;
    }
    .topbar {
      position: sticky;
      top: 0;
      z-index: 20;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      padding: 14px 28px;
      background: var(--surface);
      border-bottom: 1px solid var(--border);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .brand-text strong {
      display: block;
      font-size: 16px;
      line-height: 1.1;
      letter-spacing: -0.01em;
    }
    .brand-text span {
      font-size: 11px;
      color: var(--muted);
      font-weight: 800;
      text-transform: uppercase;
      letter-spacing: 0.12em;
    }
    .top-actions {
      display: flex;
      align-items: center;
      gap: 10px;
      flex-wrap: wrap;
      justify-content: flex-end;
    }
    .top-pill {
      padding: 8px 13px;
      border-radius: var(--radius-md);
      background: var(--surface-2);
      border: 1px solid var(--border);
      color: var(--muted);
      font-size: 12px;
      font-weight: 800;
    }
    .shell { max-width: 1180px; margin: 0 auto; padding: 28px 28px 0; position: relative; z-index: 1; }
    .container { max-width: none; margin: 0; position: relative; z-index: 1; }
    .hero {
      position: relative;
      overflow: hidden;
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      padding: 32px;
      margin-bottom: 20px;
    }
    .profile-hero {
      padding: 30px;
      background: var(--surface);
    }
    .tag {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      background: var(--surface-2);
      border: 1px solid var(--oshi-line);
      border-radius: var(--radius-md);
      padding: 7px 14px;
      font-size: 11px;
      font-weight: 700;
      color: var(--oshi-deep);
      letter-spacing: 0.1em;
      text-transform: uppercase;
      margin-bottom: 18px;
      position: relative;
      z-index: 1;
    }
    .brand-mark {
      width: 42px;
      height: 42px;
      border-radius: var(--radius-md);
      background: var(--oshi);
      color: #fff;
      display: inline-grid;
      place-items: center;
      font-size: 20px;
      font-weight: 900;
    }
    .badge-oshi {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      margin-top: 16px;
      padding: 8px 13px;
      border-radius: var(--radius-md);
      background: var(--oshi-weak);
      color: var(--oshi);
      font-weight: 900;
      font-size: 12px;
    }
    .badge-oshi .swatch-dot {
      width: 14px;
      height: 14px;
      border-radius: 50%;
      background: var(--oshi);
      border: 1px solid var(--border);
    }
    .hero-grid {
      display: grid;
      grid-template-columns: minmax(0, 1.25fr) minmax(320px, 0.9fr);
      gap: 24px;
      position: relative;
      z-index: 1;
    }
    h1 {
      margin: 0 0 14px;
      font-family: "Avenir Next Condensed", "Avenir Next", "Yu Gothic", sans-serif;
      font-size: clamp(38px, 6vw, 72px);
      line-height: 0.96;
      letter-spacing: -0.06em;
    }
    .accent { color: var(--oshi-deep); }
    .subtitle {
      color: var(--muted);
      font-size: 16px;
      line-height: 1.8;
      margin: 0 0 28px;
      max-width: 44rem;
    }
    .microcopy {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .microcopy li {
      padding: 8px 12px;
      border-radius: var(--radius-md);
      background: var(--surface-2);
      border: 1px solid var(--border);
      font-size: 12px;
      color: var(--muted);
    }
    .feature-panel {
      background: var(--surface-2);
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      padding: 22px;
    }
    .feature-panel h2 { margin: 0 0 10px; font-size: 20px; letter-spacing: -0.03em; }
    .feature-panel p { margin: 0 0 18px; color: var(--muted); line-height: 1.75; font-size: 14px; }
    .feature-stack { display: grid; gap: 12px; }
    .feature-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-md);
      padding: 14px 16px;
    }
    .feature-card strong { display: block; margin-bottom: 5px; color: #252849; font-size: 14px; }
    .feature-card span { color: var(--muted); font-size: 13px; line-height: 1.6; }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 14px;
    }
    .service-list { margin-top: 20px; }
    .card {
      grid-column: span 4;
      position: relative;
      overflow: hidden;
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      padding: 24px;
      text-decoration: none;
      color: var(--text);
      display: flex;
      flex-direction: column;
      gap: 10px;
      transition: background 0.18s ease, border-color 0.18s ease;
      min-height: 180px;
    }
    .card:hover {
      border-color: var(--oshi-line);
      background: var(--surface-2);
    }
    .card-wide { grid-column: span 6; }
    .card-tall { min-height: 210px; }
    .card-icon {
      width: 52px;
      height: 52px;
      border-radius: var(--radius-md);
      display: inline-flex;
      align-items: center;
      justify-content: center;
      background: var(--surface-2);
      border: 1px solid var(--oshi-line);
      font-size: 20px;
      color: var(--oshi-deep);
      margin-bottom: 8px;
      position: relative;
      z-index: 1;
    }
    .card-kicker {
      display: inline-flex;
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: var(--oshi-deep);
      position: relative;
      z-index: 1;
    }
    .card-title {
      font-size: 22px;
      font-weight: 800;
      letter-spacing: -0.04em;
      position: relative;
      z-index: 1;
    }
    .card-desc {
      font-size: 14px;
      color: var(--muted);
      line-height: 1.7;
      position: relative;
      z-index: 1;
    }
    .card-primary {
      background: var(--oshi-weak);
      border-color: var(--oshi-line);
    }
    .card-meta { margin-top: auto; font-size: 12px; color: #5f6484; position: relative; z-index: 1; }
    .note {
      margin-top: 18px;
      padding: 18px 20px;
      border-radius: var(--radius-lg);
      background: var(--surface);
      border: 1px solid var(--border);
      color: var(--muted);
      font-size: 13px;
      line-height: 1.8;
    }
    #oshi-picker { position: fixed; right: 22px; bottom: 22px; z-index: 20; }
    #oshi-toggle {
      width: 58px;
      height: 58px;
      border-radius: 50%;
      border: 1px solid var(--border);
      background: var(--surface);
      color: var(--oshi-deep);
      font-size: 24px;
      cursor: pointer;
    }
    #oshi-swatches {
      display: none;
      grid-template-columns: repeat(4, 1fr);
      gap: 10px;
      width: 188px;
      margin-bottom: 12px;
      padding: 14px;
      border-radius: var(--radius-lg);
      background: var(--surface);
      border: 1px solid var(--border);
    }
    .swatch {
      width: 100%;
      aspect-ratio: 1;
      border-radius: 999px;
      border: 2px solid transparent;
      cursor: pointer;
      transition: transform 0.12s ease, border-color 0.12s ease;
    }
    .swatch:hover { border-color: var(--text); }
    .swatch.active { border-color: #1d2040; }
    @media (max-width: 920px) {
      .hero-grid { grid-template-columns: 1fr; }
      .card { grid-column: span 6; }
      .card-wide { grid-column: span 12; }
    }
    @media (max-width: 640px) {
      body { padding-bottom: 92px; }
      .topbar { padding: 12px 18px; }
      .brand-text span { display: none; }
      .top-pill { display: none; }
      .shell { padding: 20px 18px 0; }
      .hero { padding: 24px; }
      .grid { grid-template-columns: 1fr; }
      .card, .card-wide { grid-column: span 1; min-height: 0; }
      h1 { font-size: 42px; }
      #oshi-toggle { width: 52px; height: 52px; }
      #oshi-swatches { width: 168px; }
    }
  </style>
  <script>
    var OSHI=['#ffb2b2','#ffb2d8','#ffb2ff','#d8b2ff','#b2b2ff','#b2d8ff','#b2ffff','#b2ffd8','#b2ffb2','#d8ffb2','#ffffb2','#ffd8b2'];
    function normalizeOshi(raw){
      raw=(raw||'').trim().toLowerCase();
      return OSHI.indexOf(raw)>=0?raw:'';
    }
    function oshiRgb(hex){return[parseInt(hex.slice(1,3),16),parseInt(hex.slice(3,5),16),parseInt(hex.slice(5,7),16)];}
    function oshiHex(r,g,b){return'#'+[r,g,b].map(function(v){return Math.min(255,Math.max(0,v)).toString(16).padStart(2,'0');}).join('');}
    function applyOshi(color){
      var c=oshiRgb(color), root=document.documentElement;
      root.style.setProperty('--oshi', color);
      root.style.setProperty('--oshi-deep', oshiHex(c[0]-90, c[1]-90, c[2]-40));
      root.style.setProperty('--oshi-soft', 'rgba('+c[0]+','+c[1]+','+c[2]+',0.18)');
      root.style.setProperty('--oshi-line', 'rgba('+c[0]+','+c[1]+','+c[2]+',0.44)');
    }
    function persistOshi(color){
      fetch('/ui/theme',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        credentials:'same-origin',
        body:JSON.stringify({oshi_color:color})
      }).catch(function(){});
    }
    var _oshi=normalizeOshi({{ printf "%q" .OshiColor }})||OSHI[4];
    applyOshi(_oshi);
  </script>
</head>
<body>
  <header class="topbar">
    <div class="brand">
      <div class="brand-mark">推</div>
      <div class="brand-text">
        <strong>Idol Auth</strong>
        <span>Account Portal</span>
      </div>
    </div>
    <div class="top-actions">
      <span class="top-pill">Kratos self-service</span>
      <a class="top-pill" href="/logout">ログアウト</a>
    </div>
  </header>
  <main class="shell">
  <div class="container">
    <section class="profile-hero hero">
      <div class="tag">Account Portal</div>
      <div class="hero-grid">
        <div>
          <h1>推しメンカラーで<br><span class="accent">認証を彩る。</span></h1>
          <p class="subtitle">ログイン、新規登録、MFA 設定、アカウント復旧——すべての認証体験をひとつのポータルに。右下のパレットから色を選ぶと、各フロー画面の雰囲気がそのまま切り替わります。</p>
          <span class="badge-oshi"><span class="swatch-dot"></span>推し色設定を各フローへ反映</span>
          <ul class="microcopy">
            <li>12 色の推しメンカラー対応</li>
            <li>Kratos self-service flow を完全サポート</li>
            <li>モバイルでも崩れない 1 カラム対応</li>
          </ul>
        </div>
        <aside class="feature-panel">
          <h2>できること</h2>
          <p>認証まわりの操作をこのポータルからまとめて行えます。</p>
          <div class="feature-stack">
            <div class="feature-card">
              <strong>ログイン / 新規登録</strong>
              <span>既存アカウントへのサインインと、新規 shared account の作成。</span>
            </div>
            <div class="feature-card">
              <strong>セキュリティ設定</strong>
              <span>TOTP、パスワード変更など MFA まわりの設定を一箇所で。</span>
            </div>
            <div class="feature-card">
              <strong>復旧 / 確認</strong>
              <span>パスワード再設定と識別子の確認フローをサポート。</span>
            </div>
          </div>
        </aside>
      </div>
    </section>
    <div class="service-list">
      <div class="grid">
        <a class="card card-primary card-wide card-tall" href="/login">
          <div class="card-icon">◉</div>
          <div class="card-kicker">Entry</div>
          <div class="card-title">ログイン</div>
          <div class="card-desc">既存の shared account でサインインします。セッション状態に応じたログイン UI が開きます。</div>
          <div class="card-meta">Kratos self-service login flow</div>
        </a>
        <a class="card card-wide" href="/registration">
          <div class="card-icon">+</div>
          <div class="card-kicker">Onboarding</div>
          <div class="card-title">アカウントを作成</div>
          <div class="card-desc">新しい shared account を登録します。Kratos の新規登録フローが開きます。</div>
          <div class="card-meta">Kratos self-service registration flow</div>
        </a>
        <a class="card" href="/settings">
          <div class="card-icon">◈</div>
          <div class="card-kicker">Security</div>
          <div class="card-title">セキュリティ設定</div>
          <div class="card-desc">MFA、パスワード変更などのセキュリティ設定を管理します。</div>
        </a>
        <a class="card" href="/recovery">
          <div class="card-icon">↺</div>
          <div class="card-kicker">Recovery</div>
          <div class="card-title">アカウント復旧</div>
          <div class="card-desc">パスワードを忘れた場合など、アカウント復旧フローを開始します。</div>
        </a>
        <a class="card" href="/verification">
          <div class="card-icon">✓</div>
          <div class="card-kicker">Verify</div>
          <div class="card-title">確認フロー</div>
          <div class="card-desc">メールアドレスや識別子の確認と検証を行います。</div>
        </a>
      </div>
    </div>
    <div class="note">推しメンカラーは右下の <strong>✦</strong> から切り替えられます。ログイン中は選択した色がアカウントに保存され、次回アクセス時も自動で引き継がれます。</div>
  </div>
  </main>
  <div id="oshi-picker" aria-label="推し色を選ぶ">
    <div id="oshi-swatches" aria-label="推し色を選ぶ"></div>
    <button id="oshi-toggle" type="button" title="推しメンカラー">✦</button>
  </div>
  <script>
    (function(){
      var sw=document.getElementById('oshi-swatches');
      var toggle=document.getElementById('oshi-toggle');
      var current=normalizeOshi({{ printf "%q" .OshiColor }})||OSHI[4];
      OSHI.forEach(function(color){
        var btn=document.createElement('button');
        btn.type='button';
        btn.className='swatch'+(color===current?' active':'');
        btn.style.background=color;
        btn.title='推しメンカラー '+(OSHI.indexOf(color)+1);
        btn.addEventListener('click', function(){
          applyOshi(color);
          persistOshi(color);
          document.querySelectorAll('.swatch').forEach(function(node){
            node.classList.toggle('active', node===btn);
          });
        });
        sw.appendChild(btn);
      });
      toggle.addEventListener('click', function(){
        sw.style.display = sw.style.display === 'grid' ? 'none' : 'grid';
      });
    })();
  </script>
</body>
</html>`
	t := template.Must(template.New("home").Parse(tpl))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.Execute(w, struct {
		OshiColor string
	}{
		OshiColor: oshiColor,
	})
}
