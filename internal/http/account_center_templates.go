package http

import "html/template"

var accountCenterTpl = template.Must(template.New("account-center").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>アカウントセンター</title>
  <style>
    *,*::before,*::after{box-sizing:border-box}
    body{margin:0;font-family:"Hiragino Sans","Yu Gothic","Noto Sans JP",system-ui,sans-serif;background:#f0f3fa;color:#1b2440}
    .profile-header{background:#fff;border-bottom:1px solid #e4e9f5;margin-bottom:24px}
    .cover{height:110px;background:linear-gradient(135deg,{{.OshiColor}} 0%,#6390f5 100%);opacity:.8}
    .profile-bar{display:flex;justify-content:space-between;align-items:flex-end;padding:0 28px;margin-top:-36px}
    .avatar{width:72px;height:72px;border-radius:50%;background:{{.OshiColor}};color:#fff;font-size:26px;font-weight:800;display:flex;align-items:center;justify-content:center;border:3px solid #fff;box-shadow:0 4px 14px rgba(0,0,0,.15);letter-spacing:.02em;flex-shrink:0;background-size:cover;background-position:center}
    .avatar.has-image{color:transparent}
    .profile-identity{padding:10px 28px 20px;display:flex;align-items:baseline;gap:14px;flex-wrap:wrap}
    .profile-identity h1{margin:0;font-size:22px;font-weight:800;line-height:1.1}
    .handle{font-size:12px;color:#8898b8;font-weight:500;word-break:break-all}
    .shell{max-width:1040px;margin:0 auto;padding:0 20px 80px}
    .grid{display:grid;grid-template-columns:1.1fr .9fr;gap:18px}
    .col{display:flex;flex-direction:column;gap:18px}
    .card{background:#fff;border:1px solid #e4e9f5;border-radius:20px;padding:20px;box-shadow:0 2px 8px rgba(18,42,88,.05)}
    .card-head{display:flex;justify-content:space-between;align-items:center;margin-bottom:14px}
    .card-head h2{margin:0;font-size:15px;font-weight:700;color:#1b2440}
    .card-danger{border-color:#f3d1c9;background:linear-gradient(160deg,#fff5f2 0%,#fff 60%)}
    .card-danger .card-head h2{color:#9f2417}
    .field-row{display:flex;align-items:flex-start;gap:14px;padding:9px 0;border-bottom:1px solid #f0f3fa}
    .field-row:last-of-type{border-bottom:none}
    .field-label{min-width:96px;font-size:11px;font-weight:700;color:#8898b8;text-transform:uppercase;letter-spacing:.07em;padding-top:2px;flex-shrink:0}
    .field-value{font-size:14px;color:#1b2440;flex:1;word-break:break-all}
    .empty{color:#c0cadc;font-style:italic}
    [data-mode="display"] .section-edit{display:none}
    [data-mode="edit"] .section-display{display:none}
    .section-edit{padding-top:6px}
    .field{display:grid;gap:5px;margin-bottom:12px}
    .field label{font-size:11px;font-weight:700;color:#65748f;text-transform:uppercase;letter-spacing:.07em}
    .field input{width:100%;border:1px solid #d9e0f0;border-radius:12px;padding:10px 12px;font-size:14px;background:#fff;font-family:inherit;color:#1b2440}
    .field input[type="checkbox"]{width:18px;height:18px;padding:0;border-radius:5px;accent-color:#1740c9;flex-shrink:0}
    .field input:focus{outline:2px solid #1740c9;outline-offset:1px;border-color:#1740c9}
    .field-note{font-size:11px;color:#8898b8;line-height:1.45}
    .check-grid{display:grid;gap:8px;margin:4px 0 14px}
    .check-row{display:flex;align-items:center;gap:9px;font-size:13px;color:#4d5c76}
    .check-row span{line-height:1.4}
    .form-error{font-size:13px;color:#c73a2b;min-height:18px;margin-bottom:6px}
    .form-actions{display:flex;gap:8px;margin-top:4px}
    .btn{appearance:none;border:none;border-radius:12px;padding:9px 14px;font-size:13px;font-weight:700;cursor:pointer;text-decoration:none;display:inline-flex;align-items:center;font-family:inherit;transition:opacity .15s}
    .btn:hover{opacity:.82}
    .btn-primary{background:#1740c9;color:#fff}
    .btn-ghost{background:#fff;border:1px solid rgba(23,64,201,.22);color:#1740c9}
    .btn-danger{background:#c73a2b;color:#fff}
    .btn-danger-ghost{color:#9f2417;border-color:rgba(199,58,43,.28)}
    .btn-sm{padding:6px 10px;font-size:12px;border-radius:10px}
    .hint{font-size:13px;color:#65748f;line-height:1.6;margin:0 0 12px}
    .status{font-size:13px;color:#4d5c76;margin-top:8px;line-height:1.5}
    .contact-footer{padding-top:14px;display:flex;align-items:center;gap:12px;flex-wrap:wrap;border-top:1px solid #f0f3fa;margin-top:4px}
    .contact-note{font-size:11px;color:#8898b8;line-height:1.4}
    .membership{display:flex;justify-content:space-between;align-items:flex-start;gap:12px;padding:12px;border:1px solid #e8edf8;border-radius:14px;background:#fbfcff;margin-bottom:10px}
    .membership:last-child{margin-bottom:0}
    .membership h3{margin:0 0 3px;font-size:14px;font-weight:700}
    .membership p{margin:0;color:#65748f;font-size:12px;line-height:1.5}
    .badge{display:inline-block;padding:3px 8px;border-radius:999px;background:#edf7ef;color:#13643a;font-size:11px;font-weight:700;margin-bottom:5px}
    .empty-state{text-align:center;padding:20px 0;color:#b0bdd4;font-size:13px}
    .toast{position:fixed;right:20px;bottom:20px;background:#162a5c;color:#fff;padding:12px 16px;border-radius:14px;display:none;max-width:320px;box-shadow:0 18px 40px rgba(14,24,54,.24);font-size:14px;z-index:100}
    .toast.show{display:block}
    @media(max-width:840px){.grid{grid-template-columns:1fr}.profile-bar{margin-top:-28px}.avatar{width:60px;height:60px;font-size:20px}.profile-identity h1{font-size:19px}}
  </style>
</head>
<body>

  <header class="profile-header">
    <div class="cover"></div>
    <div class="profile-bar">
      <div class="avatar" data-initials="{{.Initials}}">{{.Initials}}</div>
      <div style="padding-bottom:6px">
        <a class="btn btn-ghost btn-sm" href="{{.LogoutURL}}">ログアウト</a>
      </div>
    </div>
    <div class="profile-identity">
      <h1>{{if .DisplayName}}{{.DisplayName}}{{else}}アカウント{{end}}</h1>
      <span class="handle">{{.IdentityID}}</span>
    </div>
  </header>

  <main class="shell">
    <div class="grid">
      <div class="col">

        <article class="card">
          <div class="card-head"><h2>連絡先情報</h2></div>
          <div class="field-row">
            <span class="field-label">メール</span>
            <span class="field-value">{{if .Email}}{{.Email}}{{else}}<span class="empty">未登録</span>{{end}}</span>
          </div>
          <div class="field-row">
            <span class="field-label">電話番号</span>
            <span class="field-value">{{if .Phone}}{{.Phone}}{{else}}<span class="empty">未登録</span>{{end}}</span>
          </div>
          <div class="contact-footer">
            <a href="{{.KratosSettingsURL}}" class="btn btn-ghost btn-sm">連絡先を変更 →</a>
            <span class="contact-note">メール・電話番号の変更はKratosの設定画面で行います</span>
          </div>
        </article>

        <article class="card" data-section="profile" data-mode="display">
          <div class="card-head">
            <h2>共有プロフィール</h2>
            <button class="btn btn-ghost btn-sm" data-action="edit">編集</button>
          </div>
          <div class="section-display">
            <p class="hint" style="margin-bottom:8px">各アプリは連携済みユーザーに対してこのプロフィールを参照できます。</p>
            <div class="field-row"><span class="field-label">表示名</span><span class="field-value" data-display="display_name"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">推し色</span><span class="field-value" data-display="oshi_color"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">推しID</span><span class="field-value" data-display="oshi_ids"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">ファン歴</span><span class="field-value" data-display="fan_since"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">アバター</span><span class="field-value" data-display="avatar_url"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">ロケール</span><span class="field-value" data-display="locale"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">タイムゾーン</span><span class="field-value" data-display="timezone"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">生年月日</span><span class="field-value" data-display="birthdate"><span class="empty">—</span></span></div>
            <div class="field-row"><span class="field-label">通知設定</span><span class="field-value" data-display="notification_preferences"><span class="empty">—</span></span></div>
          </div>
          <form class="section-edit" id="form-profile">
            <div class="field"><label>表示名<input name="display_name" type="text" maxlength="50"></label></div>
            <div class="field"><label>推し色（#rrggbb）<input name="oshi_color" type="text" placeholder="#ff88cc"></label></div>
            <div class="field"><label>推しID（カンマ区切り、最大10件）<input name="oshi_ids" type="text"></label></div>
            <div class="field"><label>ファン歴（YYYY または YYYY-MM）<input name="fan_since" type="text" placeholder="2022-03"></label></div>
            <div class="field"><label>アバター画像<input name="avatar_file" type="file" accept="image/png,image/jpeg,image/gif,image/webp"></label><div class="field-note">PNG / JPEG / GIF / WebP、10MBまで。保存時に512px四方のJPEGへ変換されます。</div></div>
            <div class="field"><label>ロケール<input name="locale" type="text" placeholder="ja-JP"></label></div>
            <div class="field"><label>タイムゾーン<input name="timezone" type="text" placeholder="Asia/Tokyo"></label></div>
            <div class="field"><label>生年月日<input name="birthdate" type="date"></label><div class="field-note">本人向けアカウント情報として保存します。連携アプリ向け公開プロフィールには含まれません。</div></div>
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
              <button type="button" class="btn btn-ghost" data-action="cancel">キャンセル</button>
            </div>
          </form>
        </article>

      </div>
      <div class="col">

        <article class="card">
          <div class="card-head"><h2>連携中アプリ</h2></div>
          <p class="hint">各アプリでアカウント削除をする場合は、まずここで連携を解除してください。</p>
          <div id="memberships"></div>
        </article>

        <article class="card card-danger" data-section="deletion" data-mode="display">
          <div class="card-head">
            <h2>アカウント削除</h2>
            <button class="btn btn-ghost btn-sm btn-danger-ghost" data-action="edit">予約・管理</button>
          </div>
          <div class="section-display">
            <p class="hint"><strong>注意:</strong> 削除予約すると連携中の全アプリに影響します。各アプリだけをやめたい場合は連携解除を使ってください。</p>
            <div id="deletion-status" class="status">削除予約はありません。</div>
          </div>
          <div class="section-edit">
            <p class="hint">削除予約するか、予約中の場合は取り消せます。</p>
            <div class="field"><label>理由（省略可）<input id="delete-reason" type="text" placeholder="user_requested"></label></div>
            <div class="form-actions">
              <button class="btn btn-danger btn-sm" type="button" id="btn-schedule">削除を予約</button>
              <button class="btn btn-ghost btn-sm" type="button" id="btn-cancel-del">予約を取り消す</button>
            </div>
            <div id="deletion-status-edit" class="status"></div>
          </div>
        </article>

      </div>
    </div>
  </main>

  <div id="toast" class="toast"></div>
  <script>
    function showToast(msg) {
      var el = document.getElementById('toast');
      el.textContent = msg;
      el.classList.add('show');
      clearTimeout(window.__tt);
      window.__tt = setTimeout(function(){ el.classList.remove('show'); }, 3200);
    }

    function splitCSV(v) {
      return v.split(',').map(function(s){ return s.trim(); }).filter(Boolean);
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
      form.avatar_file.value = '';
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
        if (this.avatar_file.files.length) {
          var fd = new FormData();
          fd.append('avatar', this.avatar_file.files[0]);
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
      } catch(err) { errEl.textContent = err.message; }
    });

    function renderMemberships(items) {
      var root = document.getElementById('memberships');
      root.innerHTML = '';
      if (!items || items.length === 0) {
        root.innerHTML = '<div class="empty-state">連携中アプリはありません</div>';
        return;
      }
      items.forEach(function(item) {
        var row = document.createElement('div');
        row.className = 'membership';
        row.innerHTML =
          '<div><span class="badge">' + item.status + '</span>' +
          '<h3>' + item.app_name + '</h3>' +
          '<p>slug: ' + item.app_slug + ' / party: ' + item.party_type + '</p></div>' +
          '<button class="btn btn-ghost btn-sm" data-app-id="' + item.app_id + '">連携解除</button>';
        row.querySelector('button').addEventListener('click', async function() {
          if (!confirm(item.app_name + ' との連携を解除しますか？')) return;
          try {
            await req('DELETE', '/v1/account/apps/' + item.app_id);
            showToast('連携を解除しました');
            await loadOverview();
          } catch(err) { showToast(err.message); }
        });
        root.appendChild(row);
      });
    }

    function renderDeletion(deletion) {
      var st = document.getElementById('deletion-status');
      var ste = document.getElementById('deletion-status-edit');
      if (!deletion) {
        st.textContent = '削除予約はありません。';
        if (ste) ste.textContent = '';
        return;
      }
      var msg = '状態: ' + deletion.status + ' / 実行予定: ' + new Date(deletion.scheduled_for).toLocaleString('ja-JP');
      st.textContent = msg;
      if (ste) ste.textContent = msg;
    }

    document.getElementById('btn-schedule').addEventListener('click', async function() {
      var reason = document.getElementById('delete-reason').value.trim() || 'user_requested';
      try {
        await req('POST', '/v1/account/deletion', { reason: reason });
        showToast('削除を予約しました');
        await loadOverview();
      } catch(err) { showToast(err.message); }
    });

    document.getElementById('btn-cancel-del').addEventListener('click', async function() {
      try {
        await req('DELETE', '/v1/account/deletion');
        showToast('削除予約を取り消しました');
        await loadOverview();
      } catch(err) { showToast(err.message); }
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

    Promise.all([loadOverview(), loadProfile()]).catch(function(err){ showToast(err.message); });
  </script>
</body>
</html>
`))
