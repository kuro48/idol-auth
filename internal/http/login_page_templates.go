package http

import "html/template"

var loginPageTpl = template.Must(template.New("login-page").Parse(loginPageTemplate))

const loginPageTemplate = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>OshiLink — 共通アカウントでログイン</title>
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
      --shadow:0 16px 40px rgba(48,35,28,.1);
      --radius-lg:28px;
      --radius-md:18px;
      --radius-sm:12px;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #181412;
        --surface: #221d1a;
        --surface-2: #2b241f;
        --text: #fff8f1;
        --muted: #c8b8ad;
        --border: #42362f;
        --shadow: 0 16px 40px rgba(0, 0, 0, 0.35);
        --oshi-weak: #372219;
        --oshi-soft: #5a3829;
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

    /* ---------- LEFT: hero card ---------- */
    .hero-card {
      position: relative;
      padding: clamp(28px, 4vw, 64px);
      background:
        radial-gradient(circle at 18px 18px,
          color-mix(in srgb, var(--oshi) 18%, transparent) 0 5px,
          transparent 6px) 0 0 / 34px 34px,
        linear-gradient(155deg, var(--oshi-weak) 0%, var(--surface) 70%);
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      gap: 32px;
      overflow: hidden;
      border-right: 1px solid var(--border);
    }
    .hero-card::after {
      content: "";
      position: absolute;
      right: -76px;
      bottom: -76px;
      width: 240px;
      height: 240px;
      border-radius: 50%;
      border: 28px solid var(--oshi);
      opacity: 0.16;
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
      background: radial-gradient(circle, color-mix(in srgb, var(--oshi) 22%, transparent) 0%, transparent 65%);
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
      box-shadow: 0 12px 28px color-mix(in srgb, var(--oshi) 36%, transparent);
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
    .badge {
      align-self: flex-start;
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 8px 14px;
      border-radius: 999px;
      background: var(--oshi-weak);
      color: var(--oshi);
      border: 1px solid color-mix(in srgb, var(--oshi) 22%, transparent);
      font-size: 12px;
      font-weight: 800;
      letter-spacing: 0.04em;
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
    .tag-row {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .tag {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 12px;
      border-radius: 999px;
      background: var(--surface);
      border: 1px solid var(--border);
      color: var(--text);
      font-size: 12px;
      font-weight: 700;
    }
    .tag::before {
      content: "";
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: var(--oshi);
    }

    /* ---------- phone mockup ---------- */
    .phone-wrap {
      position: relative;
      z-index: 1;
      display: flex;
      justify-content: flex-end;
      margin-top: auto;
    }
    .phone-preview {
      width: 280px;
      padding: 14px 14px 22px;
      background: #1e1714;
      border-radius: 36px;
      box-shadow:
        0 30px 60px rgba(48, 35, 28, 0.25),
        inset 0 0 0 2px rgba(255, 255, 255, 0.04);
      transform: rotate(-3deg);
    }
    .phone-screen {
      background: #fffaf7;
      border-radius: 26px;
      padding: 18px 16px;
      color: #26211f;
      min-height: 320px;
      display: flex;
      flex-direction: column;
      gap: 14px;
    }
    .phone-handle {
      width: 60px;
      height: 5px;
      border-radius: 999px;
      background: rgba(0, 0, 0, 0.1);
      margin: 0 auto 8px;
    }
    .ticket {
      padding: 14px;
      border-radius: 18px;
      background: linear-gradient(135deg, var(--oshi) 0%, color-mix(in srgb, var(--oshi) 70%, #fff) 100%);
      color: #fff;
      box-shadow: 0 10px 22px color-mix(in srgb, var(--oshi) 32%, transparent);
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .ticket-eyebrow {
      font-size: 10px;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      opacity: 0.85;
      font-weight: 800;
    }
    .ticket-title {
      font-size: 18px;
      font-weight: 900;
      letter-spacing: -0.02em;
    }
    .ticket-meta {
      display: flex;
      justify-content: space-between;
      font-size: 11px;
      font-weight: 700;
      opacity: 0.85;
    }
    .mini-row {
      display: flex;
      align-items: center;
      gap: 10px;
      font-size: 12px;
      color: #5b524d;
    }
    .mini-bar {
      flex: 1;
      height: 8px;
      border-radius: 999px;
      background: #f0e6dc;
      overflow: hidden;
      position: relative;
    }
    .mini-bar span {
      position: absolute;
      inset: 0 auto 0 0;
      background: var(--oshi);
      border-radius: 999px;
    }
    .mini-label {
      font-weight: 800;
      color: #26211f;
      min-width: 70px;
    }
    .mini-pct {
      font-weight: 800;
      color: var(--oshi);
      font-variant-numeric: tabular-nums;
    }

    /* ---------- RIGHT: auth card ---------- */
    .auth-shell {
      position: relative;
      padding: clamp(24px, 3vw, 48px);
      display: flex;
      flex-direction: column;
      gap: 28px;
      background: var(--bg);
    }
    .auth-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 16px;
    }
    .auth-help {
      color: var(--muted);
      font-size: 13px;
      font-weight: 600;
    }
    .auth-help a {
      color: var(--oshi);
      font-weight: 800;
      text-decoration: none;
      border-bottom: 2px solid color-mix(in srgb, var(--oshi) 35%, transparent);
    }

    .auth-card {
      position: relative;
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      box-shadow: var(--shadow);
      padding: clamp(24px, 2.6vw, 32px);
      display: flex;
      flex-direction: column;
      gap: 24px;
      max-width: 520px;
      width: 100%;
      margin: auto 0;
    }
    .auth-head {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .auth-eyebrow {
      color: var(--oshi);
      font-size: 12px;
      font-weight: 800;
      letter-spacing: 0.18em;
      text-transform: uppercase;
    }
    .auth-title {
      margin: 0;
      font-size: clamp(26px, 2.6vw, 32px);
      letter-spacing: -0.03em;
      font-weight: 900;
      line-height: 1.15;
    }
    .auth-sub {
      margin: 0;
      color: var(--muted);
      font-size: 14px;
      line-height: 1.7;
    }

    .tabs {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
      padding: 6px;
      background: var(--surface-2);
      border: 1px solid var(--border);
      border-radius: var(--radius-md);
    }
    .tab {
      appearance: none;
      border: none;
      background: transparent;
      padding: 12px 14px;
      border-radius: 12px;
      font-size: 14px;
      font-weight: 800;
      color: var(--muted);
      transition: background 0.18s ease, color 0.18s ease, transform 0.18s ease;
    }
    .tab:hover { color: var(--text); }
    .tab.active {
      background: var(--oshi);
      color: #fff;
      box-shadow: 0 6px 16px color-mix(in srgb, var(--oshi) 30%, transparent);
    }

    .panels { position: relative; }
    .panel { display: none; flex-direction: column; gap: 16px; }
    .panel.active { display: flex; animation: rise 0.32s ease both; }
    @keyframes rise {
      from { opacity: 0; transform: translateY(6px); }
      to { opacity: 1; transform: translateY(0); }
    }

    .field {
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .field label {
      font-size: 12px;
      font-weight: 800;
      color: var(--text);
      letter-spacing: 0.02em;
    }
    .field input,
    .field select {
      width: 100%;
      padding: 14px 16px;
      border-radius: 16px;
      border: 1px solid var(--border);
      background: var(--surface-2);
      color: var(--text);
      font-size: 14px;
      transition: border-color 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
    }
    .field input::placeholder { color: color-mix(in srgb, var(--muted) 85%, transparent); }
    .field input:focus,
    .field select:focus {
      outline: none;
      border-color: var(--oshi);
      background: var(--surface);
      box-shadow: 0 0 0 4px color-mix(in srgb, var(--oshi) 22%, transparent);
    }
    .field-hint {
      color: var(--muted);
      font-size: 12px;
      margin-top: 2px;
    }

    .oshi-pick {
      display: flex;
      gap: 10px;
      padding: 12px;
      background: var(--surface-2);
      border: 1px solid var(--border);
      border-radius: 16px;
      align-items: center;
      flex-wrap: wrap;
    }
    .oshi-pick-label {
      font-size: 12px;
      font-weight: 800;
      color: var(--muted);
      letter-spacing: 0.04em;
      margin-right: 4px;
    }
    .oshi-pick-dot {
      width: 26px;
      height: 26px;
      border-radius: 999px;
      border: 2px solid transparent;
      cursor: pointer;
      transition: transform 0.18s ease, border-color 0.18s ease;
      padding: 0;
    }
    .oshi-pick-dot:hover { transform: scale(1.08); }
    .oshi-pick-dot.active {
      border-color: var(--text);
      transform: scale(1.12);
    }

    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      width: 100%;
      padding: 15px 18px;
      border-radius: 16px;
      border: 1px solid transparent;
      font-size: 15px;
      font-weight: 900;
      letter-spacing: 0.01em;
      text-decoration: none;
      transition: transform 0.18s ease, box-shadow 0.18s ease, background 0.18s ease, color 0.18s ease;
      cursor: pointer;
    }
    .btn:hover { transform: translateY(-1px); }
    .btn:active { transform: translateY(0); }
    .btn-primary {
      background: var(--oshi);
      color: #fff;
      box-shadow: 0 16px 32px color-mix(in srgb, var(--oshi) 32%, transparent);
    }
    .btn-primary:hover {
      background: color-mix(in srgb, var(--oshi) 90%, #000 10%);
      box-shadow: 0 18px 38px color-mix(in srgb, var(--oshi) 42%, transparent);
    }
    .btn-secondary {
      background: var(--surface-2);
      color: var(--text);
      border-color: var(--border);
    }
    .btn-secondary:hover {
      background: var(--surface);
      border-color: color-mix(in srgb, var(--oshi) 45%, var(--border));
    }
    .btn-ghost {
      background: transparent;
      color: var(--muted);
      border: 1px solid transparent;
    }
    .btn-ghost:hover { color: var(--text); }

    .divider {
      display: grid;
      grid-template-columns: 1fr auto 1fr;
      align-items: center;
      gap: 12px;
      color: var(--muted);
      font-size: 11px;
      font-weight: 800;
      letter-spacing: 0.2em;
      text-transform: uppercase;
    }
    .divider::before,
    .divider::after {
      content: "";
      height: 1px;
      background: var(--border);
    }

    .auth-note {
      color: var(--muted);
      font-size: 12px;
      line-height: 1.7;
      text-align: center;
    }
    .auth-note a {
      color: var(--text);
      font-weight: 800;
      border-bottom: 1px dashed var(--border);
    }

    /* ---------- floating settings panel ---------- */
    .float-panel {
      position: fixed;
      right: 20px;
      bottom: 20px;
      z-index: 30;
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 10px 14px;
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 999px;
      box-shadow: var(--shadow);
    }
    .float-panel button {
      appearance: none;
      border: none;
      background: transparent;
      color: var(--text);
      padding: 6px;
      border-radius: 999px;
      display: grid;
      place-items: center;
      font-size: 18px;
      transition: background 0.18s ease;
    }
    .float-panel button:hover { background: var(--surface-2); }
    .float-panel .sep {
      width: 1px;
      height: 18px;
      background: var(--border);
    }
    .color-dots {
      display: flex;
      gap: 6px;
      align-items: center;
    }
    .color-dot {
      width: 22px;
      height: 22px;
      border-radius: 999px;
      border: 2px solid transparent;
      cursor: pointer;
      padding: 0;
      transition: transform 0.18s ease, border-color 0.18s ease;
    }
    .color-dot:hover { transform: scale(1.1); }
    .color-dot.active {
      border-color: var(--text);
      transform: scale(1.14);
    }

    /* ---------- responsive ---------- */
    @media (max-width: 980px) {
      .stage { grid-template-columns: 1fr; }
      .hero-card { border-right: none; border-bottom: 1px solid var(--border); }
      .auth-shell { padding: 24px; }
      .auth-card { margin: 0 auto; }
      .phone-wrap { justify-content: center; }
    }
    @media (max-width: 560px) {
      .hero-card { padding: 24px; gap: 24px; }
      .phone-preview { width: 240px; transform: rotate(-2deg); }
      .auth-card { padding: 22px; border-radius: 22px; }
      .tabs { grid-template-columns: 1fr 1fr; }
      .float-panel { right: 12px; bottom: 12px; padding: 8px 12px; gap: 8px; }
    }
  </style>
</head>
<body>
  <main class="stage">
    <!-- LEFT: hero -->
    <section class="hero-card" aria-labelledby="hero-heading">
      <div class="brand-row">
        <div class="brand-mark" aria-hidden="true">★</div>
        <div class="brand-text">
          <strong>OshiLink</strong>
          <span>Common Account</span>
        </div>
      </div>

      <div class="hero-body">
        <span class="badge">🎫 推しメンカラー対応</span>
        <h1 class="hero-title" id="hero-heading">
          推し活の入口を<br /><span>ひとつに。</span>
        </h1>
        <p class="hero-lede">
          推し活サービスをひとつの共通アカウントで横断。チケット、グッズ、ファンクラブ、配信、現場メモまで——
          サービスごとのログインを卒業して、推しに使う時間を取り戻そう。
        </p>
        <div class="tag-row">
          <span class="tag">共通ログイン</span>
          <span class="tag">プロフィール連携</span>
          <span class="tag">サービス横断</span>
          <span class="tag">管理者承認</span>
        </div>
      </div>

      <div class="phone-wrap" aria-hidden="true">
        <div class="phone-preview">
          <div class="phone-screen">
            <div class="phone-handle"></div>
            <div class="ticket">
              <span class="ticket-eyebrow">Member Color</span>
              <span class="ticket-title">Orange Live ’26</span>
              <div class="ticket-meta">
                <span>SEAT A-7</span>
                <span>05·17 SAT</span>
              </div>
            </div>
            <div class="mini-row">
              <span class="mini-label">参戦数</span>
              <span class="mini-bar"><span style="width:78%"></span></span>
              <span class="mini-pct">78%</span>
            </div>
            <div class="mini-row">
              <span class="mini-label">推し貯金</span>
              <span class="mini-bar"><span style="width:54%"></span></span>
              <span class="mini-pct">54%</span>
            </div>
            <div class="mini-row">
              <span class="mini-label">グッズ</span>
              <span class="mini-bar"><span style="width:91%"></span></span>
              <span class="mini-pct">91%</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- RIGHT: auth -->
    <section class="auth-shell" aria-labelledby="auth-heading">
      <div class="auth-top">
        <span class="auth-help">困ったときは <a href="/docs">ヘルプ</a></span>
      </div>

      <div class="auth-card">
        <header class="auth-head">
          <span class="auth-eyebrow">Welcome back</span>
          <h2 class="auth-title" id="auth-heading">共通アカウントで、推し活を再開。</h2>
          <p class="auth-sub">
            メールアドレスとパスワードでログイン。初めての方は、いつもの推しメンカラーを選んで30秒で登録できます。
          </p>
        </header>

        <div class="tabs" role="tablist" aria-label="認証モード">
          <button type="button" class="tab active" role="tab" aria-selected="true" data-target="panel-login" id="tab-login">ログイン</button>
          <button type="button" class="tab" role="tab" aria-selected="false" data-target="panel-signup" id="tab-signup">アカウント登録</button>
        </div>

        <div class="panels">
          <!-- LOGIN PANEL -->
          <div class="panel active" id="panel-login" role="tabpanel" aria-labelledby="tab-login">
            <div class="field">
              <label for="login-email">メールアドレス</label>
              <input id="login-email" type="email" autocomplete="email" placeholder="you@example.com" />
            </div>
            <div class="field">
              <label for="login-password">パスワード</label>
              <input id="login-password" type="password" autocomplete="current-password" placeholder="••••••••" />
              <span class="field-hint">パスワードを忘れた場合は、Kratosの再設定フローへ遷移します。</span>
            </div>
            <a class="btn btn-primary" href="{{ .LoginURL }}">
              共通アカウントでログイン →
            </a>
            <div class="divider"><span></span>または<span></span></div>
            <a class="btn btn-secondary" href="{{ .LoginURL }}">
              Googleで続ける
            </a>
            <p class="auth-note">
              ログインすると <a href="/docs">利用規約</a> と <a href="/docs">プライバシーポリシー</a> に同意したとみなされます。
            </p>
          </div>

          <!-- SIGNUP PANEL -->
          <div class="panel" id="panel-signup" role="tabpanel" aria-labelledby="tab-signup">
            <div class="field">
              <label for="signup-name">表示名</label>
              <input id="signup-name" type="text" autocomplete="nickname" placeholder="例: オレンジ担当" />
            </div>
            <div class="field">
              <label>推しメンカラー</label>
              <div class="oshi-pick" role="radiogroup" aria-label="推しメンカラー">
                <span class="oshi-pick-label">Color</span>
                <button type="button" class="oshi-pick-dot active" data-color="#ff8a3d" style="background:#ff8a3d" aria-label="オレンジ"></button>
                <button type="button" class="oshi-pick-dot" data-color="#ef6aa8" style="background:#ef6aa8" aria-label="ピンク"></button>
                <button type="button" class="oshi-pick-dot" data-color="#5a8dee" style="background:#5a8dee" aria-label="ブルー"></button>
                <button type="button" class="oshi-pick-dot" data-color="#35a67b" style="background:#35a67b" aria-label="グリーン"></button>
                <button type="button" class="oshi-pick-dot" data-color="#8e6be8" style="background:#8e6be8" aria-label="パープル"></button>
              </div>
            </div>
            <div class="field">
              <label for="signup-email">メールアドレス</label>
              <input id="signup-email" type="email" autocomplete="email" placeholder="you@example.com" />
            </div>
            <div class="field">
              <label for="signup-password">パスワード</label>
              <input id="signup-password" type="password" autocomplete="new-password" placeholder="英数字8文字以上" />
              <span class="field-hint">8文字以上・記号を含めると強度が上がります。</span>
            </div>
            <a class="btn btn-primary" href="{{ .RegistrationURL }}">
              アカウントを作成する →
            </a>
            <p class="auth-note">
              「アカウント作成」を押すと <a href="/docs">利用規約</a> と <a href="/docs">プライバシーポリシー</a> に同意したことになります。
            </p>
          </div>
        </div>
      </div>
    </section>
  </main>

  <!-- Floating theme + color panel -->
  <div class="float-panel" role="region" aria-label="推し色を選ぶ">
    <div class="color-dots" role="radiogroup" aria-label="推しメンカラー">
      <button type="button" class="color-dot active" data-color="#ff8a3d" style="background:#ff8a3d" aria-label="オレンジ"></button>
      <button type="button" class="color-dot" data-color="#ef6aa8" style="background:#ef6aa8" aria-label="ピンク"></button>
      <button type="button" class="color-dot" data-color="#5a8dee" style="background:#5a8dee" aria-label="ブルー"></button>
      <button type="button" class="color-dot" data-color="#35a67b" style="background:#35a67b" aria-label="グリーン"></button>
      <button type="button" class="color-dot" data-color="#8e6be8" style="background:#8e6be8" aria-label="パープル"></button>
    </div>
  </div>

  <script>
    (function () {
      var root = document.documentElement;

      // color dots (floating panel)
      document.querySelectorAll('.color-dot').forEach(function (dot) {
        dot.addEventListener('click', function () {
          var color = dot.getAttribute('data-color');
          root.style.setProperty('--oshi', color);
          document.querySelectorAll('.color-dot').forEach(function (d) {
            d.classList.toggle('active', d === dot);
          });
          document.querySelectorAll('.oshi-pick-dot').forEach(function (d) {
            d.classList.toggle('active', d.getAttribute('data-color') === color);
          });
        });
      });

      // oshi picker inside signup
      document.querySelectorAll('.oshi-pick-dot').forEach(function (dot) {
        dot.addEventListener('click', function () {
          var color = dot.getAttribute('data-color');
          root.style.setProperty('--oshi', color);
          document.querySelectorAll('.oshi-pick-dot').forEach(function (d) {
            d.classList.toggle('active', d === dot);
          });
          document.querySelectorAll('.color-dot').forEach(function (d) {
            d.classList.toggle('active', d.getAttribute('data-color') === color);
          });
        });
      });

      // tab switching
      var tabs = document.querySelectorAll('.tab');
      tabs.forEach(function (tab) {
        tab.addEventListener('click', function () {
          var target = tab.getAttribute('data-target');
          tabs.forEach(function (t) {
            var on = t === tab;
            t.classList.toggle('active', on);
            t.setAttribute('aria-selected', on ? 'true' : 'false');
          });
          document.querySelectorAll('.panel').forEach(function (p) {
            p.classList.toggle('active', p.id === target);
          });
        });
      });
    })();
  </script>
</body>
</html>`
