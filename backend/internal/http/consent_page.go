package http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"html/template"
	"net/http"
	"strings"
)

const consentCSRFCookieName = "idol_auth_consent_csrf"

func writeConsentPage(w http.ResponseWriter, prompt *ConsentPrompt, secureCookies bool, cookieDomain string) error {
	const tpl = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>アプリ連携の確認</title>
  <style nonce="{{ .Nonce }}">
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
        radial-gradient(circle at 16% 20%, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0) 24%),
        radial-gradient(circle at 82% 10%, var(--oshi-soft) 0%, rgba(255,255,255,0) 32%),
        linear-gradient(160deg, #fff8fb 0%, #f4f6ff 48%, #edfaff 100%);
      color: #1d2040;
      font-family: "Avenir Next", "Hiragino Sans", "Yu Gothic", "Noto Sans JP", sans-serif;
      padding: 28px 18px 88px;
    }
    main {
      max-width: 760px;
      margin: 0 auto;
      padding: 28px;
      background: rgba(255,255,255,0.82);
      border-radius: 34px;
      border: 1px solid rgba(255,255,255,0.84);
      box-shadow: 0 26px 70px rgba(59,61,109,0.13);
      backdrop-filter: blur(24px);
      position: relative;
      overflow: hidden;
    }
    main::before {
      content: "";
      position: absolute;
      inset: -10% auto auto -10%;
      width: 260px;
      height: 260px;
      border-radius: 999px;
      background: radial-gradient(circle, var(--oshi-soft) 0%, rgba(255,255,255,0) 70%);
      pointer-events: none;
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 7px 14px;
      border-radius: 999px;
      background: rgba(255,255,255,0.72);
      border: 1px solid var(--oshi-line);
      color: var(--oshi-deep);
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      margin-bottom: 18px;
      position: relative;
      z-index: 1;
    }
    .app-row {
      display: flex;
      align-items: center;
      gap: 14px;
      margin-bottom: 18px;
      position: relative;
      z-index: 1;
    }
    .app-icon {
      width: 54px;
      height: 54px;
      border-radius: 18px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(180deg, rgba(255,255,255,0.96), var(--oshi-soft));
      border: 1px solid var(--oshi-line);
      color: var(--oshi-deep);
      font-size: 22px;
    }
    .app-name {
      font-size: 28px;
      font-weight: 800;
      letter-spacing: -0.05em;
      line-height: 1.02;
    }
    .app-id {
      color: #8a8faa;
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      margin-top: 4px;
    }
    p {
      margin: 0 0 20px;
      color: #6f7394;
      line-height: 1.8;
      position: relative;
      z-index: 1;
    }
    .section-label {
      margin: 0 0 10px;
      color: var(--oshi-deep);
      font-size: 12px;
      font-weight: 700;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      position: relative;
      z-index: 1;
    }
    ul {
      list-style: none;
      padding: 0;
      margin: 0 0 20px;
      display: grid;
      gap: 8px;
      position: relative;
      z-index: 1;
    }
    li {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 12px 14px;
      border-radius: 18px;
      background: rgba(255,255,255,0.72);
      border: 1px solid rgba(29,32,64,0.08);
      color: #4f5477;
      font-size: 14px;
    }
    li::before {
      content: "◉";
      color: var(--oshi-deep);
      font-size: 9px;
    }
    .divider {
      border: none;
      border-top: 1px solid rgba(29,32,64,0.08);
      margin: 6px 0 24px;
      position: relative;
      z-index: 1;
    }
    .actions {
      display: flex;
      gap: 12px;
      position: relative;
      z-index: 1;
    }
    button {
      flex: 1;
      border-radius: 18px;
      padding: 15px 18px;
      font-size: 15px;
      font-weight: 700;
      cursor: pointer;
      transition: transform 0.14s ease, box-shadow 0.14s ease, opacity 0.14s ease;
    }
    button:active { transform: scale(0.99); }
    .allow {
      border: none;
      background: linear-gradient(180deg, var(--oshi), rgba(255,255,255,0.68));
      color: #1d2040;
      box-shadow: 0 18px 36px rgba(59,61,109,0.14);
    }
    .allow:hover { opacity: 0.86; }
    .deny {
      background: rgba(255,255,255,0.72);
      color: #5d6382;
      border: 1px solid rgba(29,32,64,0.08);
    }
    .deny:hover { box-shadow: 0 14px 28px rgba(59,61,109,0.1); }
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
      main { padding: 22px; border-radius: 26px; }
      .app-name { font-size: 24px; }
      .actions { flex-direction: column; }
      #oshi-toggle { width: 52px; height: 52px; border-radius: 18px; }
      #oshi-swatches { width: 168px; }
    }
  </style>
  <script nonce="{{ .Nonce }}">
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
      fetch('/v1/auth/theme',{
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
  <main>
    <div class="eyebrow">✦ OAuth Consent</div>
    <div class="app-row">
      <div class="app-icon">◈</div>
      <div>
        <div class="app-name">{{ .DisplayName }}</div>
        <div class="app-id">{{ .ClientID }}</div>
      </div>
    </div>
    <p>このアプリが共有アカウントへのアクセスを求めています。スコープと利用先を確認して、連携を許可するか選んでください。</p>
    <div class="section-label">Requested scopes</div>
    <ul>{{ range .RequestedScope }}<li>{{ . }}</li>{{ end }}</ul>
    {{ if .RequestedAudience }}
    <div class="section-label">Requested audiences</div>
    <ul>{{ range .RequestedAudience }}<li>{{ . }}</li>{{ end }}</ul>
    {{ end }}
    <hr class="divider">
    <form method="post" action="/v1/auth/consent">
      <input type="hidden" name="consent_challenge" value="{{ .Challenge }}">
      <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
      <div class="actions">
        <button class="allow" type="submit" name="action" value="accept">アクセスを許可</button>
        <button class="deny" type="submit" name="action" value="deny">キャンセル</button>
      </div>
    </form>
  </main>
  <div id="oshi-picker">
    <div id="oshi-swatches" aria-label="推しメンカラーパレット"></div>
    <button id="oshi-toggle" type="button" title="推しメンカラー">✦</button>
  </div>
  <script nonce="{{ .Nonce }}">
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
	displayName := prompt.ClientName
	if strings.TrimSpace(displayName) == "" {
		displayName = prompt.ClientID
	}
	csrfToken, err := newCSRFToken()
	if err != nil {
		return err
	}
	nonce, err := newCSRFToken()
	if err != nil {
		return err
	}
	setConsentCSRFCookie(w, csrfToken, secureCookies, cookieDomain)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; style-src 'nonce-"+nonce+"'; script-src 'nonce-"+nonce+"'")
	view := struct {
		Challenge         string
		ClientID          string
		DisplayName       string
		OshiColor         string
		CSRFToken         string
		Nonce             string
		RequestedScope    []string
		RequestedAudience []string
	}{
		Challenge:         prompt.Challenge,
		ClientID:          prompt.ClientID,
		DisplayName:       displayName,
		OshiColor:         prompt.OshiColor,
		CSRFToken:         csrfToken,
		Nonce:             nonce,
		RequestedScope:    prompt.RequestedScope,
		RequestedAudience: prompt.RequestedAccessTokenAudience,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	return template.Must(template.New("consent").Parse(tpl)).Execute(w, view)
}

func newCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func validateConsentCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(consentCSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := strings.TrimSpace(r.Form.Get("csrf_token"))
	if formToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) == 1
}

func setConsentCSRFCookie(w http.ResponseWriter, token string, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     consentCSRFCookieName,
		Value:    token,
		Path:     "/v1/auth/consent",
		Domain:   domain,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   600, // 10 minutes — matches Hydra consent flow TTL
	})
}

func clearConsentCSRFCookie(w http.ResponseWriter, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     consentCSRFCookieName,
		Value:    "",
		Path:     "/v1/auth/consent",
		Domain:   domain,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   -1,
	})
}
