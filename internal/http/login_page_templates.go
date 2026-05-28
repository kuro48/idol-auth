package http

import "html/template"

var loginPageTpl = template.Must(template.New("login-page").Parse(authPageSharedCSS + loginPageBody))
var registrationPageTpl = template.Must(template.New("registration-page").Parse(authPageSharedCSS + registrationPageBody))

const authPageSharedCSS = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>OshiLink — 推し活アカウント</title>
  <style>
    :root {
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
    * { box-sizing: border-box; }
    html, body { height: 100%; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: var(--text);
      background: var(--bg);
      letter-spacing: 0.01em;
      -webkit-font-smoothing: antialiased;
    }
    a { color: inherit; text-decoration: none; }
    button, input, select { font: inherit; }
    button { cursor: pointer; }

    .stage {
      display: grid;
      grid-template-columns: 0.95fr 1.05fr;
      min-height: 100vh;
      gap: 0;
    }

    .hero-card {
      position: relative;
      padding: clamp(28px, 4vw, 64px);
      background: var(--oshi-weak);
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      gap: 32px;
      overflow: hidden;
      border-right: 2px solid var(--border);
    }
    .hero-card::after {
      content: "";
      position: absolute;
      right: -76px;
      bottom: -76px;
      width: 240px;
      height: 240px;
      border-radius: 50%;
      border: 3px solid var(--oshi-soft);
      opacity: 0.55;
      pointer-events: none;
    }
    .hero-card::before {
      content: "";
      position: absolute;
      left: -100px;
      top: -100px;
      width: 280px;
      height: 280px;
      border-radius: 50%;
      border: 3px solid color-mix(in srgb, var(--oshi) 30%, transparent);
      opacity: 0.45;
      pointer-events: none;
    }

    .brand-row {
      display: flex;
      align-items: center;
      gap: 14px;
      position: relative;
      z-index: 1;
    }
    .brand-mark {
      width: 52px;
      height: 52px;
      border-radius: 18px;
      background: var(--oshi);
      color: #fff;
      display: grid;
      place-items: center;
      font-size: 24px;
      font-weight: 900;
      border: 2px solid color-mix(in srgb, var(--oshi) 40%, transparent);
    }
    .brand-text strong {
      display: block;
      font-size: 22px;
      font-weight: 900;
      letter-spacing: -0.01em;
    }
    .brand-text span {
      display: block;
      color: var(--muted);
      font-size: 12px;
      font-weight: 700;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      margin-top: 2px;
    }

    .hero-body {
      position: relative;
      z-index: 1;
      display: flex;
      flex-direction: column;
      gap: 22px;
      max-width: 480px;
    }
    .badge-row { display: flex; flex-wrap: wrap; gap: 8px; }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 8px 14px;
      border-radius: 999px;
      background: var(--surface);
      color: var(--oshi);
      border: 2px solid var(--oshi-soft);
      font-size: 12px;
      font-weight: 800;
      letter-spacing: 0.04em;
      transform: rotate(-1.5deg);
      transition: transform 180ms ease;
    }
    .badge:hover { transform: rotate(0deg); }
    .badge-violet {
      background: #f3e8ff;
      color: #7c3aed;
      border-color: #ddd6fe;
      transform: rotate(1.2deg);
    }
    .badge-mint {
      background: #d1fae5;
      color: #059669;
      border-color: #a7f3d0;
      transform: rotate(-0.8deg);
    }
    @media (prefers-color-scheme: dark) {
      .badge-violet { background: #2d1a4a; color: #c084fc; border-color: #5b3a8a; }
      .badge-mint { background: #0d2a20; color: #34d399; border-color: #065f46; }
    }

    .hero-title {
      margin: 0;
      font-size: clamp(34px, 5.4vw, 58px);
      line-height: 1.04;
      letter-spacing: -0.04em;
      font-weight: 1000;
    }
    .hero-title span { color: var(--oshi); }
    .hero-lede {
      margin: 0;
      color: var(--muted);
      font-size: clamp(15px, 1.1vw, 17px);
      line-height: 1.85;
      max-width: 44ch;
    }
    .tag-row { display: flex; flex-wrap: wrap; gap: 8px; }
    .tag {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 12px;
      border-radius: 999px;
      background: var(--surface);
      border: 2px solid var(--border);
      color: var(--text);
      font-size: 12px;
      font-weight: 700;
      transition: border-color 180ms ease;
    }
    .tag:hover { border-color: var(--oshi); }
    .tag::before { content: ""; width: 6px; height: 6px; border-radius: 50%; background: var(--oshi); flex-shrink: 0; }

    .phone-wrap { position: relative; z-index: 1; display: flex; justify-content: flex-end; margin-top: auto; }
    .phone-preview {
      width: 280px;
      padding: 14px 14px 22px;
      background: #2a1e28;
      border-radius: 36px;
      border: 2px solid rgba(255,255,255,.07);
      transform: rotate(-3deg);
    }
    .phone-screen {
      background: #fffaf7; border-radius: 26px; padding: 18px 16px;
      color: #2a1520; min-height: 320px; display: flex; flex-direction: column; gap: 14px;
    }
    .phone-handle { width: 60px; height: 5px; border-radius: 999px; background: rgba(0, 0, 0, 0.1); margin: 0 auto 8px; }
    .ticket {
      padding: 14px; border-radius: 18px;
      background: var(--oshi);
      color: #fff;
      border: 2px solid color-mix(in srgb, #fff 22%, transparent);
      display: flex; flex-direction: column; gap: 6px;
    }
    .ticket-eyebrow { font-size: 10px; letter-spacing: 0.18em; text-transform: uppercase; opacity: 0.85; font-weight: 800; }
    .ticket-title { font-size: 18px; font-weight: 900; letter-spacing: -0.02em; }
    .ticket-meta { display: flex; justify-content: space-between; font-size: 11px; font-weight: 700; opacity: 0.85; }
    .mini-row { display: flex; align-items: center; gap: 10px; font-size: 12px; color: #7a5e70; }
    .mini-bar { flex: 1; height: 8px; border-radius: 999px; background: #f0dce8; overflow: hidden; position: relative; }
    .mini-bar span { position: absolute; inset: 0 auto 0 0; background: var(--oshi); border-radius: 999px; }
    .mini-label { font-weight: 800; color: #2a1520; min-width: 70px; }
    .mini-pct { font-weight: 800; color: var(--oshi); font-variant-numeric: tabular-nums; }

    .auth-shell {
      position: relative;
      padding: clamp(24px, 3vw, 48px);
      display: flex;
      flex-direction: column;
      gap: 28px;
      background: var(--bg);
    }
    .auth-top { display: flex; justify-content: flex-end; align-items: center; }
    .auth-alt { color: var(--muted); font-size: 13px; font-weight: 600; }
    .auth-alt a {
      color: var(--oshi); font-weight: 800; text-decoration: none;
      border-bottom: 2px solid var(--oshi-soft);
    }

    .auth-card {
      position: relative;
      background: var(--surface);
      border: 2px solid var(--border);
      border-radius: var(--radius-lg);
      padding: clamp(24px, 2.6vw, 32px);
      display: flex;
      flex-direction: column;
      gap: 24px;
      max-width: 520px;
      width: 100%;
      margin: auto 0;
    }
    .auth-head { display: flex; flex-direction: column; gap: 8px; }
    .auth-eyebrow { color: var(--oshi); font-size: 12px; font-weight: 800; letter-spacing: 0.18em; text-transform: uppercase; }
    .auth-title { margin: 0; font-size: clamp(26px, 2.6vw, 32px); letter-spacing: -0.03em; font-weight: 900; line-height: 1.15; }
    .auth-sub { margin: 0; color: var(--muted); font-size: 14px; line-height: 1.7; }

    .field { display: flex; flex-direction: column; gap: 6px; }
    .field label { font-size: 12px; font-weight: 800; color: var(--text); letter-spacing: 0.02em; }
    .field input {
      width: 100%; padding: 14px 16px; border-radius: var(--radius-sm);
      border: 2px solid var(--border); background: var(--surface-2);
      color: var(--text); font-size: 14px;
      transition: border-color 0.18s ease, background 0.18s ease;
    }
    .field input::placeholder { color: color-mix(in srgb, var(--muted) 85%, transparent); }
    .field input:focus { outline: none; border-color: var(--oshi); background: var(--surface); }
    .field-hint { color: var(--muted); font-size: 12px; margin-top: 2px; }

    .oshi-pick {
      display: flex; gap: 10px; padding: 12px; background: var(--surface-2);
      border: 2px solid var(--border); border-radius: var(--radius-sm); align-items: center; flex-wrap: wrap;
    }
    .oshi-pick-label { font-size: 12px; font-weight: 800; color: var(--muted); letter-spacing: 0.04em; margin-right: 4px; }
    .oshi-pick-dot {
      width: 26px; height: 26px; border-radius: 999px; border: 2px solid transparent;
      cursor: pointer; transition: transform 0.18s ease, border-color 0.18s ease; padding: 0;
    }
    .oshi-pick-dot:hover { transform: scale(1.08); }
    .oshi-pick-dot.active { border-color: var(--text); transform: scale(1.12); }

    .btn {
      display: inline-flex; align-items: center; justify-content: center; gap: 8px;
      width: 100%; padding: 15px 18px; border-radius: 999px; border: 2px solid transparent;
      font-size: 15px; font-weight: 900; letter-spacing: 0.01em; text-decoration: none;
      transition: transform 0.18s ease, background 0.18s ease;
      cursor: pointer;
    }
    .btn:hover { transform: translateY(-1px); }
    .btn:active { transform: translateY(0); }
    .btn-primary { background: var(--oshi); color: #fff; border-color: var(--oshi); }
    .btn-primary:hover { background: color-mix(in srgb, var(--oshi) 88%, #000 12%); }

    .auth-note { color: var(--muted); font-size: 12px; line-height: 1.7; text-align: center; }
    .auth-note a { color: var(--text); font-weight: 800; border-bottom: 1px dashed var(--border); }

    .site-footer {
      display: flex; flex-wrap: wrap; justify-content: center;
      gap: 8px 20px; padding: 24px 28px 32px; border-top: 2px solid var(--border); margin-top: 8px;
    }
    .site-footer a { font-size: 12px; color: var(--muted); transition: color .15s; }
    .site-footer a:hover { color: var(--oshi); }

    @media (max-width: 980px) {
      .stage { grid-template-columns: 1fr; }
      .hero-card { border-right: none; border-bottom: 2px solid var(--border); }
      .auth-shell { padding: 24px; }
      .auth-card { margin: 0 auto; }
      .phone-wrap { justify-content: center; }
    }
    @media (max-width: 560px) {
      .hero-card { padding: 24px; gap: 24px; }
      .phone-preview { width: 240px; transform: rotate(-2deg); }
      .auth-card { padding: 22px; border-radius: 22px; }
    }
  </style>
</head>
<body>`

const loginPageBody = `
  <main class="stage">
    <section class="hero-card" aria-labelledby="hero-heading">
      <div class="brand-row">
        <div class="brand-mark" aria-hidden="true">★</div>
        <div class="brand-text">
          <strong>OshiLink</strong>
          <span>推し活 Account</span>
        </div>
      </div>
      <div class="hero-body">
        <div class="badge-row">
          <span class="badge">🎫 推しメンカラー対応</span>
          <span class="badge badge-violet">⭐ Fan ID</span>
          <span class="badge badge-mint">🔒 安全ログイン</span>
        </div>
        <h1 class="hero-title" id="hero-heading">推し活の入口を<br /><span>ひとつに。</span></h1>
        <p class="hero-lede">
          推し活サービスをひとつの共通アカウントで横断。チケット、グッズ、ファンクラブ、配信——サービスごとのログインを卒業して、推しに使う時間を取り戻そう。
        </p>
        <div class="tag-row">
          <span class="tag">共通ログイン</span>
          <span class="tag">プロフィール連携</span>
          <span class="tag">サービス横断</span>
        </div>
      </div>
      <div class="phone-wrap" aria-hidden="true">
        <div class="phone-preview">
          <div class="phone-screen">
            <div class="phone-handle"></div>
            <div class="ticket">
              <span class="ticket-eyebrow">Member Color</span>
              <span class="ticket-title">Pink Live '26</span>
              <div class="ticket-meta"><span>SEAT A-7</span><span>05·17 SAT</span></div>
            </div>
            <div class="mini-row"><span class="mini-label">参戦数</span><span class="mini-bar"><span style="width:78%"></span></span><span class="mini-pct">78%</span></div>
            <div class="mini-row"><span class="mini-label">推し貯金</span><span class="mini-bar"><span style="width:54%"></span></span><span class="mini-pct">54%</span></div>
            <div class="mini-row"><span class="mini-label">グッズ</span><span class="mini-bar"><span style="width:91%"></span></span><span class="mini-pct">91%</span></div>
          </div>
        </div>
      </div>
    </section>

    <section class="auth-shell" aria-labelledby="auth-heading">
      <div class="auth-top">
        <span class="auth-alt">初めての方は <a href="{{ .AltPageURL }}">{{ .AltPageLabel }}</a></span>
      </div>
      <div class="auth-card">
        <header class="auth-head">
          <span class="auth-eyebrow">Welcome back, fan! ✦</span>
          <h2 class="auth-title" id="auth-heading">共通アカウントで、推し活を再開。</h2>
          <p class="auth-sub">今日の推し活も安全に。メールとパスワードでログインします。</p>
        </header>
        <a class="btn btn-primary" href="{{ .KratosFlowURL }}">ログインして推し活へ →</a>
        <p class="auth-note">
          <a href="{{ .RecoveryURL }}">パスワードを忘れた場合</a> ・ ログインすると <a href="/legal/terms">利用規約</a> と <a href="/legal/privacy">プライバシーポリシー</a> に同意したとみなされます。
        </p>
      </div>
    </section>
  </main>

  <footer class="site-footer" aria-label="サイト情報">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
    <a href="/legal/contact">問い合わせ先</a>
    <a href="/legal/incident">障害時連絡先</a>
  </footer>
</body>
</html>`

const registrationPageBody = `
  <main class="stage">
    <section class="hero-card" aria-labelledby="hero-heading">
      <div class="brand-row">
        <div class="brand-mark" aria-hidden="true">★</div>
        <div class="brand-text">
          <strong>OshiLink</strong>
          <span>推し活 Account</span>
        </div>
      </div>
      <div class="hero-body">
        <div class="badge-row">
          <span class="badge">🎫 推しメンカラー対応</span>
          <span class="badge badge-violet">⭐ Fan ID</span>
          <span class="badge badge-mint">💫 30秒登録</span>
        </div>
        <h1 class="hero-title" id="hero-heading">推し活を、<br /><span>もっと一緒に。</span></h1>
        <p class="hero-lede">
          ひとつの共通アカウントで複数の推し活サービスに参加。30秒で登録して、推しメンカラーで彩ったプロフィールをすぐに使えます。
        </p>
        <div class="tag-row">
          <span class="tag">無料登録</span>
          <span class="tag">推しメンカラー選択</span>
          <span class="tag">すぐに使える</span>
        </div>
      </div>
      <div class="phone-wrap" aria-hidden="true">
        <div class="phone-preview">
          <div class="phone-screen">
            <div class="phone-handle"></div>
            <div class="ticket">
              <span class="ticket-eyebrow">Member Color</span>
              <span class="ticket-title">Pink Live '26</span>
              <div class="ticket-meta"><span>SEAT A-7</span><span>05·17 SAT</span></div>
            </div>
            <div class="mini-row"><span class="mini-label">参戦数</span><span class="mini-bar"><span style="width:78%"></span></span><span class="mini-pct">78%</span></div>
            <div class="mini-row"><span class="mini-label">推し貯金</span><span class="mini-bar"><span style="width:54%"></span></span><span class="mini-pct">54%</span></div>
            <div class="mini-row"><span class="mini-label">グッズ</span><span class="mini-bar"><span style="width:91%"></span></span><span class="mini-pct">91%</span></div>
          </div>
        </div>
      </div>
    </section>

    <section class="auth-shell" aria-labelledby="auth-heading">
      <div class="auth-top">
        <span class="auth-alt">すでにアカウントをお持ちの方は <a href="{{ .AltPageURL }}">{{ .AltPageLabel }}</a></span>
      </div>
      <div class="auth-card">
        <header class="auth-head">
          <span class="auth-eyebrow">推し活スタート ✦</span>
          <h2 class="auth-title" id="auth-heading">推し活アカウントを、今すぐ作成。</h2>
          <p class="auth-sub">推しメンカラーを選んで、30秒で登録完了。</p>
        </header>
        <a class="btn btn-primary" href="{{ .KratosFlowURL }}">推し活アカウントを作成する →</a>
        <p class="auth-note">
          「アカウント作成」を押すと <a href="/legal/terms">利用規約</a> と <a href="/legal/privacy">プライバシーポリシー</a> に同意したことになります。
        </p>
      </div>
    </section>
  </main>

  <footer class="site-footer" aria-label="サイト情報">
    <a href="/legal/terms">利用規約</a>
    <a href="/legal/privacy">プライバシーポリシー</a>
    <a href="/legal/contact">問い合わせ先</a>
    <a href="/legal/incident">障害時連絡先</a>
  </footer>

  <script nonce="{{.Nonce}}">
    (function () {
      var root = document.documentElement;
      document.querySelectorAll('.oshi-pick-dot').forEach(function (dot) {
        dot.addEventListener('click', function () {
          var color = dot.getAttribute('data-color');
          root.style.setProperty('--oshi', color);
          document.querySelectorAll('.oshi-pick-dot').forEach(function (d) {
            d.classList.toggle('active', d === dot);
          });
        });
      });
    })();
  </script>
</body>
</html>`
