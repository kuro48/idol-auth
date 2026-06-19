package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type KratosFlow struct {
	ID    string `json:"id"`
	State string `json:"state"`
	UI    struct {
		Action   string          `json:"action"`
		Method   string          `json:"method"`
		Nodes    []KratosNode    `json:"nodes"`
		Messages []KratosMessage `json:"messages"`
	} `json:"ui"`
	Messages []KratosMessage `json:"messages"`
}

type KratosNode struct {
	Type       string           `json:"type"`
	Group      string           `json:"group"`
	Messages   []KratosMessage  `json:"messages"`
	Meta       KratosNodeMeta   `json:"meta"`
	Attributes KratosAttributes `json:"attributes"`
}

type KratosNodeMeta struct {
	Label *struct {
		Text string `json:"text"`
	} `json:"label"`
}

type KratosAttributes struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
	ID    string `json:"id"`
	Src   string `json:"src"`
	Href  string `json:"href"`
	Text  *struct {
		ID      int    `json:"id"`
		Text    string `json:"text"`
		Type    string `json:"type"`
		Context struct {
			Secret string `json:"secret"`
		} `json:"context"`
	} `json:"text"`
	Required       bool   `json:"required"`
	Disabled       bool   `json:"disabled"`
	OnClick        string `json:"onclick"`
	OnClickTrigger string `json:"onclickTrigger"`
	Async          bool   `json:"async"`
	CrossOrigin    string `json:"crossorigin"`
	Integrity      string `json:"integrity"`
	Nonce          string `json:"nonce"`
}

type KratosMessage struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Type string `json:"type"`
}

var kratosMessageJA = map[int]string{
	4000001: "入力内容に誤りがあります。もう一度お試しください。",
	4000002: "必須項目が入力されていません。",
	4000005: "パスワードが条件を満たしていません。",
	4000006: "メールアドレス・パスワードが正しくありません。",
	4000007: "このメールアドレスまたは電話番号はすでに登録されています。",
	4000015: "アカウントが存在しないか、ログイン方法が設定されていません。",
	4000016: "バックアップ用回復コードが無効です。",
	4010001: "回復コードが無効です。もう一度お試しください。",
	4010002: "回復リンクが無効か、すでに使用されています。",
	4060001: "パスワードがポリシーを満たしていません。",
	4060003: "このパスワードはよく使われているため使用できません。",
	4060005: "パスワードがメールアドレスや電話番号に似すぎているため使用できません。",
	4070001: "確認リンクが無効か、すでに使用されています。",
	4070005: "認証はすでに完了しています。",
	1010001: "ログインしました。",
	1040001: "変更が保存されました。",
	1040004: "認証アプリの設定が完了しました。",
	1060001: "アカウントの回復に成功しました。60分以内にパスワードを変更してください。",
	1070001: "メールアドレスの確認が完了しました。",
}

func translateMessage(msg KratosMessage) string {
	if ja, ok := kratosMessageJA[msg.ID]; ok {
		return ja
	}
	return msg.Text
}

var kratosLabelJA = map[string]string{ // #nosec G101 -- UI label translations, not credentials
	"ID":                        "メールアドレス / 電話番号",
	"Password":                  "パスワード",
	"New Password":              "新しいパスワード",
	"Email":                     "メールアドレス",
	"Phone":                     "電話番号",
	"Phone Number":              "電話番号",
	"Display Name":              "表示名",
	"Sign in with password":     "パスワードでログイン",
	"Sign up":                   "アカウントを作成する",
	"Save":                      "保存する",
	"Submit":                    "送信",
	"Continue":                  "続行",
	"Send recovery link":        "回復リンクを送信",
	"Send verification code":    "確認コードを送信",
	"Verify code":               "コードを確認",
	"Resend code":               "コードを再送",
	"Authenticator app QR code": "認証アプリ QR コード",
	"Authenticator secret":      "シークレットキー",
	"Disable this method":       "この方法を無効化",
	"Backup Recovery Codes":     "バックアップ回復コード",
	"Reveal codes":              "コードを表示",
	"Download":                  "ダウンロード",
	"Add Authenticator App":     "認証アプリを追加",
	"Confirm":                   "確認する",
	"Reauthenticate":            "再認証",
	"Sign in with passkey":      "パスキーでログイン",
	"Use passkey":               "パスキーを使用",
	"Register a passkey":        "パスキーを登録する",
	"Add passkey":               "パスキーを追加",
	"Remove passkey":            "パスキーを削除",
}

func translateLabel(text string) string {
	if ja, ok := kratosLabelJA[text]; ok {
		return ja
	}
	return text
}

func safeImageSrc(raw string) template.URL {
	src := strings.TrimSpace(raw)
	if src == "" {
		return ""
	}
	for _, prefix := range []string{
		"data:image/png;base64,",
		"data:image/jpeg;base64,",
		"data:image/gif;base64,",
		"data:image/webp;base64,",
	} {
		if strings.HasPrefix(src, prefix) {
			return template.URL(src) // #nosec G203 -- src is restricted to image data URL prefixes.
		}
	}
	u, err := url.Parse(src)
	if err != nil || u.Host == "" {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return template.URL(u.String()) // #nosec G203 -- URL is parsed and restricted to http(s) with a host.
	default:
		return ""
	}
}

type KratosFlowClient struct {
	apiBaseURL     string
	browserBaseURL string
	httpClient     *http.Client
}

func NewKratosFlowClient(apiBaseURL, browserBaseURL string) *KratosFlowClient {
	return &KratosFlowClient{
		apiBaseURL:     normalizeBaseURL(apiBaseURL),
		browserBaseURL: normalizeBaseURL(browserBaseURL),
		httpClient:     &http.Client{},
	}
}

func (c *KratosFlowClient) BrowserInitURL(flowType string) string {
	return c.browserBaseURL + "/self-service/" + flowType + "/browser"
}

func (c *KratosFlowClient) GetFlow(ctx context.Context, r *http.Request, flowType, flowID string) (KratosFlow, error) {
	endpoint := c.apiBaseURL + "/self-service/" + flowType + "/flows?" + url.Values{"id": []string{flowID}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return KratosFlow{}, fmt.Errorf("build kratos flow request: %w", err)
	}
	if cookie := FilterOryCookies(r.Header.Get("Cookie")); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return KratosFlow{}, fmt.Errorf("call kratos flow: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return KratosFlow{}, fmt.Errorf("kratos flow returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var flow KratosFlow
	if err := json.NewDecoder(resp.Body).Decode(&flow); err != nil {
		return KratosFlow{}, fmt.Errorf("decode kratos flow: %w", err)
	}
	return flow, nil
}

type PageData struct {
	Title            string
	Description      string
	FlowType         string
	OshiColor        string
	LegalBaseURL     string
	AccountCenterURL string
	TurnstileSiteKey string
	Flow             KratosFlow
}

func RenderPage(w http.ResponseWriter, data PageData) error {
	const tmpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }} | OshiLink</title>
  {{ if and (eq .FlowType "registration") .TurnstileSiteKey }}<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>{{ end }}
  <style>
    :root {
      --oshi:#ffb2d8;
      --oshi-weak:#fce7f3;
      --oshi-soft:#fbcfe8;
      --bg:#fdf8fc;
      --surface:#ffffff;
      --surface-2:#fef9fd;
      --text:#2a1520;
      --muted:#9d748f;
      --border:#f0d0e8;
      --danger:#e85d75;
      --radius-lg:32px;
      --radius-md:22px;
      --radius-sm:18px;
      --oshi-bg:color-mix(in srgb,var(--oshi) 14%,transparent);
      --oshi-border:color-mix(in srgb,var(--oshi) 34%,transparent);
      --oshi-text:var(--oshi);
      --oshi-glow:0 0 0 4px color-mix(in srgb,var(--oshi) 18%,transparent);
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg:#1a1018;
        --surface:#251820;
        --surface-2:#2e1f28;
        --text:#fff0f8;
        --muted:#c8a8bd;
        --border:#4a2e40;
        --oshi-weak:#3d1a2a;
        --oshi-soft:#5a2a40;
      }
    }
    *, *::before, *::after { box-sizing: border-box; }
    a { color: var(--oshi); text-decoration: none; }
    a:hover { opacity: 0.8; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px 18px 90px;
      position: relative;
      overflow-x: hidden;
    }
    body::before, body::after {
      content: "";
      position: fixed;
      border-radius: 999px;
      filter: blur(18px);
      opacity: 0.42;
      pointer-events: none;
    }
    body::before {
      width: 260px;
      height: 260px;
      left: -80px;
      top: 6%;
      background: var(--oshi-soft);
    }
    body::after {
      width: 300px;
      height: 300px;
      right: -110px;
      bottom: -60px;
      background: var(--oshi-weak);
    }
    .card {
      position: relative;
      overflow: hidden;
      width: 100%;
      max-width: 480px;
      background: var(--surface);
      border: 2px solid var(--border);
      border-radius: var(--radius-lg);
      padding: 32px;
      box-shadow: 0 4px 24px color-mix(in srgb,var(--oshi) 10%,rgba(0,0,0,0.06));
      backdrop-filter: saturate(160%) blur(14px);
      -webkit-backdrop-filter: saturate(160%) blur(14px);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 28px;
    }
    .brand-mark {
      width: 42px;
      height: 42px;
      background: var(--oshi);
      border-radius: 14px;
      display: grid;
      place-items: center;
      font-size: 20px;
      color: #fff;
      flex-shrink: 0;
      font-weight: 900;
      letter-spacing: -0.02em;
      border: 2px solid color-mix(in srgb,var(--oshi) 35%,transparent);
    }
    .brand-text strong {
      display: block;
      font-size: 16px;
      line-height: 1.1;
      letter-spacing: -0.01em;
      color: var(--text);
    }
    .brand-text span {
      font-size: 11px;
      color: var(--muted);
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.12em;
    }
    h1 { margin: 0 0 8px; font-size: 30px; font-weight: 900; letter-spacing: -0.03em; color: var(--text); line-height: 1.02; }
    .description { color: var(--muted); margin: 0 0 28px; font-size: 14px; line-height: 1.6; }
    .alert {
      padding: 11px 14px;
      border-radius: 9px;
      margin-bottom: 14px;
      font-size: 13px;
      line-height: 1.5;
    }
    .alert-error {
      background: color-mix(in srgb,var(--danger) 8%,transparent);
      border: 2px solid color-mix(in srgb,var(--danger) 24%,transparent);
      color: var(--danger);
    }
    .alert-info {
      background: var(--oshi-bg);
      border: 2px solid var(--oshi-border);
      color: var(--oshi-text);
    }
    .registration-intro {
      margin: 0 0 22px;
      padding: 18px;
      border-radius: 24px;
      background:
        linear-gradient(145deg, rgba(255,255,255,0.9), rgba(255,255,255,0.72)),
        radial-gradient(circle at top right, var(--oshi-bg), rgba(255,255,255,0) 58%);
      border: 1px solid rgba(255,255,255,0.92);
      box-shadow: 0 16px 36px rgba(59,61,109,0.08);
    }
    .registration-kicker {
      margin: 0 0 8px;
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--oshi-text);
    }
    .registration-intro h2 {
      margin: 0;
      font-size: 20px;
      line-height: 1.25;
      letter-spacing: -0.03em;
      color: #111827;
    }
    .registration-copy {
      margin: 10px 0 0;
      font-size: 13px;
      line-height: 1.65;
      color: #4b5563;
    }
    .registration-guidance {
      margin-top: 16px;
      padding: 14px 15px;
      border-radius: 18px;
      background: rgba(255,255,255,0.76);
      border: 1px solid rgba(17,24,39,0.06);
    }
    .guidance-title {
      margin: 0 0 6px;
      font-size: 12px;
      font-weight: 700;
      color: #111827;
    }
    .guidance-copy {
      margin: 0;
      font-size: 13px;
      line-height: 1.6;
      color: #4b5563;
    }
    form { display: flex; flex-direction: column; gap: 18px; }
    .field { display: flex; flex-direction: column; gap: 6px; }
    label {
      font-size: 11px;
      font-weight: 700;
      color: #9ca3af;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    input:not([type=hidden]):not([type=submit]) {
      background: var(--surface-2);
      border: 2px solid var(--border);
      border-radius: 16px;
      color: var(--text);
      font-size: 15px;
      padding: 14px 16px;
      outline: none;
      transition: border-color 160ms ease, box-shadow 160ms ease, background 160ms ease;
      width: 100%;
    }
    input:not([type=hidden]):not([type=submit]):focus {
      border-color: var(--oshi);
      background: var(--surface);
      box-shadow: var(--oshi-glow);
    }
    input.is-invalid:not([type=hidden]):not([type=submit]) {
      border-color: var(--danger);
      box-shadow: 0 0 0 3px color-mix(in srgb,var(--danger) 16%,transparent);
    }
    input[readonly] { opacity: 0.55; cursor: default; }
    .password-strength-panel {
      margin-top: -8px;
      padding: 14px 15px 15px;
      border-radius: 18px;
      background: rgba(255,255,255,0.8);
      border: 1px solid rgba(17,24,39,0.06);
      box-shadow: inset 0 1px 0 rgba(255,255,255,0.72);
    }
    .password-strength-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 10px;
      font-size: 12px;
      font-weight: 700;
      color: #374151;
    }
    .password-strength-head strong {
      font-size: 13px;
      color: #111827;
    }
    .password-strength-bar {
      width: 100%;
      height: 10px;
      border-radius: 999px;
      background: rgba(148,163,184,0.18);
      overflow: hidden;
    }
    .password-strength-bar span {
      display: block;
      height: 100%;
      width: 0%;
      border-radius: inherit;
      background: linear-gradient(90deg, #fb7185 0%, #f59e0b 50%, #34d399 100%);
      transition: width 0.2s ease;
    }
    .password-strength-copy {
      margin: 10px 0 0;
      font-size: 12px;
      line-height: 1.6;
      color: #6b7280;
    }
    .password-checklist {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 8px;
      margin-top: 12px;
    }
    .password-checklist span {
      display: inline-flex;
      align-items: center;
      gap: 7px;
      font-size: 12px;
      color: #6b7280;
    }
    .password-checklist span::before {
      content: "•";
      color: rgba(107,114,128,0.75);
      font-size: 15px;
      line-height: 1;
    }
    .password-checklist span.is-ok {
      color: #047857;
    }
    .password-checklist span.is-ok::before {
      content: "✓";
      color: #10b981;
    }
    button, input[type=submit] {
      background: var(--oshi);
      color: #fff;
      border: 2px solid var(--oshi);
      border-radius: 999px;
      font-size: 14px;
      font-weight: 900;
      padding: 14px 18px;
      cursor: pointer;
      transition: transform 160ms ease, opacity 160ms ease;
      width: 100%;
      letter-spacing: 0.01em;
      line-height: 1;
    }
    button:hover, input[type=submit]:hover { transform: translateY(-1px); opacity: 0.92; }
    button:active, input[type=submit]:active { transform: scale(0.99); }
    .passkey-btn { background: var(--oshi-bg); color: var(--oshi-text); border-color: var(--oshi-border); font-weight: 700; }
    .passkey-btn:disabled { opacity: 0.45; cursor: not-allowed; transform: none; }
    .totp-reveal {
      border: 2px solid var(--border);
      border-radius: var(--radius-sm);
      background: var(--surface-2);
      overflow: hidden;
    }
    .totp-reveal summary {
      padding: 12px 16px;
      cursor: pointer;
      font-size: 13px;
      font-weight: 700;
      color: var(--oshi);
      user-select: none;
      list-style: none;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .totp-reveal summary::-webkit-details-marker { display: none; }
    .totp-reveal summary::before { content: "▶"; font-size: 9px; }
    .totp-reveal[open] summary::before { content: "▼"; }
    .totp-reveal > .field { padding: 0 16px 16px; }
    .qr-wrap img {
      border-radius: 16px;
      background: white;
      padding: 14px;
      max-width: 200px;
      border: 2px solid var(--border);
    }
    .nav {
      display: flex;
      flex-wrap: wrap;
      gap: 4px 0;
      margin-top: 28px;
      padding-top: 22px;
      border-top: 2px solid var(--border);
    }
    .nav a {
      font-size: 13px;
      color: var(--muted);
      text-decoration: none;
      padding: 2px 0;
      transition: color 160ms ease;
    }
    .nav a:hover { color: var(--oshi); }
    .nav a:not(:last-child)::after { content: '·'; margin: 0 10px; color: var(--border); }
    #oshi-picker {
      position: fixed; bottom: 18px; right: 18px; z-index: 200;
      font-family: "Avenir Next", "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
    }
    #oshi-toggle {
      width: 56px; height: 56px;
      background: color-mix(in srgb,var(--surface) 90%,transparent);
      border: 2px solid var(--border); border-radius: 20px;
      font-size: 22px; cursor: pointer;
      display: grid; place-items: center;
      box-shadow: 0 4px 16px color-mix(in srgb,var(--oshi) 16%,rgba(0,0,0,0.06));
      transition: transform 160ms ease, box-shadow 160ms ease;
      color: var(--oshi);
      padding: 0;
      backdrop-filter: saturate(160%) blur(14px);
      -webkit-backdrop-filter: saturate(160%) blur(14px);
    }
    #oshi-toggle:hover { transform: translateY(-2px); box-shadow: 0 8px 24px color-mix(in srgb,var(--oshi) 22%,rgba(0,0,0,0.08)); }
    #oshi-swatches {
      display: none;
      grid-template-columns: repeat(4, 1fr);
      gap: 10px;
      width: 188px;
      background: color-mix(in srgb,var(--surface) 94%,transparent);
      border: 2px solid var(--border);
      border-radius: 22px;
      padding: 14px;
      box-shadow: 0 8px 32px color-mix(in srgb,var(--oshi) 12%,rgba(0,0,0,0.08));
      position: absolute;
      bottom: 68px; right: 0;
      backdrop-filter: saturate(160%) blur(14px);
      -webkit-backdrop-filter: saturate(160%) blur(14px);
    }
    .swatch {
      width: 100%; aspect-ratio: 1;
      border-radius: 50%;
      border: 2.5px solid transparent;
      cursor: pointer;
      transition: transform 160ms ease, border-color 160ms ease;
      outline: none;
      padding: 0;
    }
    .swatch:hover { transform: scale(1.1); }
    .swatch.active { border-color: var(--text); }
    .identifier-hint {
      font-size: 11px;
      color: #9ca3af;
      margin-top: 4px;
      min-height: 16px;
      display: block;
      transition: color 0.15s;
    }
    .identifier-hint.is-email, .identifier-hint.is-phone { color: #047857; }
    @media (max-width: 640px) {
      .card { padding: 24px 20px; border-radius: 28px; }
      h1 { font-size: 26px; }
      .registration-intro h2 { font-size: 18px; }
      .password-checklist { grid-template-columns: 1fr; }
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
    function phx(h){return[parseInt(h.slice(1,3),16),parseInt(h.slice(3,5),16),parseInt(h.slice(5,7),16)];}
    function thx(r,g,b){return'#'+[r,g,b].map(function(v){return Math.min(255,Math.max(0,v)).toString(16).padStart(2,'0');}).join('');}
    function applyOshi(color){
      var c=phx(color),root=document.documentElement;
      root.style.setProperty('--oshi',color);
      root.style.setProperty('--oshi-bg','rgba('+c[0]+','+c[1]+','+c[2]+',0.16)');
      root.style.setProperty('--oshi-border','rgba('+c[0]+','+c[1]+','+c[2]+',0.42)');
      root.style.setProperty('--oshi-text',thx(c[0]-90,c[1]-90,c[2]-90));
      root.style.setProperty('--oshi-glow','0 0 0 3px rgba('+c[0]+','+c[1]+','+c[2]+',0.28)');
    }
    function persistOshi(color){
      fetch('/ui/theme',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        credentials:'same-origin',
        body:JSON.stringify({oshi_color:color})
      }).catch(function(){});
    }
    var _os=normalizeOshi({{ printf "%q" .OshiColor }})||OSHI[1];
    applyOshi(_os);
  </script>
</head>
<body>
  <div class="card">
    <div class="brand">
      <div class="brand-mark" aria-hidden="true">★</div>
      <div class="brand-text">
        <strong>OshiLink</strong>
        <span>ACCOUNT</span>
      </div>
    </div>
    <h1>{{ .Title }}</h1>
    <p class="description">{{ .Description }}</p>
    {{ range .Flow.Messages }}
      <div class="alert {{ if eq .Type "error" }}alert-error{{ else }}alert-info{{ end }}">{{ translateMessage . }}</div>
    {{ end }}
    {{ range .Flow.UI.Messages }}
      <div class="alert {{ if eq .Type "error" }}alert-error{{ else }}alert-info{{ end }}">{{ translateMessage . }}</div>
    {{ end }}
    {{ if eq .FlowType "registration" }}
      <section class="registration-intro" aria-labelledby="registration-intro-title">
        <p class="registration-kicker">Registration</p>
        <h2 id="registration-intro-title">メールアドレスとパスワードを入力</h2>
        <p class="registration-copy">登録にはメールアドレスが必要です。登録後、メールに届く確認コードを入力するとログインできるようになります。</p>
        <div class="registration-guidance">
          <p class="guidance-title">パスワード条件</p>
          <p class="guidance-copy">8文字以上で、英大文字・英小文字・数字・記号のうち3種類以上を含めてください。</p>
        </div>
      </section>
      <div id="password-strength-panel" class="password-strength-panel" hidden>
        <div class="password-strength-head">
          <span>パスワードの安全性</span>
          <strong data-strength-label>未入力</strong>
        </div>
        <div class="password-strength-bar" aria-hidden="true"><span data-strength-fill></span></div>
        <p class="password-strength-copy" data-strength-copy>8文字以上かつ3種類以上を満たすと登録できます。</p>
        <div class="password-checklist">
          <span data-rule="length">8文字以上</span>
          <span data-rule="upper">英大文字</span>
          <span data-rule="lower">英小文字</span>
          <span data-rule="digit">数字</span>
          <span data-rule="symbol">記号</span>
          <span data-rule="variety">3種類以上</span>
        </div>
      </div>
    {{ end }}
    <form id="kratos-flow-form" action="{{ .Flow.UI.Action }}" method="{{ .Flow.UI.Method }}" data-flow-type="{{ .FlowType }}">
      {{ range .Flow.UI.Nodes }}
        {{ $passkeyOnClick := passkeyOnClick . }}
        {{ range .Messages }}<div class="alert alert-error">{{ translateMessage . }}</div>{{ end }}
        {{ if eq .Type "img" }}
          <details class="totp-reveal">
            <summary>QRコードを表示する</summary>
            <div class="field qr-wrap">
              <label>{{ nodeLabel . }}</label>
              <img src="{{ imageSrc . }}" alt="{{ nodeLabel . }}">
            </div>
          </details>
        {{ else if eq .Type "text" }}
          <details class="totp-reveal">
            <summary>シークレットキーを表示する</summary>
            <div class="field">
              <label>{{ nodeLabel . }}</label>
              <input type="text" value="{{ textValue . }}" readonly>
            </div>
          </details>
        {{ else if eq .Attributes.Name "traits.primary_identifier_type" }}
          <input type="hidden" name="{{ .Attributes.Name }}" value="email">
        {{ else if and (eq $.FlowType "registration") (eq .Attributes.Name "traits.phone") }}
        {{ else if eq .Attributes.Type "hidden" }}
          <input type="hidden" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}">
        {{ else if and (eq $.FlowType "registration") (eq .Attributes.Name "method") (eq .Attributes.Value "profile") }}
          {{ if not (hasPasswordNode $.Flow.UI.Nodes) }}
            <div class="field">
              <label for="password">Password</label>
              <input id="password" name="password" type="password" required autocomplete="new-password">
            </div>
          {{ end }}
          <button type="submit" name="{{ .Attributes.Name }}" value="password">{{ nodeLabel . }}</button>
        {{ else if eq .Type "script" }}
          <script src="{{ .Attributes.Src }}" async{{ if .Attributes.CrossOrigin }} crossorigin="{{ .Attributes.CrossOrigin }}"{{ end }}{{ if .Attributes.Integrity }} integrity="{{ .Attributes.Integrity }}"{{ end }}{{ if .Attributes.Nonce }} nonce="{{ .Attributes.Nonce }}"{{ end }}></script>
        {{ else if and (eq .Attributes.Type "button") (ne $passkeyOnClick "") }}
          <button type="button" class="passkey-btn" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}" onclick="{{ $passkeyOnClick }}"{{ if .Attributes.Disabled }} disabled{{ end }}>{{ nodeLabel . }}</button>
        {{ else if eq .Attributes.Type "submit" }}
          <button type="submit" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}"{{ if .Attributes.Disabled }} disabled{{ end }}>{{ nodeLabel . }}</button>
        {{ else if eq .Type "a" }}
          <a href="{{ safeHref .Attributes.Href }}" style="display:block;text-align:center;padding:14px 18px;background:var(--oshi);color:#fff;border-radius:999px;font-weight:900;font-size:14px;text-decoration:none;">{{ nodeLabel . }}</a>
        {{ else }}
          <div class="field">
            <label for="{{ .Attributes.Name }}">{{ nodeLabel . }}</label>
            <input id="{{ .Attributes.Name }}" name="{{ .Attributes.Name }}" type="{{ inputType .Attributes.Name .Attributes.Type }}" value="{{ .Attributes.Value }}" {{ if .Attributes.Required }}required{{ end }} {{ if .Attributes.Disabled }}disabled{{ end }}>
          </div>
        {{ end }}
      {{ end }}
      {{ if and (eq .FlowType "registration") .TurnstileSiteKey }}
        <div class="cf-turnstile" data-sitekey="{{ .TurnstileSiteKey }}"></div>
      {{ end }}
    </form>
    {{ if eq .FlowType "login" }}
    <p class="auth-note"><a href="/recovery">パスワードを忘れた場合</a></p>
	    {{ end }}
	    <nav class="nav">
	      {{ if eq .FlowType "settings" }}
	      <a href="{{ accountCenterURL .AccountCenterURL }}">アカウントセンター</a>
	      {{ else }}
	      <a href="/login">ログイン</a>
	      <a href="/registration">新規登録</a>
	      {{ if isPublicAuthFlow .FlowType }}
	      <a href="{{ legalURL .LegalBaseURL "/legal/terms" }}">利用規約</a>
	      <a href="{{ legalURL .LegalBaseURL "/legal/privacy" }}">プライバシーポリシー</a>
	      {{ else }}
	      <a href="/settings">セキュリティ設定</a>
	      {{ end }}
	      {{ end }}
	    </nav>
  </div>
  <div id="oshi-picker" aria-label="推し色を選ぶ">
    <button id="oshi-toggle" type="button" title="推しメンカラー">✦</button>
    <div id="oshi-swatches" aria-label="推し色を選ぶ"></div>
  </div>
  <script>
    (function(){
      var form=document.getElementById('kratos-flow-form');
      if(form && form.dataset.flowType==='registration'){
        var hidden=form.querySelector('input[name="traits.primary_identifier_type"]');
        if(!hidden){
          hidden=document.createElement('input');
          hidden.type='hidden';
          hidden.name='traits.primary_identifier_type';
          form.appendChild(hidden);
        }
        var email=form.querySelector('input[name="traits.email"]');
        var passwordField=form.querySelector('input[name="password"]');
        var passwordStrengthPanel=document.getElementById('password-strength-panel');
        function passwordState(value){
          var hasUpper=/[A-Z]/.test(value);
          var hasLower=/[a-z]/.test(value);
          var hasDigit=/[0-9]/.test(value);
          var hasSymbol=/[^A-Za-z0-9]/.test(value);
          var categoryCount=[hasUpper,hasLower,hasDigit,hasSymbol].filter(Boolean).length;
          var lengthOk=value.length>=8;
          var valid=lengthOk&&categoryCount>=3;
          var score=0;
          if(value.length>=8) score++;
          if(value.length>=12) score++;
          if(categoryCount>=2) score++;
          if(categoryCount>=3) score++;
          if(value.length>=16&&categoryCount===4) score++;
          var label='未入力';
          if(value.length>0&&score<=2){
            label='弱い';
          }else if(score===3){
            label='ふつう';
          }else if(score===4){
            label='強い';
          }else if(score>=5){
            label='とても強い';
          }
          return {
            lengthOk:lengthOk,
            hasUpper:hasUpper,
            hasLower:hasLower,
            hasDigit:hasDigit,
            hasSymbol:hasSymbol,
            categoryCount:categoryCount,
            valid:valid,
            score:score,
            label:label
          };
        }
        function syncPrimaryIdentifierType(){
          hidden.value='email';
        }
        function syncPasswordStrength(){
          if(!passwordField||!passwordStrengthPanel){
            return true;
          }
          var label=passwordStrengthPanel.querySelector('[data-strength-label]');
          var fill=passwordStrengthPanel.querySelector('[data-strength-fill]');
          var copy=passwordStrengthPanel.querySelector('[data-strength-copy]');
          var state=passwordState(passwordField.value);
          passwordStrengthPanel.hidden=false;
          passwordStrengthPanel.querySelectorAll('[data-rule]').forEach(function(rule){
            var key=rule.getAttribute('data-rule');
            var ok=false;
            if(key==='length') ok=state.lengthOk;
            if(key==='upper') ok=state.hasUpper;
            if(key==='lower') ok=state.hasLower;
            if(key==='digit') ok=state.hasDigit;
            if(key==='symbol') ok=state.hasSymbol;
            if(key==='variety') ok=state.categoryCount>=3;
            rule.classList.toggle('is-ok',ok);
          });
          fill.style.width=(state.score/5*100)+'%';
          label.textContent=state.label;
          if(!passwordField.value){
            copy.textContent='8文字以上かつ3種類以上を満たすと登録できます。';
          }else if(state.valid){
            copy.textContent='このパスワードは登録条件を満たしています。';
          }else{
            copy.textContent='8文字以上で、英大文字・英小文字・数字・記号のうち3種類以上が必要です。';
          }
          passwordField.classList.toggle('is-invalid',passwordField.value.length>0&&!state.valid);
          passwordField.setCustomValidity(state.valid||passwordField.value.length===0?'':'パスワードは8文字以上で、英大文字・英小文字・数字・記号のうち3種類以上を含めてください。');
          return state.valid||passwordField.value.length===0;
        }
        if(email){
          email.autocomplete='email';
          email.inputMode='email';
          email.placeholder='you@example.com';
        }
        if(passwordField){
          passwordField.autocomplete='new-password';
          passwordField.minLength=8;
          passwordField.placeholder='8文字以上 / 3種類以上';
          if(passwordStrengthPanel && passwordField.parentNode){
            passwordField.parentNode.insertAdjacentElement('afterend',passwordStrengthPanel);
          }
          passwordField.addEventListener('input',syncPasswordStrength);
          passwordField.addEventListener('blur',syncPasswordStrength);
          syncPasswordStrength();
        }
        if(email){
          email.addEventListener('input',syncPrimaryIdentifierType);
        }
        form.addEventListener('submit',function(){
          syncPrimaryIdentifierType();
          syncPasswordStrength();
        });
        syncPrimaryIdentifierType();
      }
      var sw=document.getElementById('oshi-swatches');
      var tog=document.getElementById('oshi-toggle');
      var cur=normalizeOshi({{ printf "%q" .OshiColor }})||OSHI[4];
      OSHI.forEach(function(c){
        var btn=document.createElement('button');
        btn.type='button';
        btn.className='swatch'+(c===cur?' active':'');
        btn.style.background=c;
        btn.title='推しメンカラー '+(OSHI.indexOf(c)+1);
        btn.addEventListener('click',function(){
          applyOshi(c);
          persistOshi(c);
          document.querySelectorAll('.swatch').forEach(function(s){s.classList.toggle('active',s===btn);});
        });
        sw.appendChild(btn);
      });
      tog.addEventListener('click',function(){
        sw.style.display=sw.style.display==='grid'?'none':'grid';
      });
    })();
  </script>
</body>
</html>`

	t := template.Must(template.New("page").Funcs(template.FuncMap{
		"inputType": func(name, inputType string) string {
			if name == "password" {
				return "password"
			}
			if inputType == "submit" || inputType == "hidden" {
				return inputType
			}
			if inputType == "" {
				return "text"
			}
			return inputType
		},
		"hasBothIdentifiers": func(nodes []KratosNode) bool {
			hasEmail, hasPhone := false, false
			for _, n := range nodes {
				if n.Attributes.Name == "traits.email" {
					hasEmail = true
				}
				if n.Attributes.Name == "traits.phone" {
					hasPhone = true
				}
			}
			return hasEmail && hasPhone
		},
		"isPrimaryIdentifierTrait": func(name string) bool {
			return name == "traits.email" || name == "traits.phone"
		},
		"hasPasswordNode": func(nodes []KratosNode) bool {
			for _, node := range nodes {
				if node.Attributes.Name == "password" {
					return true
				}
			}
			return false
		},
		"isPublicAuthFlow": func(flowType string) bool {
			return flowType == "login" || flowType == "registration"
		},
		"legalURL": func(base, path string) string {
			if strings.TrimSpace(base) == "" {
				return path
			}
			return strings.TrimRight(base, "/") + path
		},
		"accountCenterURL": func(target string) string {
			if strings.TrimSpace(target) == "" {
				return "/account/"
			}
			return target
		},
		"passkeyOnClick": func(node KratosNode) template.JS {
			allowedTriggers := map[string]bool{
				"oryPasskeyLogin":                true,
				"oryPasskeyRegistration":         true,
				"oryPasskeySettingsRegistration": true,
				"oryWebAuthnLogin":               true,
				"oryWebAuthnRegistration":        true,
				"oryWebAuthnSettings":            true,
			}
			trigger := strings.TrimSpace(node.Attributes.OnClickTrigger)
			if trigger != "" && allowedTriggers[trigger] {
				return template.JS("window." + trigger + "()") // #nosec G203 -- trigger is restricted by allowlist above.
			}
			onClick := strings.TrimSpace(node.Attributes.OnClick)
			for allowed := range allowedTriggers {
				if onClick == "window."+allowed+"()" {
					return template.JS(onClick) // #nosec G203 -- onclick is matched exactly against allowlisted function calls.
				}
			}
			return ""
		},
		"nodeLabel": func(node KratosNode) string {
			if node.Meta.Label != nil && node.Meta.Label.Text != "" {
				return translateLabel(node.Meta.Label.Text)
			}
			if node.Attributes.Text != nil && node.Attributes.Text.Text != "" {
				return translateLabel(node.Attributes.Text.Text)
			}
			if node.Attributes.Value != nil {
				return translateLabel(fmt.Sprint(node.Attributes.Value))
			}
			return node.Attributes.Name
		},
		"safeHref": func(raw string) template.URL {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				return ""
			}
			u, err := url.Parse(raw)
			if err != nil || u.Host == "" {
				return ""
			}
			switch strings.ToLower(u.Scheme) {
			case "http", "https":
				return template.URL(u.String()) // #nosec G203 -- URL is parsed and restricted to http(s) with a host.
			default:
				return ""
			}
		},
		"imageSrc": func(node KratosNode) template.URL {
			return safeImageSrc(node.Attributes.Src)
		},
		"textValue": func(node KratosNode) string {
			if node.Attributes.Text != nil {
				if node.Attributes.Text.Context.Secret != "" {
					return node.Attributes.Text.Context.Secret
				}
				return node.Attributes.Text.Text
			}
			if node.Attributes.Value != nil {
				return fmt.Sprint(node.Attributes.Value)
			}
			return ""
		},
		"translateMessage": translateMessage,
	}).Parse(tmpl))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.Execute(w, data)
}
