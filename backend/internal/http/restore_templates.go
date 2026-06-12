package http

import "html/template"

var restoreAccountTpl = template.Must(template.New("restore-account").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>アカウント復活 | OshiLink</title>
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
      --warning:#e9a23b;
    }
    @media (prefers-color-scheme: dark){
      :root{
        --bg:#1a1018; --surface:#251820; --surface-2:#2e1f28; --text:#fff0f8; --muted:#c8a8bd; --border:#4a2e40;
        --oshi-weak:#3d1a2a; --oshi-soft:#5a2a40;
      }
    }
    *,*::before,*::after{box-sizing:border-box}
    html,body{margin:0;padding:0}
    body{
      min-height:100vh;
      font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Hiragino Sans","Yu Gothic","Noto Sans JP",sans-serif;
      color:var(--text);
      background:var(--bg);
      letter-spacing:.01em;
      display:flex; flex-direction:column;
    }
    button,input,select{font:inherit;color:inherit}
    button{cursor:pointer}
    a{color:var(--oshi);text-decoration:none}
    a:hover{opacity:.8}

    .topbar{
      position:sticky; top:0; z-index:20;
      display:flex; align-items:center; gap:16px; padding:14px 28px;
      background:color-mix(in srgb, var(--surface) 88%, transparent);
      backdrop-filter:saturate(160%) blur(14px);
      -webkit-backdrop-filter:saturate(160%) blur(14px);
      border-bottom:2px solid var(--border);
    }
    .brand{display:flex; align-items:center; gap:12px}
    .brand-mark{
      width:42px; height:42px; border-radius:14px;
      background:var(--oshi); color:#fff;
      display:grid; place-items:center;
      font-weight:900; font-size:20px; letter-spacing:-.02em;
      border:2px solid color-mix(in srgb,var(--oshi) 35%,transparent);
    }
    .brand-text strong{display:block; font-size:16px; line-height:1.1; letter-spacing:-.01em}
    .brand-text span{font-size:11px; color:var(--muted); font-weight:700; text-transform:uppercase; letter-spacing:.12em}

    .shell{
      flex:1; display:flex; align-items:center; justify-content:center;
      padding:40px 24px;
    }
    .card{
      width:100%; max-width:480px;
      background:var(--surface);
      border:2px solid var(--border);
      border-radius:var(--radius-lg);
      padding:36px;
      display:flex; flex-direction:column; gap:28px;
    }
    .icon-wrap{
      width:72px; height:72px; border-radius:24px;
      background:var(--oshi-weak); color:var(--oshi);
      display:grid; place-items:center;
      font-size:32px;
      border:2px solid var(--oshi-soft);
    }
    .card-head{display:flex; flex-direction:column; gap:10px}
    .card-head h1{margin:0; font-size:26px; font-weight:900; letter-spacing:-.02em; line-height:1.1}
    .card-head p{margin:0; font-size:15px; color:var(--muted); line-height:1.7}

    .countdown{
      display:flex; align-items:center; gap:14px;
      padding:18px; border-radius:var(--radius-md);
      background:var(--oshi-weak); border:2px solid var(--oshi-soft);
    }
    .countdown .days{font-size:48px; font-weight:900; color:var(--oshi); line-height:1; letter-spacing:-.03em}
    .countdown .label{font-size:13px; color:var(--muted); line-height:1.5; font-weight:600}
    .countdown .label strong{display:block; color:var(--text); font-size:16px; font-weight:800}

    .warning-box{
      padding:14px 16px; border-radius:var(--radius-sm);
      background:rgba(233,162,59,.1); border:2px solid rgba(233,162,59,.3);
      font-size:13px; color:var(--text); line-height:1.7;
    }
    .warning-box strong{color:var(--warning)}

    .btn-restore{
      width:100%; padding:16px;
      background:var(--oshi); color:#fff;
      border:none; border-radius:999px;
      font-size:16px; font-weight:800; letter-spacing:-.01em;
      transition:opacity 160ms ease, transform 160ms ease;
    }
    .btn-restore:hover{opacity:.88; transform:translateY(-1px)}
    .btn-restore:active{transform:translateY(0)}

    .action-footer{display:flex; flex-direction:column; gap:10px}
    .logout-link{text-align:center; font-size:13px; color:var(--muted)}
    .logout-link a{color:var(--muted); font-weight:600}
    .logout-link a:hover{color:var(--text)}

    .site-footer{padding:24px 28px 40px;display:flex;flex-wrap:wrap;gap:8px 24px;border-top:2px solid var(--border);margin-top:auto}
    .site-footer a{font-size:12px;color:var(--muted);transition:color .15s}
    .site-footer a:hover{color:var(--oshi)}

    @media(max-width:640px){
      .card{padding:24px; border-radius:var(--radius-md)}
    }
  </style>
</head>
<body>
  <header class="topbar">
    <div class="brand">
      <div class="brand-mark">★</div>
      <div class="brand-text">
        <strong>OshiLink</strong>
        <span>ACCOUNT</span>
      </div>
    </div>
  </header>

  <main class="shell">
    <div class="card">
      <div class="icon-wrap">🌸</div>
      <div class="card-head">
        <h1>アカウントを復活させますか？</h1>
        <p>このアカウントは削除申請済みです。期限内であれば、今すぐ復活できます。</p>
      </div>

      <div class="countdown">
        <div class="days">{{.DaysLeft}}</div>
        <div class="label">
          <strong>日後に完全削除</strong>
          削除予定日: {{.ScheduledFor.Format "2006年01月02日"}}
        </div>
      </div>

      <div class="warning-box">
        <strong>⚠ 注意：</strong>
        期限を過ぎるとアカウントのデータはすべて削除され、復活できなくなります。復活後は通常通りご利用いただけます。
      </div>

      <div class="action-footer">
        <form method="POST" action="/account/restore">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <button type="submit" class="btn-restore">🌸 アカウントを復活させる</button>
        </form>
        <div class="logout-link">
          <a href="{{.LogoutURL}}">このままログアウトする</a>
        </div>
      </div>
    </div>
  </main>

  <footer class="site-footer">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
  </footer>
</body>
</html>`))
