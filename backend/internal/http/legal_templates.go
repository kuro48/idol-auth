package http

import "html/template"

var legalTpl = template.Must(template.New("legal").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} | OshiLink</title>
  <style>
    :root{
      --oshi:#f472b6;
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
    }
    @media (prefers-color-scheme: dark){
      :root{
        --bg:#1a1018; --surface:#251820; --surface-2:#2e1f28; --text:#fff0f8;
        --muted:#c8a8bd; --border:#4a2e40; --oshi-weak:#3d1a2a; --oshi-soft:#5a2a40;
      }
    }
    *,*::before,*::after{box-sizing:border-box}
    html,body{margin:0;padding:0}
    body{min-height:100vh;font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Hiragino Sans","Yu Gothic","Noto Sans JP",sans-serif;color:var(--text);background:var(--bg);letter-spacing:.01em}
    a{color:var(--oshi);text-decoration:none}
    a:hover{text-decoration:underline}

    .topbar{position:sticky;top:0;z-index:20;display:flex;align-items:center;justify-content:space-between;gap:16px;padding:14px 28px;background:color-mix(in srgb,var(--surface) 88%,transparent);backdrop-filter:saturate(160%) blur(14px);-webkit-backdrop-filter:saturate(160%) blur(14px);border-bottom:2px solid var(--border)}
    .brand{display:flex;align-items:center;gap:12px;text-decoration:none;color:var(--text)}
    .brand-mark{width:42px;height:42px;border-radius:14px;background:var(--oshi);color:#fff;display:grid;place-items:center;font-weight:900;font-size:20px;letter-spacing:-.02em;border:2px solid color-mix(in srgb,var(--oshi) 35%,transparent)}
    .brand-text strong{display:block;font-size:16px;line-height:1.1;letter-spacing:-.01em}
    .brand-text span{font-size:11px;color:var(--muted);font-weight:700;text-transform:uppercase;letter-spacing:.12em}
    .top-login{display:inline-flex;align-items:center;gap:6px;padding:8px 18px;border-radius:999px;border:2px solid var(--oshi);background:var(--oshi);color:#fff;font-size:13px;font-weight:700;text-decoration:none;transition:opacity .15s}
    .top-login:hover{opacity:.85;text-decoration:none}

    .page-hero{padding:40px 28px 28px;background:var(--oshi-weak);border-bottom:2px solid var(--border);position:relative;overflow:hidden}
    .page-hero::after{content:"";position:absolute;right:-60px;bottom:-60px;width:180px;height:180px;border-radius:50%;border:3px solid var(--oshi-soft);opacity:.5;pointer-events:none}
    .hero-inner{max-width:720px;margin:0 auto;position:relative;z-index:1}
    .back-link{display:inline-flex;align-items:center;gap:6px;color:var(--muted);font-size:13px;font-weight:600;margin-bottom:14px;transition:color .15s}
    .back-link:hover{color:var(--oshi);text-decoration:none}
    .eyebrow{font-size:11px;font-weight:800;letter-spacing:.14em;text-transform:uppercase;color:var(--oshi);margin-bottom:8px}
    .hero-title{font-size:28px;font-weight:900;letter-spacing:-.02em;line-height:1.1}
    .updated{font-size:12px;color:var(--muted);margin-top:6px}

    .content{max-width:720px;margin:0 auto;padding:36px 28px 80px}

    .legal-doc h2{font-size:16px;font-weight:800;margin:2em 0 .6em;padding-bottom:.4em;border-bottom:2px solid var(--border)}
    .legal-doc h2:first-child{margin-top:0}
    .legal-doc p{font-size:14px;line-height:1.8;color:var(--text);margin:.6em 0}
    .legal-doc ul,.legal-doc ol{padding-left:1.5em;margin:.6em 0}
    .legal-doc li{font-size:14px;line-height:1.8;margin:.2em 0}
    .legal-doc ol li{list-style:decimal}
    .legal-doc a{color:var(--oshi)}

    .contact-card{background:var(--surface-2);border:2px solid var(--border);border-radius:var(--radius-md);padding:20px 24px;margin:16px 0 24px;display:flex;flex-direction:column;gap:12px}
    .contact-row{display:flex;gap:16px;align-items:baseline;flex-wrap:wrap}
    .contact-label{font-size:12px;font-weight:700;color:var(--muted);text-transform:uppercase;letter-spacing:.06em;min-width:140px;flex-shrink:0}
    .contact-value{font-size:14px;color:var(--text);font-weight:500}

    .legal-footer{margin-top:48px;padding-top:24px;border-top:2px solid var(--border);display:flex;flex-wrap:wrap;gap:8px 24px}
    .legal-footer a{font-size:12px;color:var(--muted);transition:color .15s}
    .legal-footer a:hover{color:var(--oshi);text-decoration:none}

    @media(max-width:600px){
      .topbar{padding:12px 16px}
      .page-hero{padding:28px 16px 20px}
      .content{padding:24px 16px 60px}
      .contact-label{min-width:100px}
    }
  </style>
</head>
<body>

<header class="topbar">
  <a href="/" class="brand">
    <div class="brand-mark">O</div>
    <div class="brand-text"><strong>OshiLink</strong><span>{{.Eyebrow}}</span></div>
  </a>
  <a href="/login" class="top-login">ログイン</a>
</header>

<div class="page-hero">
  <div class="hero-inner">
    <a href="/" class="back-link">← トップへ戻る</a>
    <div class="eyebrow">{{.Eyebrow}}</div>
    <div class="hero-title">{{.Title}}</div>
    <div class="updated">最終更新日: {{.UpdatedOn}}</div>
  </div>
</div>

<div class="content">
  <article class="legal-doc">
    {{.Body}}
  </article>

  <nav class="legal-footer" aria-label="関連ページ">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
    <a href="/legal/contact">問い合わせ先</a>
    <a href="/legal/incident">障害時連絡先</a>
  </nav>
</div>

</body>
</html>`))
