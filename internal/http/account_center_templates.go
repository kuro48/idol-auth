package http

import "html/template"

var accountCenterTpl = template.Must(template.New("account-center").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Account Center</title>
  <style>
    *,*::before,*::after{box-sizing:border-box}body{margin:0;font-family:"Hiragino Sans","Yu Gothic","Noto Sans JP",system-ui,sans-serif;background:linear-gradient(180deg,#fffef7 0%,#f4f8ff 100%);color:#1b2440}
    .shell{max-width:1120px;margin:0 auto;padding:32px 20px 80px}
    .hero{display:flex;justify-content:space-between;gap:20px;align-items:flex-start;margin-bottom:24px}
    .hero-card,.panel{background:rgba(255,255,255,.92);border:1px solid rgba(18,42,88,.08);border-radius:24px;box-shadow:0 20px 50px rgba(25,48,99,.08)}
    .hero-card{padding:28px 28px 24px;flex:1}
    .eyebrow{display:inline-block;padding:6px 10px;border-radius:999px;background:#ebf2ff;color:#1840aa;font-size:11px;font-weight:700;letter-spacing:.08em;text-transform:uppercase}
    h1{margin:14px 0 10px;font-size:34px;line-height:1.05}
    .sub{margin:0;color:#53627f;line-height:1.7}
    .hero-meta{min-width:260px;padding:22px}
    .meta-line{font-size:13px;color:#53627f;margin-bottom:10px}
    .meta-strong{display:block;color:#1b2440;font-size:15px;font-weight:700}
    .actions{display:flex;gap:12px;flex-wrap:wrap;margin-top:18px}
    .btn{appearance:none;border:none;border-radius:14px;padding:12px 16px;font-size:14px;font-weight:700;cursor:pointer}
    .btn-primary{background:#1740c9;color:#fff}.btn-secondary{background:#eef3ff;color:#1f3d8d}.btn-danger{background:#c73a2b;color:#fff}.btn-ghost{background:#fff;border:1px solid rgba(23,64,201,.18);color:#1740c9}
    .grid{display:grid;grid-template-columns:1.15fr .85fr;gap:18px}.stack{display:grid;gap:18px}
    .panel{padding:22px}
    .panel h2{margin:0 0 14px;font-size:18px}
    .hint{font-size:13px;color:#65748f;line-height:1.6}
    .field{display:grid;gap:6px;margin-bottom:14px}
    .field label{font-size:12px;font-weight:700;color:#53627f;text-transform:uppercase;letter-spacing:.06em}
    .field input{width:100%;border:1px solid #d9e0f0;border-radius:14px;padding:12px 14px;font-size:14px;background:#fff}
    .table{display:grid;gap:12px}
    .membership{display:flex;justify-content:space-between;gap:12px;align-items:flex-start;padding:16px;border:1px solid #e3e9f5;border-radius:18px;background:#fbfcff}
    .membership h3{margin:0 0 6px;font-size:16px}
    .membership p{margin:0;color:#65748f;font-size:13px}
    .badge{display:inline-block;padding:4px 9px;border-radius:999px;background:#edf7ef;color:#13643a;font-size:11px;font-weight:700;margin-bottom:8px}
    .danger{background:linear-gradient(180deg,#fff5f2 0%,#fff 100%);border-color:#f3d1c9}
    .danger strong{color:#9f2417}
    .status{margin-top:10px;font-size:13px;color:#4d5c76}
    .toast{position:fixed;right:20px;bottom:20px;background:#162a5c;color:#fff;padding:12px 14px;border-radius:14px;display:none;max-width:320px;box-shadow:0 18px 40px rgba(14,24,54,.24)}
    .toast.show{display:block}
    @media (max-width: 900px){.grid{grid-template-columns:1fr}.hero{flex-direction:column}.hero-meta{width:100%}}
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="hero-card">
        <span class="eyebrow">Shared Account</span>
        <h1>Account Center</h1>
        <p class="sub">このページでは共有アカウント本体のプロフィール管理、連携中アプリの確認、連携解除、共有アカウント削除予約を行います。</p>
        <div class="actions">
          <button class="btn btn-primary" onclick="saveProfile()">プロフィールを保存</button>
          <button class="btn btn-secondary" onclick="reloadAll()">最新状態を再読込</button>
          <a class="btn btn-ghost" href="{{.LogoutURL}}">ログアウト</a>
        </div>
      </div>
      <aside class="hero-card hero-meta">
        <div class="meta-line">ログイン中の共有アカウント<span class="meta-strong">{{.Email}}</span></div>
        <div class="meta-line">Identity ID<span class="meta-strong" id="identity-id">{{.IdentityID}}</span></div>
        <div class="meta-line">利用方法<span class="meta-strong">各アプリはこのアカウントでログインし、必要な公開プロフィールだけ参照します。</span></div>
      </aside>
    </section>

    <section class="grid">
      <div class="stack">
        <div class="panel">
          <h2>共有プロフィール</h2>
          <p class="hint">ここで編集した内容は、共有アカウントのプロフィールとして保存されます。各アプリは連携済みユーザーに対して公開プロフィールだけ参照できます。</p>
          <div class="field"><label for="display-name">Display Name</label><input id="display-name" type="text" placeholder="推し活太郎"></div>
          <div class="field"><label for="oshi-color">Oshi Color</label><input id="oshi-color" type="text" placeholder="#ffb2d8"></div>
          <div class="field"><label for="oshi-ids">Oshi IDs</label><input id="oshi-ids" type="text" placeholder="idol-1,idol-2"></div>
          <div class="field"><label for="fan-since">Fan Since</label><input id="fan-since" type="text" placeholder="2020-04"></div>
        </div>

        <div class="panel">
          <h2>連携中アプリ</h2>
          <p class="hint">各アプリでアカウント削除をする場合は、そのアプリのローカルデータ削除に加えて、ここに表示されている shared account との連携を解除します。</p>
          <div id="memberships" class="table"></div>
        </div>
      </div>

      <div class="stack">
        <div class="panel danger">
          <h2>共有アカウント削除</h2>
          <p class="hint"><strong>注意:</strong> ここで削除予約すると、連携中の全アプリに影響します。各アプリだけをやめたい場合は、左側の連携解除を使ってください。</p>
          <div class="field"><label for="delete-reason">Reason</label><input id="delete-reason" type="text" placeholder="user_requested"></div>
          <div class="actions">
            <button class="btn btn-danger" onclick="scheduleDeletion()">削除を予約</button>
            <button class="btn btn-ghost" onclick="cancelDeletion()">削除予約を取り消す</button>
          </div>
          <div class="status" id="deletion-status">削除予約はありません。</div>
        </div>

        <div class="panel">
          <h2>このページの役割</h2>
          <p class="hint">shared account 本体の管理はこのサービスだけで行います。各アプリ側は Google / Microsoft ログインのようにこのアカウントで sign-in し、アプリごとに必要な範囲だけプロフィールを参照する構成です。</p>
        </div>
      </div>
    </section>
  </div>
  <div id="toast" class="toast"></div>
  <script>
    function showToast(message){
      var el=document.getElementById('toast');
      el.textContent=message;
      el.classList.add('show');
      clearTimeout(window.__toastTimer);
      window.__toastTimer=setTimeout(function(){el.classList.remove('show');},3200);
    }

    function splitCSV(value){
      return value.split(',').map(function(v){return v.trim();}).filter(Boolean);
    }

    async function request(method, path, body){
      var opts={method:method,credentials:'same-origin',headers:{}};
      if(body!==undefined){
        opts.headers['Content-Type']='application/json';
        opts.body=JSON.stringify(body);
      }
      var res=await fetch(path,opts);
      var text=await res.text();
      var data=text?JSON.parse(text):null;
      if(!res.ok){
        throw new Error(data&&data.error?data.error:('request failed: '+res.status));
      }
      return data;
    }

    function renderMemberships(items){
      var root=document.getElementById('memberships');
      root.innerHTML='';
      if(!items||items.length===0){
        root.innerHTML='<div class="membership"><div><h3>連携中アプリはありません</h3><p>この shared account でログインしたアプリがここに並びます。</p></div></div>';
        return;
      }
      items.forEach(function(item){
        var row=document.createElement('div');
        row.className='membership';
        row.innerHTML='' +
          '<div>' +
          '<div class="badge">'+item.status+'</div>' +
          '<h3>'+item.app_name+'</h3>' +
          '<p>slug: '+item.app_slug+'</p>' +
          '<p>party: '+item.party_type+'</p>' +
          '</div>' +
          '<div><button class="btn btn-ghost" data-app-id="'+item.app_id+'">連携解除</button></div>';
        row.querySelector('button').addEventListener('click', async function(){
          if(!confirm(item.app_name+' との連携を解除しますか？')) return;
          await request('DELETE','/v1/account/apps/'+item.app_id);
          showToast('連携を解除しました');
          await reloadAll();
        });
        root.appendChild(row);
      });
    }

    function renderDeletion(request){
      var status=document.getElementById('deletion-status');
      if(!request){
        status.textContent='削除予約はありません。';
        return;
      }
      status.textContent='状態: '+request.status+' / 実行予定: '+new Date(request.scheduled_for).toLocaleString();
    }

    async function loadOverview(){
      var data=await request('GET','/v1/account');
      renderMemberships(data.memberships||[]);
      renderDeletion(data.deletion_request||null);
    }

    async function loadProfile(){
      var data=await request('GET','/v1/account/profile');
      document.getElementById('display-name').value=data.display_name||'';
      document.getElementById('oshi-color').value=data.oshi_color||'';
      document.getElementById('oshi-ids').value=(data.oshi_ids||[]).join(',');
      document.getElementById('fan-since').value=data.fan_since||'';
    }

    async function saveProfile(){
      var payload={
        display_name:document.getElementById('display-name').value,
        oshi_color:document.getElementById('oshi-color').value,
        oshi_ids:splitCSV(document.getElementById('oshi-ids').value),
        fan_since:document.getElementById('fan-since').value
      };
      await request('PATCH','/v1/account/profile',payload);
      showToast('プロフィールを保存しました');
      await reloadAll();
    }

    async function scheduleDeletion(){
      var reason=document.getElementById('delete-reason').value.trim()||'user_requested';
      await request('POST','/v1/account/deletion',{reason:reason});
      showToast('共有アカウント削除を予約しました');
      await loadOverview();
    }

    async function cancelDeletion(){
      await request('DELETE','/v1/account/deletion');
      showToast('削除予約を取り消しました');
      await loadOverview();
    }

    async function reloadAll(){
      await Promise.all([loadOverview(), loadProfile()]);
    }

    reloadAll().catch(function(err){showToast(err.message);});
  </script>
</body>
</html>
`))
