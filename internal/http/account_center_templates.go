package http

import "html/template"

var accountCenterTpl = template.Must(template.New("account-center").Parse(`<!DOCTYPE html>
<html lang="ja" data-theme="light">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>アカウントセンター | OshiLink</title>
  <style>
    :root{
      --oshi:{{.OshiColor}};
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
      --danger:#e85d75;
      --success:#35a67b;
      --warning:#e9a23b;
    }
    html[data-theme="dark"]{
      --bg:#181412; --surface:#221d1a; --surface-2:#2b241f; --text:#fff8f1; --muted:#c8b8ad; --border:#42362f;
      --shadow:0 16px 40px rgba(0,0,0,.35); --oshi-weak:#372219; --oshi-soft:#5a3829;
    }
    *,*::before,*::after{box-sizing:border-box}
    html,body{margin:0;padding:0}
    body{
      min-height:100vh;
      font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Hiragino Sans","Yu Gothic","Noto Sans JP",sans-serif;
      color:var(--text);
      background:var(--bg);
      letter-spacing:.01em;
    }
    button,input,select{font:inherit;color:inherit}
    button{cursor:pointer}
    a{color:var(--oshi);text-decoration:none}

    .topbar{
      position:sticky; top:0; z-index:20;
      display:flex; align-items:center; justify-content:space-between;
      gap:16px; padding:14px 28px;
      background:color-mix(in srgb, var(--surface) 88%, transparent);
      backdrop-filter:saturate(160%) blur(14px);
      -webkit-backdrop-filter:saturate(160%) blur(14px);
      border-bottom:1px solid var(--border);
    }
    .brand{display:flex; align-items:center; gap:12px}
    .brand-mark{
      width:42px; height:42px; border-radius:14px;
      background:var(--oshi); color:#fff;
      display:grid; place-items:center;
      font-weight:900; font-size:20px; letter-spacing:-.02em;
      box-shadow:0 10px 24px color-mix(in srgb,var(--oshi) 32%,transparent);
    }
    .brand-text strong{display:block; font-size:16px; line-height:1.1; letter-spacing:-.01em}
    .brand-text span{font-size:11px; color:var(--muted); font-weight:700; text-transform:uppercase; letter-spacing:.12em}
    .top-actions{display:flex; align-items:center; gap:12px}
    .user-chip{
      display:flex; align-items:center; gap:10px;
      padding:6px 12px 6px 6px; border-radius:999px;
      background:var(--surface-2); border:1px solid var(--border);
      font-size:13px; color:var(--text); max-width:280px;
    }
    .user-chip .dot{
      width:32px; height:32px; border-radius:50%;
      background:var(--oshi); color:#fff;
      display:grid; place-items:center; font-weight:900; font-size:13px;
    }
    .user-chip .email{overflow:hidden; text-overflow:ellipsis; white-space:nowrap; max-width:200px; color:var(--muted)}
    .logout-btn{
      border:1px solid var(--border); background:var(--surface);
      color:var(--text); padding:9px 14px; border-radius:999px;
      font-weight:800; font-size:13px;
      transition:transform 160ms ease, background 160ms ease;
    }
    .logout-btn:hover{transform:translateY(-1px); background:var(--surface-2)}

    .shell{max-width:1180px; margin:0 auto; padding:28px 28px 120px}
    .grid{display:grid; grid-template-columns:minmax(0,1.1fr) minmax(0,.9fr); gap:20px}
    .col{display:flex; flex-direction:column; gap:20px; min-width:0}

    .card{
      background:var(--surface);
      border:1px solid var(--border);
      border-radius:var(--radius-lg);
      box-shadow:var(--shadow);
      padding:24px;
    }
    .card-head{display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:18px}
    .card-head h2{margin:0; font-size:20px; font-weight:700; letter-spacing:-.01em}
    .hint{margin:0 0 14px; color:var(--muted); font-size:13px; line-height:1.7}

    .profile-hero{
      position:relative; overflow:hidden;
      padding:30px;
      background:var(--surface);
      border:1px solid var(--border);
      border-radius:var(--radius-lg);
      box-shadow:var(--shadow);
    }
    .profile-hero::before{
      content:""; position:absolute; inset:0;
      background:
        linear-gradient(90deg, color-mix(in srgb, var(--oshi) 15%, transparent) 1px, transparent 1px) 0 0 / 28px 28px,
        linear-gradient(0deg, color-mix(in srgb, var(--oshi) 15%, transparent) 1px, transparent 1px) 0 0 / 28px 28px;
      opacity:.5; pointer-events:none;
    }
    .profile-inner{position:relative; display:flex; gap:24px; align-items:center; flex-wrap:wrap}
    .profile-photo{
      width:146px; height:146px; border-radius:36px;
      background:
        radial-gradient(circle at 45px 36px, #fff 0 14px, transparent 15px),
        var(--oshi);
      border:8px solid var(--surface);
      box-shadow:0 16px 34px color-mix(in srgb,var(--oshi) 24%,transparent);
      display:grid; place-items:center;
      color:#fff; font-size:48px; font-weight:1000;
      background-size:auto, cover; background-position:center;
      flex-shrink:0;
    }
    .profile-photo.has-image{color:transparent}
    .profile-name{flex:1; min-width:200px}
    .profile-name h2{margin:0; font-size:34px; letter-spacing:-.03em; line-height:1.05}
    .profile-name .handle{
      margin:8px 0 0; color:var(--muted); font-size:13px;
      word-break:break-all; line-height:1.5;
    }
    .badge-oshi{
      display:inline-flex; align-items:center; gap:8px;
      margin-top:14px;
      padding:8px 13px; border-radius:999px;
      background:var(--oshi-weak); color:var(--oshi);
      font-weight:800; font-size:12px;
    }
    .badge-oshi .swatch{
      width:14px; height:14px; border-radius:50%;
      background:var(--oshi);
      box-shadow:0 0 0 2px color-mix(in srgb,var(--oshi) 35%,transparent);
    }
    .profile-actions{display:flex; gap:10px; flex-wrap:wrap; margin-top:18px}

    .info-list{display:grid; gap:10px}
    .info-item{
      display:flex; justify-content:space-between; align-items:center; gap:14px;
      padding:14px; border-radius:16px;
      background:var(--surface-2); border:1px solid var(--border);
    }
    .info-item .k{font-size:11px; font-weight:800; color:var(--muted); text-transform:uppercase; letter-spacing:.1em}
    .info-item .v{font-size:14px; color:var(--text); font-weight:600; text-align:right; word-break:break-all; max-width:62%}
    .empty{color:var(--muted); font-style:italic; font-weight:500}

    [data-mode="display"] .section-edit{display:none}
    [data-mode="edit"] .section-display{display:none}
    .section-edit{display:grid; gap:14px}

    .field{display:grid; gap:6px}
    .field label{font-size:12px; font-weight:800; color:var(--muted); letter-spacing:.05em}
    .field input[type=text],
    .field input[type=date],
    .field input[type=file]{
      width:100%;
      border:1px solid var(--border);
      background:var(--surface-2);
      color:var(--text);
      padding:14px 16px;
      border-radius:16px;
      outline:none;
      transition:border-color 160ms ease, box-shadow 160ms ease;
    }
    .field input[type=text]:focus,
    .field input[type=date]:focus,
    .field input[type=file]:focus{
      border-color:var(--oshi);
      box-shadow:0 0 0 4px color-mix(in srgb,var(--oshi) 18%,transparent);
    }
    .field input[type=file]{padding:11px 14px; font-size:13px}
    .field-note{font-size:11px; color:var(--muted); line-height:1.5}

    .avatar-editor{
      display:grid;
      gap:14px;
      padding:14px;
      border:1px solid var(--border);
      border-radius:18px;
      background:var(--surface-2);
    }
    .avatar-editor-layout{display:grid; grid-template-columns:180px 1fr; gap:16px; align-items:start}
    .avatar-canvas-wrap{
      width:180px;
      aspect-ratio:1;
      border-radius:24px;
      overflow:hidden;
      border:1px solid var(--border);
      background:var(--surface);
    }
    .avatar-canvas{display:block; width:100%; height:100%}
    .avatar-controls{display:grid; gap:12px; min-width:0}
    .avatar-control{display:grid; gap:6px}
    .avatar-control label{
      display:flex;
      justify-content:space-between;
      gap:10px;
      font-size:11px;
      font-weight:800;
      color:var(--muted);
      letter-spacing:.05em;
    }
    .avatar-control input[type=range]{
      width:100%;
      accent-color:var(--oshi);
    }
    .avatar-rotate-actions{display:flex; gap:8px; flex-wrap:wrap}

    .check-grid{display:grid; gap:10px; padding:14px; background:var(--surface-2); border:1px solid var(--border); border-radius:16px}
    .check-row{display:flex; align-items:center; gap:10px; font-size:14px; color:var(--text); cursor:pointer}
    .check-row input[type=checkbox]{
      width:18px; height:18px; border-radius:5px;
      accent-color:var(--oshi); flex-shrink:0;
    }

    .form-error{
      font-size:13px; color:var(--danger);
      min-height:18px; padding:0 4px;
      font-weight:700;
    }
    .form-actions{display:flex; gap:10px; flex-wrap:wrap; margin-top:6px}

    .btn{
      border:0; border-radius:16px; padding:14px 18px;
      font-weight:900; font-size:14px; line-height:1;
      display:inline-flex; align-items:center; gap:8px;
      transition:transform 160ms ease, box-shadow 160ms ease, background 160ms ease;
    }
    .btn:hover{transform:translateY(-1px)}
    .btn-primary{
      background:var(--oshi); color:#fff;
      box-shadow:0 12px 24px color-mix(in srgb,var(--oshi) 28%,transparent);
    }
    .btn-secondary{
      background:var(--surface-2); border:1px solid var(--border); color:var(--text);
    }
    .btn-danger{background:var(--danger); color:#fff}
    .btn-sm{padding:9px 13px; font-size:12px; border-radius:12px}
    .btn-ghost-danger{
      background:transparent; border:1px solid color-mix(in srgb,var(--danger) 40%,transparent);
      color:var(--danger);
    }

    .service-list{display:grid; gap:12px}
    .service-card{
      padding:18px; border-radius:22px;
      background:var(--surface);
      border:1px solid var(--border);
      transition:transform 160ms ease, border-color 160ms ease;
      display:flex; justify-content:space-between; align-items:flex-start; gap:14px;
    }
    .service-card:hover{transform:translateY(-2px); border-color:var(--oshi)}
    .service-card h4{margin:6px 0 4px; font-size:16px; font-weight:800; letter-spacing:-.01em}
    .service-card p{margin:0; color:var(--muted); font-size:12px; line-height:1.6}
    .status-pill{
      padding:5px 10px; border-radius:999px; font-size:11px; font-weight:900;
      background:color-mix(in srgb,var(--success) 14%,transparent); color:var(--success);
      text-transform:uppercase; letter-spacing:.08em;
    }
    .empty-state{
      padding:28px 14px; text-align:center; color:var(--muted); font-size:13px;
      border:1px dashed var(--border); border-radius:18px; background:var(--surface-2);
    }

    .card-danger{
      border-color:color-mix(in srgb,var(--danger) 24%,var(--border));
      background:
        linear-gradient(180deg, color-mix(in srgb,var(--danger) 6%,transparent) 0%, transparent 60%),
        var(--surface);
    }
    .card-danger .card-head h2{color:var(--danger)}
    .status-line{
      margin-top:10px; padding:14px; border-radius:16px;
      background:var(--surface-2); border:1px solid var(--border);
      font-size:13px; color:var(--text); line-height:1.6;
    }

    .contact-footer{
      display:flex; align-items:center; gap:12px; flex-wrap:wrap;
      padding-top:14px; margin-top:14px;
      border-top:1px solid var(--border);
    }
    .contact-note{font-size:11px; color:var(--muted); line-height:1.5; flex:1; min-width:180px}

    .toast{
      position:fixed; right:22px; bottom:22px;
      background:var(--text); color:var(--surface);
      padding:14px 18px; border-radius:18px;
      display:none; max-width:340px; font-size:14px; line-height:1.5;
      box-shadow:0 18px 40px rgba(0,0,0,.2);
      z-index:200; font-weight:700;
    }
    .toast.error{background:var(--danger); color:#fff}
    .toast.show{display:block; animation:toastIn 220ms ease both}
    @keyframes toastIn{from{opacity:0; transform:translateY(8px)} to{opacity:1; transform:translateY(0)}}

    .float-tools{
      position:fixed; right:22px; bottom:88px; z-index:150;
      display:flex; flex-direction:column; align-items:flex-end; gap:10px;
    }
    .theme-toggle{
      width:48px; height:48px; border-radius:50%;
      background:var(--surface); border:1px solid var(--border);
      color:var(--text); box-shadow:var(--shadow);
      display:grid; place-items:center; font-size:20px;
      transition:transform 160ms ease;
    }
    .theme-toggle:hover{transform:translateY(-2px)}
    .color-dots{
      display:flex; gap:8px; padding:10px 12px;
      background:var(--surface); border:1px solid var(--border);
      border-radius:999px; box-shadow:var(--shadow);
    }
    .color-dot{
      width:24px; height:24px; border-radius:50%;
      border:3px solid var(--surface);
      box-shadow:0 0 0 1px var(--border);
      transition:transform 160ms ease;
      padding:0;
    }
    .color-dot:hover{transform:scale(1.1)}
    .color-dot[aria-pressed="true"]{box-shadow:0 0 0 2px var(--text)}

    @media(max-width:920px){
      .grid{grid-template-columns:1fr}
      .topbar{padding:12px 18px}
      .shell{padding:20px 18px 120px}
      .user-chip .email{max-width:130px}
      .profile-name h2{font-size:26px}
      .profile-photo{width:112px; height:112px; border-radius:28px; font-size:36px; border-width:6px}
      .card{padding:18px}
      .profile-hero{padding:22px}
      .avatar-editor-layout{grid-template-columns:1fr}
      .avatar-canvas-wrap{width:min(100%,220px)}
    }
    @media(max-width:520px){
      .brand-text{display:none}
      .user-chip .email{max-width:90px}
      .float-tools{right:14px; bottom:80px}
    }
  </style>
</head>
<body>

  <header class="topbar">
    <div class="brand">
      <div class="brand-mark">推</div>
      <div class="brand-text">
        <strong>OshiLink</strong>
        <span>Account Center</span>
      </div>
    </div>
    <div class="top-actions">
      <div class="user-chip" title="{{if .Email}}{{.Email}}{{else}}{{.IdentityID}}{{end}}">
        <span class="dot">{{.Initials}}</span>
        <span class="email">{{if .Email}}{{.Email}}{{else}}{{.IdentityID}}{{end}}</span>
      </div>
      <a class="logout-btn" href="{{.LogoutURL}}">ログアウト</a>
    </div>
  </header>

  <main class="shell">
    <div class="grid">

      <div class="col">

        <section class="profile-hero">
          <div class="profile-inner">
            <div class="profile-photo avatar" data-initials="{{.Initials}}">{{.Initials}}</div>
            <div class="profile-name">
              <h2>{{if .DisplayName}}{{.DisplayName}}{{else}}アカウント{{end}}</h2>
              <p class="handle">{{.IdentityID}}</p>
              <span class="badge-oshi"><span class="swatch"></span>推し色設定済み</span>
              <div class="profile-actions">
                <button type="button" class="btn btn-primary" id="btn-edit-profile">プロフィール編集</button>
                <a class="btn btn-secondary" href="{{.KratosSettingsURL}}">設定を変更</a>
              </div>
            </div>
          </div>
        </section>

        <article class="card">
          <div class="card-head"><h2>連絡先情報</h2></div>
          <div class="info-list">
            <div class="info-item">
              <span class="k">メール</span>
              <span class="v">{{if .Email}}{{.Email}}{{else}}<span class="empty">未登録</span>{{end}}</span>
            </div>
            <div class="info-item">
              <span class="k">電話番号</span>
              <span class="v">{{if .Phone}}{{.Phone}}{{else}}<span class="empty">未登録</span>{{end}}</span>
            </div>
          </div>
          <div class="contact-footer">
            <a href="{{.KratosSettingsURL}}" class="btn btn-secondary btn-sm">連絡先を変更 →</a>
            <span class="contact-note">メール・電話番号の変更はKratosの設定画面で行います</span>
          </div>
        </article>

        <article class="card" data-section="profile" data-mode="display">
          <div class="card-head">
            <h2>共有プロフィール</h2>
            <button class="btn btn-secondary btn-sm" data-action="edit">編集</button>
          </div>
          <div class="section-display">
            <p class="hint">各アプリは連携済みユーザーに対してこのプロフィールを参照できます。</p>
            <div class="info-list">
              <div class="info-item"><span class="k">表示名</span><span class="v" data-display="display_name"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">推し色</span><span class="v" data-display="oshi_color"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">推しID</span><span class="v" data-display="oshi_ids"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">ファン歴</span><span class="v" data-display="fan_since"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">アバター</span><span class="v" data-display="avatar_url"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">ロケール</span><span class="v" data-display="locale"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">タイムゾーン</span><span class="v" data-display="timezone"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">生年月日</span><span class="v" data-display="birthdate"><span class="empty">—</span></span></div>
              <div class="info-item"><span class="k">通知設定</span><span class="v" data-display="notification_preferences"><span class="empty">—</span></span></div>
            </div>
          </div>
          <form class="section-edit" id="form-profile">
            <div class="field"><label>表示名</label><input name="display_name" type="text" maxlength="50"></div>
            <div class="field"><label>推し色（#rrggbb）</label><input name="oshi_color" type="text" placeholder="#ff88cc"></div>
            <div class="field"><label>推しID（カンマ区切り、最大10件）</label><input name="oshi_ids" type="text"></div>
            <div class="field"><label>ファン歴（YYYY または YYYY-MM）</label><input name="fan_since" type="text" placeholder="2022-03"></div>
            <div class="field">
              <label>アバター画像</label>
              <input id="avatar-file" name="avatar_file" type="file" accept="image/png,image/jpeg,image/gif,image/webp">
              <div class="field-note">PNG / JPEG / GIF / WebP、10MBまで。保存時に512px四方のJPEGへ変換されます。</div>
              <div class="avatar-editor" id="avatar-editor" aria-label="アバター画像を加工">
                <div class="avatar-editor-layout">
                  <div class="avatar-canvas-wrap">
                    <canvas class="avatar-canvas" id="avatar-canvas" width="512" height="512"></canvas>
                  </div>
                  <div class="avatar-controls">
                    <div class="avatar-control">
                      <label for="avatar-zoom"><span>拡大</span><span id="avatar-zoom-value">100%</span></label>
                      <input id="avatar-zoom" type="range" min="1" max="3" step="0.01" value="1">
                    </div>
                    <div class="avatar-control">
                      <label for="avatar-crop-x"><span>左右</span><span id="avatar-crop-x-value">0</span></label>
                      <input id="avatar-crop-x" type="range" min="-100" max="100" step="1" value="0">
                    </div>
                    <div class="avatar-control">
                      <label for="avatar-crop-y"><span>上下</span><span id="avatar-crop-y-value">0</span></label>
                      <input id="avatar-crop-y" type="range" min="-100" max="100" step="1" value="0">
                    </div>
                    <div class="avatar-rotate-actions">
                      <button type="button" class="btn btn-secondary btn-sm" id="avatar-rotate-left">左へ回転</button>
                      <button type="button" class="btn btn-secondary btn-sm" id="avatar-rotate-right">右へ回転</button>
                      <button type="button" class="btn btn-secondary btn-sm" id="avatar-reset-edit">リセット</button>
                    </div>
                    <div class="field-note">プレビューの正方形が保存されます。保存後もサーバー側で512px JPEGへ再変換します。</div>
                  </div>
                </div>
              </div>
            </div>
            <div class="field"><label>ロケール</label><input name="locale" type="text" placeholder="ja-JP"></div>
            <div class="field"><label>タイムゾーン</label><input name="timezone" type="text" placeholder="Asia/Tokyo"></div>
            <div class="field">
              <label>生年月日</label>
              <input name="birthdate" type="date">
              <div class="field-note">本人向けアカウント情報として保存します。連携アプリ向け公開プロフィールには含まれません。</div>
            </div>
            <div class="field">
              <label>通知設定</label>
              <div class="check-grid">
                <label class="check-row"><input name="notify_email_enabled" type="checkbox"><span>メール通知を受け取る</span></label>
                <label class="check-row"><input name="notify_product_updates" type="checkbox"><span>プロダクト更新</span></label>
                <label class="check-row"><input name="notify_security_alerts" type="checkbox"><span>セキュリティ通知</span></label>
                <label class="check-row"><input name="notify_community_notifications" type="checkbox"><span>コミュニティ通知</span></label>
                <label class="check-row"><input name="notify_app_membership_notifications" type="checkbox"><span>アプリ連携通知</span></label>
              </div>
            </div>
            <div class="form-error" id="profile-error" role="alert"></div>
            <div class="form-actions">
              <button type="submit" class="btn btn-primary">保存</button>
              <button type="button" class="btn btn-secondary" data-action="cancel">キャンセル</button>
            </div>
          </form>
        </article>

      </div>

      <div class="col">

        <article class="card">
          <div class="card-head"><h2>連携中アプリ</h2></div>
          <p class="hint">各アプリでアカウント削除をする場合は、まずここで連携を解除してください。</p>
          <div id="memberships" class="service-list"></div>
        </article>

        <article class="card card-danger" data-section="deletion" data-mode="display">
          <div class="card-head">
            <h2>アカウント削除</h2>
            <button class="btn btn-ghost-danger btn-sm" data-action="edit">予約・管理</button>
          </div>
          <div class="section-display">
            <p class="hint"><strong>注意:</strong> 削除予約すると連携中の全アプリに影響します。各アプリだけをやめたい場合は連携解除を使ってください。</p>
            <div id="deletion-status" class="status-line">削除予約はありません。</div>
          </div>
          <div class="section-edit">
            <p class="hint">削除予約するか、予約中の場合は取り消せます。</p>
            <div class="field"><label>理由（省略可）</label><input id="delete-reason" type="text" placeholder="user_requested"></div>
            <div class="form-actions">
              <button class="btn btn-danger btn-sm" type="button" id="btn-schedule">削除を予約</button>
              <button class="btn btn-secondary btn-sm" type="button" id="btn-cancel-del">予約を取り消す</button>
              <button class="btn btn-secondary btn-sm" type="button" data-action="cancel">閉じる</button>
            </div>
            <div id="deletion-status-edit" class="status-line" style="display:none"></div>
          </div>
        </article>

      </div>
    </div>
  </main>

  <div class="float-tools">
    <div class="color-dots" role="group" aria-label="推し色を選ぶ">
      <button class="color-dot" type="button" data-color="#ff8a3d" style="background:#ff8a3d" aria-pressed="true" title="オレンジ"></button>
      <button class="color-dot" type="button" data-color="#ff5fa2" style="background:#ff5fa2" aria-pressed="false" title="ピンク"></button>
      <button class="color-dot" type="button" data-color="#7c5cff" style="background:#7c5cff" aria-pressed="false" title="パープル"></button>
      <button class="color-dot" type="button" data-color="#39b58a" style="background:#39b58a" aria-pressed="false" title="グリーン"></button>
      <button class="color-dot" type="button" data-color="#4b7bec" style="background:#4b7bec" aria-pressed="false" title="ブルー"></button>
    </div>
  </div>
  <button class="theme-toggle" id="theme-toggle" type="button" aria-label="テーマを切り替える" title="テーマ切替"
    style="position:fixed;right:22px;bottom:22px;z-index:150">☾</button>

  <div id="toast" class="toast"></div>

  <script>
    function showToast(msg, type) {
      var el = document.getElementById('toast');
      el.textContent = msg;
      el.className = 'toast';
      if (type === 'error') el.classList.add('error');
      el.classList.add('show');
      clearTimeout(window.__tt);
      window.__tt = setTimeout(function(){ el.classList.remove('show'); }, 3200);
    }

    function showError(err) {
      showToast(err && err.message ? err.message : 'エラーが発生しました', 'error');
    }

    function splitCSV(v) {
      return v.split(',').map(function(s){ return s.trim(); }).filter(Boolean);
    }

    var AVATAR_MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

    var avatarEditState = {
      image: null,
      rotation: 0,
      ready: false
    };

    function assertAvatarSize(blob) {
      if (blob && blob.size > AVATAR_MAX_UPLOAD_BYTES) {
        throw new Error('画像サイズが10MBを超えています。加工後の画像を小さくするか、別の画像を選択してください。');
      }
    }

    function avatarFileInput(form) {
      return form ? form.querySelector('input[name="avatar_file"]') : null;
    }

    function avatarEditorElements() {
      return {
        editor: document.getElementById('avatar-editor'),
        canvas: document.getElementById('avatar-canvas'),
        zoom: document.getElementById('avatar-zoom'),
        cropX: document.getElementById('avatar-crop-x'),
        cropY: document.getElementById('avatar-crop-y'),
        zoomValue: document.getElementById('avatar-zoom-value'),
        cropXValue: document.getElementById('avatar-crop-x-value'),
        cropYValue: document.getElementById('avatar-crop-y-value')
      };
    }

    function resetAvatarEditor() {
      avatarEditState.image = null;
      avatarEditState.rotation = 0;
      avatarEditState.ready = false;
      var els = avatarEditorElements();
      if (els.editor) els.editor.classList.remove('is-open');
      if (els.zoom) els.zoom.value = '1';
      if (els.cropX) els.cropX.value = '0';
      if (els.cropY) els.cropY.value = '0';
      if (els.canvas) {
        var ctx = els.canvas.getContext('2d');
        ctx.clearRect(0, 0, els.canvas.width, els.canvas.height);
      }
    }

    function updateAvatarEditorLabels(els) {
      if (els.zoomValue && els.zoom) els.zoomValue.textContent = Math.round(Number(els.zoom.value) * 100) + '%';
      if (els.cropXValue && els.cropX) els.cropXValue.textContent = els.cropX.value;
      if (els.cropYValue && els.cropY) els.cropYValue.textContent = els.cropY.value;
    }

    function renderAvatarEditor() {
      var els = avatarEditorElements();
      if (!els.canvas || !avatarEditState.image) return;
      var ctx = els.canvas.getContext('2d');
      var size = els.canvas.width;
      var img = avatarEditState.image;
      var rotation = avatarEditState.rotation * Math.PI / 180;
      var zoom = Number(els.zoom.value || 1);
      var panX = Number(els.cropX.value || 0) / 100 * size * 0.5;
      var panY = Number(els.cropY.value || 0) / 100 * size * 0.5;
      var cos = Math.abs(Math.cos(rotation));
      var sin = Math.abs(Math.sin(rotation));
      var rotatedWidth = img.naturalWidth * cos + img.naturalHeight * sin;
      var rotatedHeight = img.naturalWidth * sin + img.naturalHeight * cos;
      var scale = Math.max(size / rotatedWidth, size / rotatedHeight) * zoom;

      updateAvatarEditorLabels(els);
      ctx.clearRect(0, 0, size, size);
      ctx.fillStyle = getComputedStyle(document.documentElement).getPropertyValue('--surface').trim() || '#fff';
      ctx.fillRect(0, 0, size, size);
      ctx.save();
      ctx.translate(size / 2 + panX, size / 2 + panY);
      ctx.rotate(rotation);
      ctx.drawImage(img, -img.naturalWidth * scale / 2, -img.naturalHeight * scale / 2, img.naturalWidth * scale, img.naturalHeight * scale);
      ctx.restore();
    }

    function loadAvatarEditorFile(file) {
      if (!file) {
        resetAvatarEditor();
        return;
      }
      try {
        assertAvatarSize(file);
      } catch (err) {
        resetAvatarEditor();
        showError(err);
        var form = document.getElementById('form-profile');
        var input = avatarFileInput(form);
        if (input) input.value = '';
        return;
      }
      if (!/^image\/(png|jpe?g|gif|webp)$/.test(file.type)) {
        resetAvatarEditor();
        showError(new Error('PNG / JPEG / GIF / WebP の画像を選択してください'));
        return;
      }
      var reader = new FileReader();
      reader.onload = function() {
        var img = new Image();
        img.onload = function() {
          var els = avatarEditorElements();
          avatarEditState.image = img;
          avatarEditState.rotation = 0;
          avatarEditState.ready = true;
          if (els.zoom) els.zoom.value = '1';
          if (els.cropX) els.cropX.value = '0';
          if (els.cropY) els.cropY.value = '0';
          if (els.editor) els.editor.classList.add('is-open');
          renderAvatarEditor();
        };
        img.onerror = function() {
          resetAvatarEditor();
          showError(new Error('画像を読み込めませんでした'));
        };
        img.src = String(reader.result || '');
      };
      reader.onerror = function() {
        resetAvatarEditor();
        showError(new Error('画像を読み込めませんでした'));
      };
      reader.readAsDataURL(file);
    }

    function exportEditedAvatarBlob() {
      return new Promise(function(resolve, reject) {
        var els = avatarEditorElements();
        if (!avatarEditState.ready || !els.canvas) {
          resolve(null);
          return;
        }
        renderAvatarEditor();
        els.canvas.toBlob(function(blob) {
          if (!blob) {
            reject(new Error('アバター画像の加工に失敗しました'));
            return;
          }
          resolve(blob);
        }, 'image/jpeg', 0.92);
      });
    }

    async function req(method, path, body) {
      var opts = { method: method, credentials: 'same-origin', headers: {} };
      if (body !== undefined) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
      var res = await fetch(path, opts);
      var text = await res.text();
      var data = text ? JSON.parse(text) : null;
      if (!res.ok) throw new Error(data && data.error ? data.error : 'request failed: ' + res.status);
      return data;
    }

    function setDisplay(card, field, value) {
      var el = card.querySelector('[data-display="' + field + '"]');
      if (!el) return;
      el.textContent = '';
      var child = document.createElement('span');
      if (value) {
        child.textContent = value;
      } else {
        child.className = 'empty';
        child.textContent = '—';
      }
      el.appendChild(child);
    }

    function notificationSummary(prefs) {
      if (!prefs) return null;
      var enabled = [];
      if (prefs.email_enabled) enabled.push('メール');
      if (prefs.product_updates) enabled.push('プロダクト更新');
      if (prefs.security_alerts) enabled.push('セキュリティ');
      if (prefs.community_notifications) enabled.push('コミュニティ');
      if (prefs.app_membership_notifications) enabled.push('アプリ連携');
      return enabled.length ? enabled.join(', ') : 'すべてOFF';
    }

    function enterEdit(card) {
      var form = card.querySelector('form');
      if (form) {
        form.querySelectorAll('input').forEach(function(inp){
          if (inp.type === 'file') { inp.value = ''; return; }
          inp.dataset.snapshot = inp.type === 'checkbox' ? String(inp.checked) : inp.value;
        });
      }
      card.dataset.mode = 'edit';
      if (form) { var f = form.querySelector('input'); if (f) f.focus(); }
    }

    function exitEdit(card) {
      card.dataset.mode = 'display';
      var form = card.querySelector('form');
      if (form) {
        form.querySelectorAll('input').forEach(function(inp){
          if (inp.type === 'file') {
            inp.value = '';
            resetAvatarEditor();
            return;
          }
          if (inp.dataset.snapshot === undefined) return;
          if (inp.type === 'checkbox') {
            inp.checked = inp.dataset.snapshot === 'true';
          } else {
            inp.value = inp.dataset.snapshot;
          }
        });
        var errEl = form.querySelector('[role="alert"]');
        if (errEl) errEl.textContent = '';
      }
      var editBtn = card.querySelector('[data-action="edit"]');
      if (editBtn) editBtn.focus();
    }

    document.addEventListener('click', function(e) {
      var card = e.target.closest('[data-section]');
      if (!card) return;
      if (e.target.closest('[data-action="edit"]')) { enterEdit(card); return; }
      if (e.target.closest('[data-action="cancel"]')) { exitEdit(card); return; }
    });

    var heroEditBtn = document.getElementById('btn-edit-profile');
    if (heroEditBtn) {
      heroEditBtn.addEventListener('click', function() {
        var card = document.querySelector('[data-section="profile"]');
        if (!card) return;
        enterEdit(card);
        card.scrollIntoView({ behavior: 'smooth', block: 'start' });
      });
    }

    function applyProfile(card, data) {
      var avatar = document.querySelector('.avatar');
      if (avatar) {
        try {
          var avatarURL = data.avatar_url ? new URL(data.avatar_url, window.location.href) : null;
          if (avatarURL && avatarURL.origin === window.location.origin) {
            avatar.style.backgroundImage = 'url("' + avatarURL.href.replace(/"/g, '%22') + '")';
            avatar.classList.add('has-image');
            avatar.textContent = avatar.dataset.initials || '';
          } else {
            avatar.style.backgroundImage = '';
            avatar.classList.remove('has-image');
            avatar.textContent = avatar.dataset.initials || '?';
          }
        } catch (_) {
          avatar.style.backgroundImage = '';
          avatar.classList.remove('has-image');
        }
      }
      if (data.oshi_color && /^#[0-9a-fA-F]{6}$/.test(data.oshi_color)) {
        document.documentElement.style.setProperty('--oshi', data.oshi_color);
      }
      setDisplay(card, 'display_name', data.display_name || null);
      setDisplay(card, 'oshi_color', data.oshi_color || null);
      setDisplay(card, 'oshi_ids', (data.oshi_ids || []).length ? (data.oshi_ids || []).join(', ') : null);
      setDisplay(card, 'fan_since', data.fan_since || null);
      setDisplay(card, 'avatar_url', data.avatar_url || null);
      setDisplay(card, 'locale', data.locale || null);
      setDisplay(card, 'timezone', data.timezone || null);
      setDisplay(card, 'birthdate', data.birthdate || null);
      setDisplay(card, 'notification_preferences', notificationSummary(data.notification_preferences));
      var form = card.querySelector('form');
      if (!form) return;
      form.display_name.value = data.display_name || '';
      form.oshi_color.value = data.oshi_color || '';
      form.oshi_ids.value = (data.oshi_ids || []).join(',');
      form.fan_since.value = data.fan_since || '';
      var avatarInput = avatarFileInput(form);
      if (avatarInput) avatarInput.value = '';
      resetAvatarEditor();
      form.locale.value = data.locale || '';
      form.timezone.value = data.timezone || '';
      form.birthdate.value = data.birthdate || '';
      var prefs = data.notification_preferences || {};
      form.notify_email_enabled.checked = !!prefs.email_enabled;
      form.notify_product_updates.checked = !!prefs.product_updates;
      form.notify_security_alerts.checked = !!prefs.security_alerts;
      form.notify_community_notifications.checked = !!prefs.community_notifications;
      form.notify_app_membership_notifications.checked = !!prefs.app_membership_notifications;
    }

    document.getElementById('form-profile').addEventListener('submit', async function(e) {
      e.preventDefault();
      var card = this.closest('[data-section]');
      var errEl = document.getElementById('profile-error');
      errEl.textContent = '';
      try {
        var data = await req('PATCH', '/v1/account/profile', {
          display_name: this.display_name.value,
          oshi_color: this.oshi_color.value,
          oshi_ids: splitCSV(this.oshi_ids.value),
          fan_since: this.fan_since.value,
          locale: this.locale.value,
          timezone: this.timezone.value,
          birthdate: this.birthdate.value,
          notification_preferences: {
            email_enabled: this.notify_email_enabled.checked,
            product_updates: this.notify_product_updates.checked,
            security_alerts: this.notify_security_alerts.checked,
            community_notifications: this.notify_community_notifications.checked,
            app_membership_notifications: this.notify_app_membership_notifications.checked
          }
        });
        var avatarInput = avatarFileInput(this);
        if (avatarInput && avatarInput.files.length) {
          var fd = new FormData();
          var editedAvatar = await exportEditedAvatarBlob();
          var uploadAvatar = editedAvatar || avatarInput.files[0];
          assertAvatarSize(uploadAvatar);
          fd.append('avatar', uploadAvatar, 'avatar.jpg');
          var res = await fetch('/v1/account/profile/avatar', {
            method: 'POST',
            credentials: 'same-origin',
            body: fd
          });
          var text = await res.text();
          data = text ? JSON.parse(text) : data;
          if (!res.ok) throw new Error(data && data.error ? data.error : 'avatar upload failed: ' + res.status);
        }
        applyProfile(card, data);
        showToast('プロフィールを保存しました');
        card.dataset.mode = 'display';
      } catch(err) {
        errEl.textContent = err.message;
        showError(err);
      }
    });

    (function() {
      var form = document.getElementById('form-profile');
      var input = avatarFileInput(form);
      if (!input) return;
      var els = avatarEditorElements();
      input.addEventListener('change', function() {
        loadAvatarEditorFile(this.files && this.files.length ? this.files[0] : null);
      });
      [els.zoom, els.cropX, els.cropY].forEach(function(input) {
        if (!input) return;
        input.addEventListener('input', renderAvatarEditor);
      });
      var rotateLeft = document.getElementById('avatar-rotate-left');
      var rotateRight = document.getElementById('avatar-rotate-right');
      var resetButton = document.getElementById('avatar-reset-edit');
      if (rotateLeft) {
        rotateLeft.addEventListener('click', function() {
          avatarEditState.rotation = (avatarEditState.rotation + 270) % 360;
          renderAvatarEditor();
        });
      }
      if (rotateRight) {
        rotateRight.addEventListener('click', function() {
          avatarEditState.rotation = (avatarEditState.rotation + 90) % 360;
          renderAvatarEditor();
        });
      }
      if (resetButton) {
        resetButton.addEventListener('click', function() {
          if (els.zoom) els.zoom.value = '1';
          if (els.cropX) els.cropX.value = '0';
          if (els.cropY) els.cropY.value = '0';
          avatarEditState.rotation = 0;
          renderAvatarEditor();
        });
      }
    })();

    function renderMemberships(items) {
      var root = document.getElementById('memberships');
      root.innerHTML = '';
      if (!items || items.length === 0) {
        root.innerHTML = '<div class="empty-state">連携中アプリはありません</div>';
        return;
      }
      items.forEach(function(item) {
        var row = document.createElement('div');
        row.className = 'service-card';
        row.innerHTML =
          '<div style="min-width:0;flex:1">' +
            '<span class="status-pill">' + item.status + '</span>' +
            '<h4>' + item.app_name + '</h4>' +
            '<p>slug: ' + item.app_slug + ' / party: ' + item.party_type + '</p>' +
          '</div>' +
          '<button class="btn btn-secondary btn-sm" data-app-id="' + item.app_id + '">連携解除</button>';
        row.querySelector('button').addEventListener('click', async function() {
          if (!confirm(item.app_name + ' との連携を解除しますか？')) return;
          try {
            await req('DELETE', '/v1/account/apps/' + item.app_id);
            showToast('連携を解除しました');
            await loadOverview();
          } catch(err) { showError(err); }
        });
        root.appendChild(row);
      });
    }

    function renderDeletion(deletion) {
      var st = document.getElementById('deletion-status');
      var ste = document.getElementById('deletion-status-edit');
      if (!deletion) {
        st.textContent = '削除予約はありません。';
        if (ste) { ste.textContent = ''; ste.style.display = 'none'; }
        return;
      }
      var msg = '状態: ' + deletion.status + ' / 実行予定: ' + new Date(deletion.scheduled_for).toLocaleString('ja-JP');
      st.textContent = msg;
      if (ste) { ste.textContent = msg; ste.style.display = 'block'; }
    }

    document.getElementById('btn-schedule').addEventListener('click', async function() {
      var reason = document.getElementById('delete-reason').value.trim() || 'user_requested';
      try {
        await req('POST', '/v1/account/deletion', { reason: reason });
        showToast('削除を予約しました');
        await loadOverview();
      } catch(err) { showError(err); }
    });

    document.getElementById('btn-cancel-del').addEventListener('click', async function() {
      try {
        await req('DELETE', '/v1/account/deletion');
        showToast('削除予約を取り消しました');
        await loadOverview();
      } catch(err) { showError(err); }
    });

    async function loadOverview() {
      var data = await req('GET', '/v1/account');
      renderMemberships(data.memberships || []);
      renderDeletion(data.deletion_request || null);
    }

    async function loadProfile() {
      var data = await req('GET', '/v1/account/profile');
      var card = document.querySelector('[data-section="profile"]');
      if (card) applyProfile(card, data);
    }

    (function() {
      var root = document.documentElement;
      var btn = document.getElementById('theme-toggle');
      var stored = null;
      try { stored = window.localStorage.getItem('oshi-theme'); } catch(_) {}
      if (stored === 'dark' || stored === 'light') root.setAttribute('data-theme', stored);
      function syncIcon() {
        btn.textContent = root.getAttribute('data-theme') === 'dark' ? '☀' : '☾';
      }
      syncIcon();
      btn.addEventListener('click', function() {
        var next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
        root.setAttribute('data-theme', next);
        try { window.localStorage.setItem('oshi-theme', next); } catch(_) {}
        syncIcon();
      });
    })();

    (function() {
      var dots = document.querySelectorAll('.color-dot');
      var stored = null;
      try { stored = window.localStorage.getItem('oshi-color'); } catch(_) {}
      if (stored) {
        document.documentElement.style.setProperty('--oshi', stored);
        dots.forEach(function(d) {
          d.setAttribute('aria-pressed', d.dataset.color === stored ? 'true' : 'false');
        });
      }
      dots.forEach(function(d) {
        d.addEventListener('click', function() {
          var c = d.dataset.color;
          document.documentElement.style.setProperty('--oshi', c);
          try { window.localStorage.setItem('oshi-color', c); } catch(_) {}
          dots.forEach(function(x) { x.setAttribute('aria-pressed', x === d ? 'true' : 'false'); });
        });
      });
    })();

    Promise.all([loadOverview(), loadProfile()]).catch(showError);
  </script>
</body>
</html>
`))
