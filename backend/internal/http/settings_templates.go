package http

import "html/template"

var settingsTpl = template.Must(template.New("settings").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>セキュリティ設定 | OshiLink</title>
  <style>
    :root{
      --oshi:{{.OshiColor}};
      --oshi-weak:#fce7f3;
      --oshi-soft:#fbcfe8;
      --bg:#fdf8fc;
      --surface:#ffffff;
      --surface-2:#fef9fd;
      --text:#2a1520;
      --muted:#9d748f;
      --border:#f0d0e8;
      --radius-lg:32px;
      --radius-md:22px;
      --radius-sm:18px;
      --danger:#e85d75;
      --success:#10b981;
    }
    @media (prefers-color-scheme: dark){
      :root{
        --bg:#1a1018; --surface:#251820; --surface-2:#2e1f28; --text:#fff0f8; --muted:#c8a8bd; --border:#4a2e40;
        --oshi-weak:#3d1a2a; --oshi-soft:#5a2a40;
      }
    }
    *,*::before,*::after{box-sizing:border-box}
    html,body{margin:0;padding:0}
    body{min-height:100vh;font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Hiragino Sans","Yu Gothic","Noto Sans JP",sans-serif;color:var(--text);background:var(--bg);letter-spacing:.01em}
    button,input,select{font:inherit;color:inherit}
    button{cursor:pointer}
    a{color:var(--oshi);text-decoration:none}

    .topbar{position:sticky;top:0;z-index:20;display:flex;align-items:center;justify-content:space-between;gap:16px;padding:14px 28px;background:color-mix(in srgb,var(--surface) 88%,transparent);backdrop-filter:saturate(160%) blur(14px);-webkit-backdrop-filter:saturate(160%) blur(14px);border-bottom:2px solid var(--border)}
    .brand{display:flex;align-items:center;gap:12px;text-decoration:none;color:var(--text)}
    .brand-mark{width:42px;height:42px;border-radius:14px;background:var(--oshi);color:#fff;display:grid;place-items:center;font-weight:900;font-size:20px;letter-spacing:-.02em;border:2px solid color-mix(in srgb,var(--oshi) 35%,transparent)}
    .brand-text strong{display:block;font-size:16px;line-height:1.1;letter-spacing:-.01em}
    .brand-text span{font-size:11px;color:var(--muted);font-weight:700;text-transform:uppercase;letter-spacing:.12em}
    .top-actions{display:flex;align-items:center;gap:12px}
    .user-chip{display:flex;align-items:center;gap:10px;padding:6px 12px 6px 6px;border-radius:999px;background:var(--surface-2);border:2px solid var(--border);font-size:13px;color:var(--text);max-width:280px}
    .user-chip .dot{width:32px;height:32px;border-radius:50%;background:var(--oshi);color:#fff;display:grid;place-items:center;font-weight:900;font-size:13px}
    .user-chip .email{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:200px;color:var(--muted)}
    .icon-btn{width:38px;height:38px;border-radius:12px;border:2px solid var(--border);background:var(--surface);display:grid;place-items:center;font-size:16px;color:var(--muted);transition:background .15s,border-color .15s}
    .icon-btn:hover{background:var(--oshi-weak);border-color:var(--oshi);color:var(--oshi)}

    .page-hero{padding:40px 28px 28px;background:var(--oshi-weak);border-bottom:2px solid var(--border);position:relative;overflow:hidden}
    .page-hero::after{content:"";position:absolute;right:-60px;bottom:-60px;width:180px;height:180px;border-radius:50%;border:3px solid var(--oshi-soft);opacity:.5;pointer-events:none}
    .hero-inner{max-width:720px;margin:0 auto;position:relative;z-index:1}
    .back-link{display:inline-flex;align-items:center;gap:6px;color:var(--muted);font-size:13px;font-weight:600;margin-bottom:16px;transition:color .15s}
    .back-link:hover{color:var(--oshi)}
    .hero-row{display:flex;align-items:center;gap:16px}
    .hero-avatar{width:56px;height:56px;border-radius:18px;background:var(--oshi);color:#fff;display:grid;place-items:center;font-weight:900;font-size:20px;flex-shrink:0;border:2px solid color-mix(in srgb,var(--oshi) 35%,transparent)}
    .hero-title{font-size:26px;font-weight:900;letter-spacing:-.02em;line-height:1.1}
    .hero-sub{font-size:13px;color:var(--muted);margin-top:4px}

    .content{max-width:720px;margin:0 auto;padding:28px 28px 64px}

    .flash{padding:14px 18px;border-radius:var(--radius-md);font-size:14px;font-weight:600;margin-bottom:20px}
    .flash-error{background:color-mix(in srgb,var(--danger) 10%,var(--surface));color:var(--danger);border:2px solid color-mix(in srgb,var(--danger) 25%,transparent)}
    .flash-info{background:var(--oshi-weak);color:var(--text);border:2px solid var(--oshi-soft)}

    .section{background:var(--surface);border-radius:var(--radius-lg);border:2px solid var(--border);padding:28px;margin-bottom:20px}
    .section-title{font-size:11px;font-weight:800;letter-spacing:.12em;text-transform:uppercase;color:var(--muted);margin-bottom:20px}

    .field{margin-bottom:18px}
    .field label{display:block;font-size:12px;font-weight:700;color:var(--muted);margin-bottom:7px;letter-spacing:.04em;text-transform:uppercase}
    .field input{width:100%;padding:12px 16px;border:2px solid var(--border);border-radius:var(--radius-sm);font-size:15px;color:var(--text);background:var(--surface-2);outline:none;transition:border-color .15s}
    .field input:focus{border-color:var(--oshi);background:var(--surface)}
    .field input:disabled{opacity:.5;cursor:not-allowed}
    .field-error{font-size:12px;color:var(--danger);margin-top:6px;font-weight:600}

    .btn-save{display:inline-flex;align-items:center;gap:8px;padding:12px 24px;border-radius:999px;font-size:14px;font-weight:800;cursor:pointer;border:2px solid var(--oshi);background:var(--oshi);color:#fff;transition:opacity .15s,transform .1s;margin-top:8px}
    .btn-save:hover{opacity:.88;transform:translateY(-1px)}
    .btn-save:active{transform:scale(.97)}

    .color-picker{display:flex;gap:8px;align-items:center}
    .color-swatch{width:28px;height:28px;border-radius:50%;border:2px solid transparent;cursor:pointer;transition:transform .15s,border-color .15s;padding:0}
    .color-swatch:hover,.color-swatch.active{transform:scale(1.2);border-color:var(--text)}

    @media(max-width:600px){
      .topbar{padding:12px 16px}
      .page-hero{padding:28px 16px 20px}
      .content{padding:20px 16px 48px}
      .section{padding:20px}
    }
  </style>
</head>
<body>

<header class="topbar">
  <a href="/account" class="brand">
    <div class="brand-mark">O</div>
    <div class="brand-text"><strong>OshiLink</strong><span>Account</span></div>
  </a>
  <div class="top-actions">
    <div class="user-chip">
      <div class="dot">{{ .Initials }}</div>
      <span class="email">{{ if .Email }}{{ .Email }}{{ else }}{{ .Phone }}{{ end }}</span>
    </div>
    <div class="color-picker" title="推し色を選ぶ">
      <button class="color-swatch" data-color="#f472b6" style="background:#f472b6" title="ピンク"></button>
      <button class="color-swatch" data-color="#ff8a3d" style="background:#ff8a3d" title="オレンジ"></button>
      <button class="color-swatch" data-color="#a78bfa" style="background:#a78bfa" title="ラベンダー"></button>
      <button class="color-swatch" data-color="#60a5fa" style="background:#60a5fa" title="スカイブルー"></button>
      <button class="color-swatch" data-color="#34d399" style="background:#34d399" title="ミント"></button>
    </div>
  </div>
</header>

<div class="page-hero">
  <div class="hero-inner">
    <a href="/account" class="back-link">← 推し活アカウントへ戻る</a>
    <div class="hero-row">
      <div class="hero-avatar">{{ .Initials }}</div>
      <div>
        <div class="hero-title">セキュリティ設定</div>
        <div class="hero-sub">大切なアカウントと推し活の履歴を安全に守るための設定です。{{ if .DisplayName }}{{ .DisplayName }} · {{ end }}{{ if .Email }}{{ .Email }}{{ else }}{{ .Phone }}{{ end }}</div>
      </div>
    </div>
  </div>
</div>

<div class="content">
  {{- range .Messages }}
  <div class="flash {{ if eq .Type "error" }}flash-error{{ else }}flash-info{{ end }}">{{ .Text }}</div>
  {{- end }}

  {{- range .Sections }}
  <div class="section">
    <div class="section-title">{{ .Title }}</div>
    <form method="{{ $.Method }}" action="{{ $.Action }}">
      {{- range .Hidden }}
      <input type="hidden" name="{{ .Name }}" value="{{ .Value }}">
      {{- end }}
      {{- range .Inputs }}
      <div class="field">
        {{- if .Label }}<label>{{ .Label }}</label>{{- end }}
        <input
          type="{{ .InputType }}"
          name="{{ .Name }}"
          value="{{ .Value }}"
          {{- if .Required }} required{{- end }}
          {{- if .Disabled }} disabled{{- end }}
        >
        {{- range .Messages }}
        <div class="field-error">{{ .Text }}</div>
        {{- end }}
      </div>
      {{- end }}
      {{- if .Submit }}
      <button class="btn-save" type="submit" name="{{ .Submit.Name }}" value="{{ .Submit.Value }}">
        {{ if .Submit.Label }}{{ .Submit.Label }}{{ else }}保存{{ end }}
      </button>
      {{- end }}
    </form>
  </div>
  {{- end }}
</div>

<script nonce="{{.Nonce}}">
(function(){
  const oshi = localStorage.getItem('oshi_color');
  if(oshi){
    document.documentElement.style.setProperty('--oshi', oshi);
    document.querySelectorAll('.color-swatch').forEach(s=>{
      s.classList.toggle('active', s.dataset.color===oshi);
    });
  }
  document.querySelectorAll('.color-swatch').forEach(s=>{
    s.addEventListener('click',()=>setOshi(s));
  });
})();
function setOshi(el){
  const c = el.dataset.color;
  document.documentElement.style.setProperty('--oshi', c);
  localStorage.setItem('oshi_color', c);
  document.querySelectorAll('.color-swatch').forEach(s=>s.classList.toggle('active', s===el));
  fetch('/v1/auth/theme',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({color:c})}).catch(()=>{});
}
</script>
</body>
</html>`))
