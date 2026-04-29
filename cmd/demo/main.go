package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ryunosukekurokawa/idol-auth/internal/demo"
	kratosinfra "github.com/ryunosukekurokawa/idol-auth/internal/infra/kratos"
)

const (
	stateCookieName    = "idol_demo_state"
	verifierCookieName = "idol_demo_verifier"
	clientKindCookie   = "idol_demo_client_kind"
)

func main() {
	if err := run(); err != nil {
		slog.Error("demo server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := demo.LoadConfig()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	controlPlane := demo.NewControlPlaneClient(cfg.AuthInternalURL, cfg.AdminToken)
	kratosClient := demo.NewKratosFlowClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)
	sessionClient := kratosinfra.NewFrontendClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)
	themeUpdater := kratosinfra.NewAdminClient(cfg.KratosAdminURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderHome(w, cfg, demo.ResolveSessionOshiColor(r.Context(), sessionClient, r))
	})
	mux.HandleFunc("/ui/theme", func(w http.ResponseWriter, r *http.Request) {
		demo.HandleThemePreference(w, r, sessionClient, themeUpdater)
	})
	mux.HandleFunc("/oauth/start", func(w http.ResponseWriter, r *http.Request) {
		spec := selectedAppSpec(cfg, r.URL.Query().Get("app"))
		clientID, err := controlPlane.EnsureDemoClient(r.Context(), cfg, spec)
		if err != nil {
			http.Error(w, "デモクライアントの初期化に失敗しました: "+err.Error(), http.StatusBadGateway)
			return
		}
		verifier, err := demo.GeneratePKCEVerifier()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		state, err := demo.GenerateState()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		secure := r.TLS != nil
		http.SetCookie(w, &http.Cookie{Name: verifierCookieName, Value: verifier, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
		http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: state, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
		http.SetCookie(w, &http.Cookie{Name: clientKindCookie, Value: spec.Key, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})

		redirectURI := strings.TrimRight(cfg.AppURL, "/") + "/oauth/callback"
		authURL, err := demo.BuildAuthorizationURL(demo.AuthorizationParams{
			HydraBrowserURL: cfg.HydraBrowserURL,
			ClientID:        clientID,
			RedirectURI:     redirectURI,
			State:           state,
			CodeChallenge:   demo.ComputeS256Challenge(verifier),
			Scopes:          []string{"openid", "offline_access"},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
	})
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		if errValue := r.URL.Query().Get("error"); errValue != "" {
			http.Error(w, "OAuth エラー: "+errValue, http.StatusBadRequest)
			return
		}
		stateCookie, err := r.Cookie(stateCookieName)
		if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "OAuth の state が不正です", http.StatusBadRequest)
			return
		}
		verifierCookie, err := r.Cookie(verifierCookieName)
		if err != nil || verifierCookie.Value == "" {
			http.Error(w, "PKCE verifier がありません", http.StatusBadRequest)
			return
		}
		clientKind := ""
		if clientKindCookieValue, err := r.Cookie(clientKindCookie); err == nil {
			clientKind = clientKindCookieValue.Value
		}

		clientID, err := controlPlane.EnsureDemoClient(r.Context(), cfg, selectedAppSpec(cfg, clientKind))
		if err != nil {
			http.Error(w, "デモクライアントの初期化に失敗しました: "+err.Error(), http.StatusBadGateway)
			return
		}

		tokenResp, err := exchangeCode(r.Context(), cfg, clientID, verifierCookie.Value, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: "", Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{Name: verifierCookieName, Value: "", Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{Name: clientKindCookie, Value: "", Path: "/", MaxAge: -1})
		renderToken(w, tokenResp, demo.ResolveSessionOshiColor(r.Context(), sessionClient, r))
	})

	registerFlow(mux, kratosClient, sessionClient, "login", "ログイン", "共有アカウントでサインインします。")
	registerFlow(mux, kratosClient, sessionClient, "registration", "新規登録", "共有アカウントを作成します。")
	registerFlow(mux, kratosClient, sessionClient, "recovery", "アカウント復旧", "アカウント復旧フローを開始します。")
	registerFlow(mux, kratosClient, sessionClient, "verification", "確認", "メールアドレスや識別子を確認します。")
	registerFlow(mux, kratosClient, sessionClient, "settings", "セキュリティ設定", "セキュリティ設定や MFA を管理します。")

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("demo server starting", "addr", server.Addr)
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

func renderHome(w http.ResponseWriter, cfg *demo.Config, oshiColor string) {
	const tpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Idol Auth — デモ</title>
  <style>
    :root {
      --oshi: #b2b2ff;
      --oshi-deep: #4c4cc6;
      --oshi-soft: rgba(178,178,255,0.18);
      --oshi-line: rgba(178,178,255,0.44);
      --surface: rgba(255,255,255,0.74);
      --text: #1d2040;
      --muted: #6f7394;
      --shadow: 0 24px 80px rgba(72,54,120,0.14);
    }
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background:
        radial-gradient(circle at 12% 18%, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0) 28%),
        radial-gradient(circle at 82% 14%, var(--oshi-soft) 0%, rgba(255,255,255,0) 34%),
        radial-gradient(circle at 80% 84%, rgba(216,178,255,0.22) 0%, rgba(255,255,255,0) 30%),
        linear-gradient(160deg, #fff8fb 0%, #f4f6ff 46%, #edfaff 100%);
      color: var(--text);
      font-family: "Avenir Next", "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
      padding: 40px 20px 72px;
      position: relative;
      overflow-x: hidden;
    }
    body::before, body::after {
      content: "";
      position: fixed;
      border-radius: 999px;
      filter: blur(14px);
      opacity: 0.45;
      pointer-events: none;
    }
    body::before {
      width: 220px;
      height: 220px;
      top: 10%;
      left: -60px;
      background: var(--oshi);
    }
    body::after {
      width: 280px;
      height: 280px;
      right: -90px;
      bottom: -40px;
      background: rgba(178,255,255,0.72);
    }
    .container { max-width: 1080px; margin: 0 auto; position: relative; z-index: 1; }
    .hero {
      position: relative;
      overflow: hidden;
      background: var(--surface);
      border: 1px solid rgba(255,255,255,0.56);
      border-radius: 36px;
      padding: 32px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(24px);
      margin-bottom: 20px;
    }
    .hero::before {
      content: "";
      position: absolute;
      inset: -25% auto auto -10%;
      width: 320px;
      height: 320px;
      background: radial-gradient(circle, var(--oshi-soft) 0%, rgba(255,255,255,0) 70%);
      pointer-events: none;
    }
    .tag {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      background: rgba(255,255,255,0.62);
      border: 1px solid var(--oshi-line);
      border-radius: 100px;
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
    .accent { color: var(--oshi-deep); text-shadow: 0 0 30px rgba(255,255,255,0.45); }
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
      border-radius: 999px;
      background: rgba(255,255,255,0.7);
      border: 1px solid rgba(29,32,64,0.08);
      font-size: 12px;
      color: #5f6484;
    }
    .feature-panel {
      background: linear-gradient(180deg, rgba(255,255,255,0.9), rgba(255,255,255,0.7));
      border: 1px solid rgba(255,255,255,0.84);
      border-radius: 28px;
      padding: 22px;
      box-shadow: inset 0 1px 0 rgba(255,255,255,0.75);
    }
    .feature-panel h2 { margin: 0 0 10px; font-size: 20px; letter-spacing: -0.03em; }
    .feature-panel p { margin: 0 0 18px; color: var(--muted); line-height: 1.75; font-size: 14px; }
    .feature-stack { display: grid; gap: 12px; }
    .feature-card {
      background: rgba(255,255,255,0.66);
      border: 1px solid rgba(29,32,64,0.08);
      border-radius: 22px;
      padding: 14px 16px;
    }
    .feature-card strong { display: block; margin-bottom: 5px; color: #252849; font-size: 14px; }
    .feature-card span { color: var(--muted); font-size: 13px; line-height: 1.6; }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 14px;
    }
    .card {
      grid-column: span 4;
      position: relative;
      overflow: hidden;
      background: var(--surface);
      border: 1px solid rgba(255,255,255,0.68);
      border-radius: 28px;
      padding: 24px;
      text-decoration: none;
      color: var(--text);
      display: flex;
      flex-direction: column;
      gap: 10px;
      transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
      box-shadow: 0 18px 50px rgba(58,61,109,0.1);
      min-height: 180px;
      backdrop-filter: blur(24px);
    }
    .card::before {
      content: "";
      position: absolute;
      inset: 0;
      background: linear-gradient(135deg, rgba(255,255,255,0.7), rgba(255,255,255,0));
      pointer-events: none;
    }
    .card:hover {
      transform: translateY(-4px);
      border-color: var(--oshi-line);
      box-shadow: 0 24px 64px rgba(58,61,109,0.14);
    }
    .card-wide { grid-column: span 6; }
    .card-tall { min-height: 210px; }
    .card-icon {
      width: 52px;
      height: 52px;
      border-radius: 18px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(180deg, rgba(255,255,255,0.96), var(--oshi-soft));
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
      background: linear-gradient(160deg, rgba(255,255,255,0.92), var(--oshi-soft));
      border-color: var(--oshi-line);
    }
    .card-meta { margin-top: auto; font-size: 12px; color: #5f6484; position: relative; z-index: 1; }
    .note {
      margin-top: 18px;
      padding: 18px 20px;
      border-radius: 22px;
      background: rgba(255,255,255,0.62);
      border: 1px solid rgba(255,255,255,0.8);
      color: var(--muted);
      font-size: 13px;
      line-height: 1.8;
      box-shadow: 0 12px 34px rgba(58,61,109,0.08);
    }
    #oshi-picker { position: fixed; right: 18px; bottom: 18px; z-index: 20; }
    #oshi-toggle {
      width: 58px;
      height: 58px;
      border-radius: 20px;
      border: 1px solid rgba(255,255,255,0.84);
      background: linear-gradient(180deg, rgba(255,255,255,0.96), var(--oshi-soft));
      color: var(--oshi-deep);
      font-size: 24px;
      cursor: pointer;
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      backdrop-filter: blur(24px);
    }
    #oshi-swatches {
      display: none;
      grid-template-columns: repeat(4, 1fr);
      gap: 10px;
      width: 188px;
      margin-bottom: 12px;
      padding: 14px;
      border-radius: 22px;
      background: rgba(255,255,255,0.86);
      border: 1px solid rgba(255,255,255,0.84);
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      backdrop-filter: blur(24px);
    }
    .swatch {
      width: 100%;
      aspect-ratio: 1;
      border-radius: 999px;
      border: 2px solid transparent;
      cursor: pointer;
      transition: transform 0.12s ease, border-color 0.12s ease;
    }
    .swatch:hover { transform: scale(1.08); }
    .swatch.active { border-color: #1d2040; }
    @keyframes rise {
      from { opacity: 0; transform: translateY(18px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .hero, .card, .note { animation: rise 0.55s ease both; }
    .grid .card:nth-child(2) { animation-delay: 0.04s; }
    .grid .card:nth-child(3) { animation-delay: 0.08s; }
    .grid .card:nth-child(4) { animation-delay: 0.12s; }
    .grid .card:nth-child(5) { animation-delay: 0.16s; }
    @media (max-width: 920px) {
      .hero-grid { grid-template-columns: 1fr; }
      .card { grid-column: span 6; }
      .card-wide { grid-column: span 12; }
    }
    @media (max-width: 640px) {
      body { padding: 18px 14px 92px; }
      .hero { padding: 24px; border-radius: 28px; }
      .grid { grid-template-columns: 1fr; }
      .card, .card-wide { grid-column: span 1; min-height: 0; }
      h1 { font-size: 42px; }
      #oshi-toggle { width: 52px; height: 52px; border-radius: 18px; }
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
  <div class="container">
    <section class="hero">
      <div class="tag">✦ 推し色で染まる Demo Stage</div>
      <div class="hero-grid">
        <div>
          <h1>Idol Auth を<br><span class="accent">推しメンカラー</span>で試す。</h1>
          <p class="subtitle">ローカル OIDC デモ、Kratos ブラウザフロー、MFA 設定をひとつのステージにまとめた確認 UI です。右下のパレットから色を切り替えると、画面全体の温度感がそのまま変わります。</p>
          <ul class="microcopy">
            <li>12 色の推し色パレット</li>
            <li>OIDC / MFA / Kratos flow を横断</li>
            <li>モバイルでも崩れない 1 カラム対応</li>
          </ul>
        </div>
        <aside class="feature-panel">
          <h2>今日の見どころ</h2>
          <p>認証そのものは堅く、見た目は柔らかく。first-party / partner の違い、セキュリティ設定、登録導線を色付きのカードで切り替えやすくしています。</p>
          <div class="feature-stack">
            <div class="feature-card">
              <strong>First-party login</strong>
              <span>skip consent と shared account の最短フロー確認用。</span>
            </div>
            <div class="feature-card">
              <strong>Partner login</strong>
              <span>consent ありの third-party 体験をそのまま見られます。</span>
            </div>
            <div class="feature-card">
              <strong>Security settings</strong>
              <span>TOTP とパスワード設定を同じトーンで横断できます。</span>
            </div>
          </div>
        </aside>
      </div>
    </section>
    <div class="grid">
      <a class="card card-primary card-wide card-tall" href="/oauth/start">
        <div class="card-icon">→</div>
        <div class="card-kicker">Main Flow</div>
        <div class="card-title">ファーストパーティでログイン</div>
        <div class="card-desc">メインアプリ向けの OIDC フローを開始します。shared account と PKCE の標準導線を一番短く確認できます。</div>
        <div class="card-meta">OAuth2 authorization_code / PKCE</div>
      </a>
      <a class="card card-wide" href="/oauth/start?app=partner">
        <div class="card-icon">⇄</div>
        <div class="card-kicker">Consent Flow</div>
        <div class="card-title">パートナーアプリでログイン</div>
        <div class="card-desc">third-party client で flow を開始し、consent 画面まで含めてブラウザ導線を確認します。</div>
        <div class="card-meta">Hydra consent / role claims</div>
      </a>
      <a class="card" href="/registration">
        <div class="card-icon">+</div>
        <div class="card-kicker">Onboarding</div>
        <div class="card-title">アカウントを作成</div>
        <div class="card-desc">Kratos の新規登録フローを開き、shared account の作成体験を確認します。</div>
      </a>
      <a class="card" href="/login">
        <div class="card-icon">◉</div>
        <div class="card-kicker">Entry</div>
        <div class="card-title">ログイン画面を開く</div>
        <div class="card-desc">現在のセッション状態に応じたログイン UI を確認します。</div>
      </a>
      <a class="card" href="/settings">
        <div class="card-icon">◈</div>
        <div class="card-kicker">Security</div>
        <div class="card-title">セキュリティ設定</div>
        <div class="card-desc">MFA、パスワード変更、セキュリティ設定のフローをまとめて確認します。</div>
      </a>
    </div>
    <div class="note">推しメンカラーは右下の <strong>✦</strong> から切り替えられます。ログイン中は選択した色がアカウントに保存され、次回アクセス時も自動で引き継がれます。</div>
  </div>
  <div id="oshi-picker">
    <div id="oshi-swatches" aria-label="推しメンカラーパレット"></div>
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

func selectedAppSpec(cfg *demo.Config, key string) demo.DemoAppSpec {
	if key == "partner" {
		return demo.PartnerAppSpec(cfg)
	}
	return demo.PrimaryAppSpec(cfg)
}

func exchangeCode(ctx context.Context, cfg *demo.Config, clientID, verifier, code string) (map[string]any, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("client_id", clientID)
	values.Set("redirect_uri", strings.TrimRight(cfg.AppURL, "/")+"/oauth/callback")
	values.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.HydraPublicURL, "/")+"/oauth2/token", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return decoded, nil
}

func renderToken(w http.ResponseWriter, tokenResp map[string]any, oshiColor string) {
	const tpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>OIDC トークン — Idol Auth</title>
  <style>
    :root {
      --oshi: #b2b2ff;
      --oshi-deep: #4c4cc6;
      --oshi-soft: rgba(178,178,255,0.18);
      --oshi-line: rgba(178,178,255,0.42);
    }
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background:
        radial-gradient(circle at 18% 22%, rgba(255,255,255,0.88) 0%, rgba(255,255,255,0) 26%),
        radial-gradient(circle at 82% 14%, var(--oshi-soft) 0%, rgba(255,255,255,0) 30%),
        linear-gradient(160deg, #fff8fb 0%, #f4f6ff 48%, #edfaff 100%);
      color: #1d2040;
      font-family: "Avenir Next", "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
      padding: 32px 18px 86px;
    }
    .container { max-width: 860px; margin: 0 auto; }
    .panel {
      background: rgba(255,255,255,0.8);
      border: 1px solid rgba(255,255,255,0.84);
      border-radius: 32px;
      padding: 28px;
      box-shadow: 0 26px 70px rgba(59,61,109,0.13);
      backdrop-filter: blur(24px);
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      background: rgba(255,255,255,0.76);
      border: 1px solid var(--oshi-line);
      border-radius: 100px;
      padding: 7px 14px;
      font-size: 11px;
      font-weight: 700;
      color: var(--oshi-deep);
      letter-spacing: 0.1em;
      text-transform: uppercase;
      margin-bottom: 20px;
    }
    h1 {
      margin: 0 0 8px;
      font-size: clamp(28px, 4vw, 46px);
      font-weight: 800;
      letter-spacing: -0.05em;
      line-height: 1.02;
    }
    .lede {
      margin: 0 0 20px;
      color: #6f7394;
      line-height: 1.75;
      font-size: 15px;
    }
    .back {
      display: inline-block;
      color: #6f7394;
      font-size: 14px;
      text-decoration: none;
      margin-bottom: 22px;
      transition: color 0.15s;
    }
    .back:hover { color: var(--oshi-deep); }
    pre {
      margin: 0;
      background: linear-gradient(180deg, rgba(31,35,72,0.98), rgba(23,26,58,0.98));
      border: 1px solid rgba(29,32,64,0.08);
      border-radius: 24px;
      padding: 24px;
      font-family: "SF Mono", "Fira Code", "Cascadia Code", monospace;
      font-size: 13px;
      line-height: 1.7;
      color: #bff8ff;
      overflow-x: auto;
      white-space: pre-wrap;
      word-break: break-word;
      box-shadow: inset 0 1px 0 rgba(255,255,255,0.04);
    }
    #oshi-picker { position: fixed; right: 18px; bottom: 18px; z-index: 20; }
    #oshi-toggle {
      width: 58px;
      height: 58px;
      border-radius: 20px;
      border: 1px solid rgba(255,255,255,0.84);
      background: linear-gradient(180deg, rgba(255,255,255,0.96), var(--oshi-soft));
      color: var(--oshi-deep);
      font-size: 24px;
      cursor: pointer;
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      backdrop-filter: blur(24px);
    }
    #oshi-swatches {
      display: none;
      grid-template-columns: repeat(4, 1fr);
      gap: 10px;
      width: 188px;
      margin-bottom: 12px;
      padding: 14px;
      border-radius: 22px;
      background: rgba(255,255,255,0.86);
      border: 1px solid rgba(255,255,255,0.84);
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      backdrop-filter: blur(24px);
    }
    .swatch {
      width: 100%;
      aspect-ratio: 1;
      border-radius: 999px;
      border: 2px solid transparent;
      cursor: pointer;
    }
    .swatch.active { border-color: #1d2040; }
    @media (max-width: 640px) {
      .panel { padding: 22px; border-radius: 24px; }
      pre { padding: 18px; font-size: 12px; }
      #oshi-toggle { width: 52px; height: 52px; border-radius: 18px; }
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
      root.style.setProperty('--oshi-line', 'rgba('+c[0]+','+c[1]+','+c[2]+',0.42)');
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
  <div class="container">
    <div class="panel">
      <div class="badge">✓ 認証完了</div>
      <h1>OIDC コールバック完了</h1>
      <p class="lede">トークン自体はマスク済みで表示し、レスポンス構造だけを確認できるようにしています。推し色はこの画面でもそのまま維持されます。</p>
      <a class="back" href="/">← デモのホームに戻る</a>
      <pre>{{ .Body }}</pre>
    </div>
  </div>
  <div id="oshi-picker">
    <div id="oshi-swatches" aria-label="推しメンカラーパレット"></div>
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
	b, _ := json.MarshalIndent(sanitizeTokenResponse(tokenResp), "", "  ")
	t := template.Must(template.New("token").Parse(tpl))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = t.Execute(w, struct {
		OshiColor string
		Body      string
	}{
		OshiColor: oshiColor,
		Body:      string(b),
	})
}

func sanitizeTokenResponse(tokenResp map[string]any) map[string]any {
	if tokenResp == nil {
		return nil
	}
	sanitized := make(map[string]any, len(tokenResp))
	for key, value := range tokenResp {
		switch key {
		case "access_token", "refresh_token", "id_token":
			sanitized[key] = "<redacted>"
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}
