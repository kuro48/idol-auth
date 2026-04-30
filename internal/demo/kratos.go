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
	ID string `json:"id"`
	State string `json:"state"`
	UI struct {
		Action string       `json:"action"`
		Method string       `json:"method"`
		Nodes  []KratosNode `json:"nodes"`
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
	Text  *struct {
		ID      int    `json:"id"`
		Text    string `json:"text"`
		Type    string `json:"type"`
		Context struct {
			Secret string `json:"secret"`
		} `json:"context"`
	} `json:"text"`
	Required bool `json:"required"`
	Disabled bool `json:"disabled"`
}

type KratosMessage struct {
	Text string `json:"text"`
	Type string `json:"type"`
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
	if cookie := r.Header.Get("Cookie"); cookie != "" {
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
	Title       string
	Description string
	FlowType    string
	OshiColor   string
	Flow        KratosFlow
}

func RenderPage(w http.ResponseWriter, data PageData) error {
	const tmpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }} — Idol Auth</title>
  <style>
    :root {
      --oshi: #b2b2ff;
      --oshi-bg: rgba(178,178,255,0.18);
      --oshi-border: rgba(178,178,255,0.42);
      --oshi-text: #4646b0;
      --oshi-glow: 0 0 0 3px rgba(178,178,255,0.28);
      --surface: rgba(255,255,255,0.86);
    }
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background:
        radial-gradient(circle at 12% 18%, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0) 24%),
        radial-gradient(circle at 84% 14%, var(--oshi-bg) 0%, rgba(255,255,255,0) 34%),
        radial-gradient(circle at 75% 84%, rgba(216,178,255,0.18) 0%, rgba(255,255,255,0) 28%),
        linear-gradient(160deg, #fff8fb 0%, #f4f6ff 50%, #eefcff 100%);
      color: #1a1a2e;
      font-family: "Avenir Next", "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
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
      width: 220px;
      height: 220px;
      left: -70px;
      top: 8%;
      background: var(--oshi);
    }
    body::after {
      width: 280px;
      height: 280px;
      right: -100px;
      bottom: -50px;
      background: rgba(178,255,255,0.7);
    }
    .card {
      position: relative;
      overflow: hidden;
      width: 100%;
      max-width: 470px;
      background: var(--surface);
      border: 1px solid rgba(255,255,255,0.86);
      border-radius: 32px;
      padding: 32px;
      box-shadow: 0 26px 70px rgba(59,61,109,0.13);
      backdrop-filter: blur(24px);
    }
    .card::before {
      content: "";
      position: absolute;
      inset: 0;
      background: linear-gradient(180deg, rgba(255,255,255,0.5), rgba(255,255,255,0));
      pointer-events: none;
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 7px 12px;
      border-radius: 999px;
      background: rgba(255,255,255,0.7);
      border: 1px solid var(--oshi-border);
      color: var(--oshi-text);
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      margin-bottom: 18px;
      position: relative;
      z-index: 1;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 32px;
    }
    .brand-icon {
      width: 34px;
      height: 34px;
      background: var(--oshi-bg);
      border: 1.5px solid var(--oshi-border);
      border-radius: 9px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 15px;
      color: var(--oshi-text);
      flex-shrink: 0;
    }
    .brand-name {
      font-size: 12px;
      font-weight: 700;
      color: #9ca3af;
      letter-spacing: 0.12em;
      text-transform: uppercase;
    }
    h1 { margin: 0 0 8px; font-size: 30px; font-weight: 800; letter-spacing: -0.05em; color: #111827; line-height: 1.02; }
    .description { color: #6b7280; margin: 0 0 28px; font-size: 14px; line-height: 1.6; }
    .alert {
      padding: 11px 14px;
      border-radius: 9px;
      margin-bottom: 14px;
      font-size: 13px;
      line-height: 1.5;
    }
    .alert-error {
      background: rgba(239,68,68,0.07);
      border: 1px solid rgba(239,68,68,0.2);
      color: #dc2626;
    }
    .alert-info {
      background: var(--oshi-bg);
      border: 1px solid var(--oshi-border);
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
      background: #f9f9fd;
      border: 1.5px solid rgba(0,0,0,0.09);
      border-radius: 10px;
      color: #111827;
      font-size: 15px;
      padding: 11px 13px;
      outline: none;
      transition: border-color 0.15s, box-shadow 0.15s;
      width: 100%;
    }
    input:not([type=hidden]):not([type=submit]):focus {
      border-color: var(--oshi);
      box-shadow: var(--oshi-glow);
    }
    input.is-invalid:not([type=hidden]):not([type=submit]) {
      border-color: #f97316;
      box-shadow: 0 0 0 3px rgba(249,115,22,0.16);
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
      color: #1a1a2e;
      border: none;
      border-radius: 10px;
      font-size: 14px;
      font-weight: 700;
      padding: 13px;
      cursor: pointer;
      transition: opacity 0.15s, transform 0.1s;
      width: 100%;
      letter-spacing: 0.01em;
    }
    button:hover, input[type=submit]:hover { opacity: 0.82; }
    button:active, input[type=submit]:active { transform: scale(0.99); }
    .qr-wrap img {
      border-radius: 16px;
      background: white;
      padding: 14px;
      max-width: 200px;
      border: 1px solid rgba(29,32,64,0.08);
      box-shadow: 0 12px 26px rgba(59,61,109,0.08);
    }
    .nav {
      display: flex;
      flex-wrap: wrap;
      gap: 4px 0;
      margin-top: 28px;
      padding-top: 22px;
      border-top: 1px solid rgba(0,0,0,0.06);
    }
    .nav a {
      font-size: 13px;
      color: #9ca3af;
      text-decoration: none;
      padding: 2px 0;
      transition: color 0.15s;
    }
    .nav a:hover { color: var(--oshi-text); }
    .nav a:not(:last-child)::after { content: '·'; margin: 0 10px; color: rgba(0,0,0,0.15); }
    #oshi-picker {
      position: fixed; bottom: 18px; right: 18px; z-index: 200;
      font-family: "Avenir Next", "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
    }
    #oshi-toggle {
      width: 58px; height: 58px;
      background: linear-gradient(180deg, rgba(255,255,255,0.96), var(--oshi-bg));
      border: 1px solid rgba(255,255,255,0.84); border-radius: 20px;
      font-size: 24px; cursor: pointer;
      display: flex; align-items: center; justify-content: center;
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      transition: transform 0.15s, box-shadow 0.15s;
      color: var(--oshi-text);
      padding: 0;
      backdrop-filter: blur(24px);
    }
    #oshi-toggle:hover { transform: translateY(-2px); box-shadow: 0 22px 52px rgba(59,61,109,0.18); }
    #oshi-swatches {
      display: none;
      grid-template-columns: repeat(4, 1fr);
      gap: 10px;
      width: 188px;
      background: rgba(255,255,255,0.88);
      border: 1px solid rgba(255,255,255,0.84);
      border-radius: 22px;
      padding: 14px;
      box-shadow: 0 18px 44px rgba(59,61,109,0.16);
      position: absolute;
      bottom: 70px; right: 0;
      backdrop-filter: blur(24px);
    }
    .swatch {
      width: 100%; aspect-ratio: 1;
      border-radius: 50%;
      border: 2.5px solid transparent;
      cursor: pointer;
      transition: transform 0.1s, border-color 0.1s;
      outline: none;
      padding: 0;
    }
    .swatch:hover { transform: scale(1.08); }
    .swatch.active { border-color: #1a1a2e; }
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
      .card { padding: 24px 20px; border-radius: 26px; }
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
    var _os=normalizeOshi({{ printf "%q" .OshiColor }})||OSHI[4];
    applyOshi(_os);
  </script>
</head>
<body>
  <div class="card">
    <div class="eyebrow">✦ {{ .FlowType }} flow</div>
    <div class="brand">
      <div class="brand-icon">✦</div>
      <span class="brand-name">Idol Auth</span>
    </div>
    <h1>{{ .Title }}</h1>
    <p class="description">{{ .Description }}</p>
    {{ range .Flow.Messages }}
      <div class="alert {{ if eq .Type "error" }}alert-error{{ else }}alert-info{{ end }}">{{ .Text }}</div>
    {{ end }}
    {{ if eq .FlowType "registration" }}
      <section class="registration-intro" aria-labelledby="registration-intro-title">
        <p class="registration-kicker">Registration</p>
        <h2 id="registration-intro-title">メールアドレスまたは電話番号とパスワードを同じ画面で入力</h2>
        <p class="registration-copy">この画面で、連絡先とログイン用パスワードをまとめて登録します。メールアドレスか電話番号のどちらかは必ず入力してください。</p>
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
      {{ if and (eq $.FlowType "registration") (hasBothIdentifiers $.Flow.UI.Nodes) }}
        <div class="field" id="primary-identifier-field">
          <label for="primary_identifier_display">メールアドレスまたは電話番号</label>
          <input id="primary_identifier_display" name="primary_identifier_display" type="text" autocomplete="email" inputmode="email" placeholder="you@example.com または 09012345678" required>
          <input type="hidden" name="traits.email" id="hidden-traits-email">
          <input type="hidden" name="traits.phone" id="hidden-traits-phone">
          <span id="identifier-type-hint" class="identifier-hint"></span>
        </div>
      {{ end }}
      {{ range .Flow.UI.Nodes }}
        {{ range .Messages }}<div class="alert alert-error">{{ .Text }}</div>{{ end }}
        {{ if eq .Type "img" }}
          <div class="field qr-wrap">
            <label>{{ nodeLabel . }}</label>
            <img src="{{ imageSrc . }}" alt="{{ nodeLabel . }}">
          </div>
        {{ else if eq .Type "text" }}
          <div class="field">
            <label>{{ nodeLabel . }}</label>
            <input type="text" value="{{ textValue . }}" readonly>
          </div>
        {{ else if eq .Attributes.Name "traits.primary_identifier_type" }}
          <input type="hidden" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}">
        {{ else if and (eq $.FlowType "registration") (isPrimaryIdentifierTrait .Attributes.Name) (hasBothIdentifiers $.Flow.UI.Nodes) }}
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
        {{ else if eq .Attributes.Type "submit" }}
          <button type="submit" name="{{ .Attributes.Name }}" value="{{ .Attributes.Value }}">{{ nodeLabel . }}</button>
        {{ else }}
          <div class="field">
            <label for="{{ .Attributes.Name }}">{{ nodeLabel . }}</label>
            <input id="{{ .Attributes.Name }}" name="{{ .Attributes.Name }}" type="{{ inputType .Attributes.Name .Attributes.Type }}" value="{{ .Attributes.Value }}" {{ if .Attributes.Required }}required{{ end }} {{ if .Attributes.Disabled }}disabled{{ end }}>
          </div>
        {{ end }}
      {{ end }}
    </form>
    <nav class="nav">
      <a href="/">ホーム</a>
      <a href="/login">ログイン</a>
      <a href="/registration">新規登録</a>
      <a href="/recovery">復旧</a>
      <a href="/verification">確認</a>
      <a href="/settings">設定</a>
    </nav>
  </div>
  <div id="oshi-picker">
    <button id="oshi-toggle" type="button" title="推しメンカラー">✦</button>
    <div id="oshi-swatches"></div>
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
        var phone=form.querySelector('input[name="traits.phone"]');
        var primaryInput=form.querySelector('#primary_identifier_display');
        var hiddenEmailField=form.querySelector('#hidden-traits-email');
        var hiddenPhoneField=form.querySelector('#hidden-traits-phone');
        var passwordField=form.querySelector('input[name="password"]');
        var passwordStrengthPanel=document.getElementById('password-strength-panel');
        var lastIdentifierType='';
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
        function detectIdentifierType(v){
          if(!v)return null;
          if(v.indexOf('@')>=0)return 'email';
          var cleaned=v.replace(/[\s\-()+]/g,'');
          if(/^[0-9+]/.test(v)&&/^[0-9\s\-().+]+$/.test(v)&&cleaned.length>=7)return 'phone';
          return null;
        }
        function syncCombinedIdentifier(){
          var val=(primaryInput.value||'').trim();
          var type=detectIdentifierType(val);
          var hint=document.getElementById('identifier-type-hint');
          if(type==='email'){
            hiddenEmailField.value=val;
            hiddenPhoneField.value='';
            hidden.value='email';
            primaryInput.setAttribute('autocomplete','email');
            primaryInput.setAttribute('inputmode','email');
            if(hint){hint.textContent='メールアドレスとして登録されます';hint.className='identifier-hint is-email';}
          }else if(type==='phone'){
            hiddenEmailField.value='';
            hiddenPhoneField.value=val;
            hidden.value='phone';
            primaryInput.setAttribute('autocomplete','tel');
            primaryInput.setAttribute('inputmode','tel');
            if(hint){hint.textContent='電話番号として登録されます';hint.className='identifier-hint is-phone';}
          }else{
            hiddenEmailField.value=val;
            hiddenPhoneField.value='';
            hidden.value=val?'email':'';
            if(hint){hint.textContent='';hint.className='identifier-hint';}
          }
        }
        function syncPrimaryIdentifierType(){
          if(primaryInput&&hiddenEmailField&&hiddenPhoneField){syncCombinedIdentifier();return;}
          var emailValue=email&&email.value.trim();
          var phoneValue=phone&&phone.value.trim();
          if(emailValue&&!phoneValue){hidden.value='email';return;}
          if(phoneValue&&!emailValue){hidden.value='phone';return;}
          if(emailValue&&phoneValue){hidden.value=lastIdentifierType||hidden.value||'email';return;}
          hidden.value='';
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
        if(!primaryInput){
          if(email){
            email.autocomplete='email';
            email.placeholder='you@example.com';
          }
          if(phone){
            phone.autocomplete='tel';
            phone.placeholder='09012345678';
          }
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
        if(primaryInput){
          primaryInput.addEventListener('input',syncPrimaryIdentifierType);
        }else{
          if(email){
            email.addEventListener('input',function(){
              lastIdentifierType='email';
              syncPrimaryIdentifierType();
            });
          }
          if(phone){
            phone.addEventListener('input',function(){
              lastIdentifierType='phone';
              syncPrimaryIdentifierType();
            });
          }
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
		"nodeLabel": func(node KratosNode) string {
			if node.Meta.Label != nil && node.Meta.Label.Text != "" {
				return node.Meta.Label.Text
			}
			if node.Attributes.Text != nil && node.Attributes.Text.Text != "" {
				return node.Attributes.Text.Text
			}
			if node.Attributes.Value != nil {
				return fmt.Sprint(node.Attributes.Value)
			}
			return node.Attributes.Name
		},
		"imageSrc": func(node KratosNode) template.URL {
			return template.URL(node.Attributes.Src)
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
	}).Parse(tmpl))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.Execute(w, data)
}
