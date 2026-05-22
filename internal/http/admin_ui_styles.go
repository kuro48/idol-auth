package http

// adminUIStyles holds the OshiLink design-system CSS used by the admin dashboard
// templates. Split from admin_ui_templates.go to keep both files under 800 lines.
// Consumed in admin_ui_templates.go via template.New(...).Parse(adminUIStyles + adminUITemplates).
const adminUIStyles = `
{{define "head"}}
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --oshi:#f472b6; --oshi-weak:#fce7f3; --oshi-soft:#fbcfe8; --bg:#fdf8fc; --surface:#fff;
  --surface-2:#fef9fd; --text:#2a1520; --muted:#9d748f; --border:#f0d0e8;
  --radius-lg:32px; --radius-md:22px;
  --danger:#e85d75; --success:#10b981; --warning:#e9a23b; --sw:280px;
}
html{color-scheme:light}
@media (prefers-color-scheme: dark){
  :root{
    --bg:#1a1018; --surface:#251820; --surface-2:#2e1f28; --text:#fff0f8;
    --muted:#c8a8bd; --border:#4a2e40; --oshi-weak:#3d1a2a; --oshi-soft:#5a2a40;
  }
  html{color-scheme:dark}
}
body{font-family:"Hiragino Sans","Yu Gothic","Noto Sans JP",ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,sans-serif;font-size:14px;background:var(--bg);color:var(--text);min-height:100vh;letter-spacing:.01em}
button,input,select,textarea{font:inherit;color:inherit}
button{cursor:pointer}
a{color:var(--oshi);text-decoration:none}a:hover{text-decoration:underline}
code{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}

.topnav{position:fixed;top:0;left:0;right:0;z-index:100;height:64px;background:var(--surface-2);border-bottom:2px solid var(--border);display:flex;align-items:center;padding:0 24px;gap:14px;backdrop-filter:blur(10px)}
.topnav-mark{width:38px;height:38px;border-radius:13px;background:var(--oshi);color:#fff;display:grid;place-items:center;font-size:18px;font-weight:900;border:2px solid color-mix(in srgb,var(--oshi) 35%,transparent)}
.topnav-brand{display:flex;flex-direction:column;line-height:1.1}
.topnav-brand strong{font-size:15px;font-weight:900;letter-spacing:.02em}
.topnav-brand span{font-size:11px;color:var(--muted);font-weight:700;letter-spacing:.08em;text-transform:uppercase}
.topnav-spacer{flex:1}
.topnav-email{font-size:12px;color:var(--muted);font-weight:700;padding:8px 14px;background:var(--surface);border:2px solid var(--border);border-radius:999px}

.layout{display:flex;padding-top:64px;min-height:100vh}
.sidebar{position:fixed;top:64px;left:0;bottom:0;width:var(--sw);background:var(--surface-2);border-right:2px solid var(--border);padding:24px 18px;display:flex;flex-direction:column;gap:22px;overflow-y:auto}
.brand{display:flex;align-items:center;gap:12px;padding:4px 6px}
.brand-mark{width:44px;height:44px;border-radius:16px;background:var(--oshi);display:grid;place-items:center;color:#fff;font-size:22px;border:2px solid color-mix(in srgb,var(--oshi) 35%,transparent)}
.brand-text strong{display:block;font-size:17px;font-weight:900;line-height:1.1}
.brand-text span{font-size:11px;color:var(--muted);letter-spacing:.06em;text-transform:uppercase;font-weight:700}
.nav-section-title{margin:6px 10px 4px;color:var(--muted);font-size:11px;font-weight:800;text-transform:uppercase;letter-spacing:.1em}
.nav-list{display:grid;gap:6px}
.nav-link{width:100%;border:0;color:var(--muted);background:transparent;padding:11px 12px;border-radius:16px;text-align:left;display:flex;align-items:center;gap:11px;font-weight:700;text-decoration:none;transition:transform 160ms ease,background 160ms ease,color 160ms ease;outline:2px solid transparent;outline-offset:-2px}
.nav-link:hover{transform:translateX(2px);background:color-mix(in srgb,var(--oshi) 10%,transparent);color:var(--text);text-decoration:none}
.nav-link.active{color:var(--text);background:var(--surface);outline:2px solid color-mix(in srgb,var(--oshi) 38%,transparent)}
.nav-icon{width:28px;height:28px;border-radius:10px;background:var(--oshi-weak);display:grid;place-items:center;color:var(--oshi);font-size:14px;flex-shrink:0}

.sidebar-footer{margin-top:auto;padding:14px;background:var(--surface);border:2px solid var(--border);border-radius:20px}
.color-dots{display:flex;gap:8px;flex-wrap:wrap}
.color-dot{width:26px;height:26px;border-radius:50%;border:2px solid var(--border);cursor:pointer;padding:0;transition:transform 160ms ease}
.color-dot:hover{transform:scale(1.1)}

.main{margin-left:var(--sw);flex:1;padding:32px 36px 48px;min-width:0}
.page-header{display:flex;align-items:flex-start;justify-content:space-between;gap:18px;margin-bottom:28px;flex-wrap:wrap}
.page-title-block h1{margin:0;font-size:clamp(24px,2.4vw,34px);line-height:1.1;letter-spacing:-0.03em;font-weight:900}
.page-title-block p{margin:8px 0 0;color:var(--muted);font-size:14px;line-height:1.6;max-width:560px}

.metric-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:18px;margin-bottom:28px}
.metric-card{padding:20px;border-radius:24px;border:2px solid var(--border);background:var(--surface)}
.metric-top{display:flex;align-items:center;justify-content:space-between;gap:10px}
.metric-top span{color:var(--muted);font-weight:800;font-size:13px}
.metric-icon{width:42px;height:42px;border-radius:15px;display:grid;place-items:center;background:var(--oshi-weak);color:var(--oshi);font-size:18px}
.metric-card strong{display:block;margin-top:16px;font-size:32px;letter-spacing:-0.04em;font-weight:900;line-height:1}
.metric-card p{margin:6px 0 0;color:var(--muted);font-weight:700;font-size:12px}

.card{background:var(--surface);border:2px solid var(--border);border-radius:var(--radius-lg);padding:24px 26px;margin-bottom:22px}
.card-head{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:16px;flex-wrap:wrap}
.card-head h3{margin:0;font-size:18px;font-weight:900;letter-spacing:-0.01em}
.card-head p{margin:4px 0 0;color:var(--muted);font-size:13px}
.card-title{font-size:11px;font-weight:900;letter-spacing:.1em;text-transform:uppercase;color:var(--muted);margin-bottom:14px}

.table-card{background:var(--surface);border:2px solid var(--border);border-radius:var(--radius-lg);overflow:hidden;margin-bottom:22px}
.table-card .table-head{padding:22px 24px 16px;display:flex;justify-content:space-between;gap:12px;align-items:center;flex-wrap:wrap}
.table-card .table-head h3{margin:0;font-size:18px;font-weight:900;letter-spacing:-0.01em}
.table-card .table-head p{margin:4px 0 0;color:var(--muted);font-size:13px}
.table-wrap{overflow-x:auto}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;padding:12px 24px;background:var(--surface-2);border-top:2px solid var(--border);border-bottom:2px solid var(--border);font-weight:800;color:var(--muted);font-size:11px;text-transform:uppercase;letter-spacing:.08em;white-space:nowrap}
td{padding:14px 24px;border-top:1px solid var(--border);vertical-align:middle}
tr:hover td{background:color-mix(in srgb,var(--oshi) 5%,transparent)}

.badge{display:inline-flex;align-items:center;gap:6px;padding:5px 11px;border-radius:999px;font-size:11px;font-weight:900;letter-spacing:.02em}
.badge-active,.badge-success{background:color-mix(in srgb,var(--success) 14%,transparent);color:var(--success)}
.badge-inactive,.badge-failure{background:color-mix(in srgb,var(--danger) 14%,transparent);color:var(--danger)}
.badge-rotated,.badge-changes{background:color-mix(in srgb,var(--warning) 16%,transparent);color:var(--warning)}
.badge-pending{background:color-mix(in srgb,var(--oshi) 14%,transparent);color:var(--oshi)}
.badge-review{background:color-mix(in srgb,#8e6be8 14%,transparent);color:#8e6be8}

.btn{display:inline-flex;align-items:center;gap:6px;padding:11px 18px;border-radius:999px;font-size:13px;font-weight:900;border:2px solid transparent;text-decoration:none;transition:transform 160ms ease,opacity 160ms ease,background 160ms ease;letter-spacing:.01em}
.btn:hover{transform:translateY(-1px);text-decoration:none}
.btn-primary{background:var(--oshi);color:#fff;border-color:var(--oshi)}
.btn-primary:hover{opacity:.88}
.btn-secondary{background:var(--surface-2);color:var(--text);border-color:var(--border)}
.btn-secondary:hover{background:var(--surface);border-color:color-mix(in srgb,var(--oshi) 40%,transparent)}
.btn-danger{background:var(--danger);color:#fff;border-color:var(--danger)}
.btn-approve{background:var(--success);color:#fff;border-color:var(--success)}
.btn-changes{background:var(--warning);color:#fff;border-color:var(--warning)}
.btn-sm{padding:7px 12px;font-size:12px}

.form-row{display:flex;gap:12px;align-items:flex-end;flex-wrap:wrap;margin-bottom:22px;padding:18px 22px;background:var(--surface);border:2px solid var(--border);border-radius:var(--radius-md)}
.form-group{display:flex;flex-direction:column;gap:6px}
.form-group label{font-size:12px;font-weight:900;color:var(--muted);letter-spacing:.02em}
input[type=text],input[type=password],input[type=email],select,textarea{padding:11px 14px;border:2px solid var(--border);border-radius:14px;font-size:13px;color:var(--text);background:var(--surface-2);min-width:180px;outline:none;transition:border-color 160ms ease}
input[type=text]:focus,input[type=password]:focus,input[type=email]:focus,select:focus,textarea:focus{border-color:var(--oshi);background:var(--surface)}

.modal-overlay{display:none;position:fixed;inset:0;background:color-mix(in srgb,#1a120e 55%,transparent);z-index:200;align-items:center;justify-content:center;backdrop-filter:blur(4px);padding:20px}
.modal-overlay.open{display:flex}
.modal{background:var(--surface);border:2px solid var(--border);border-radius:var(--radius-lg);padding:28px;width:540px;max-width:100%;max-height:90vh;overflow-y:auto}
.modal-title{font-size:20px;font-weight:900;margin-bottom:8px;letter-spacing:-0.02em}
.modal-desc{margin-bottom:18px;color:var(--muted);font-size:13px;line-height:1.7}
.modal-actions{display:flex;gap:10px;justify-content:flex-end;margin-top:22px;flex-wrap:wrap}
.modal .form-group{margin-bottom:14px}
.modal input[type=text],.modal input[type=password],.modal input[type=email],.modal select,.modal textarea{width:100%;min-width:0}
.modal textarea{min-height:96px;resize:vertical;font-family:inherit}

.toast-area{position:fixed;bottom:24px;right:24px;z-index:300;display:flex;flex-direction:column;gap:10px;max-width:calc(100% - 48px)}
.toast{padding:14px 20px;border-radius:16px;font-size:13px;font-weight:800;color:#fff;animation:slideIn .22s ease both}
.toast-success{background:var(--success)}
.toast-error{background:var(--danger)}
@keyframes slideIn{from{opacity:0;transform:translateY(8px)}to{opacity:1;transform:none}}

.empty-row td{text-align:center;color:var(--muted);padding:32px;font-weight:700}
.actions-cell{display:flex;gap:6px;flex-wrap:wrap}
.user-cell{display:flex;align-items:center;gap:12px}
.mini-face{width:36px;height:36px;border-radius:12px;background:var(--oshi-weak);color:var(--oshi);display:grid;place-items:center;font-weight:1000;font-size:13px;flex-shrink:0}
.user-cell strong{font-weight:800;font-size:13px}
.user-cell .user-sub{display:block;color:var(--muted);font-size:11px;margin-top:2px;font-weight:600}
.role-pill{display:inline-flex;padding:5px 11px;border-radius:999px;font-size:11px;font-weight:900;background:var(--surface-2);border:2px solid var(--border);color:var(--muted)}

.kv-table{width:100%;border-collapse:collapse;font-size:13px}
.kv-table th{width:200px;text-align:left;padding:13px 16px;background:var(--surface-2);border-bottom:2px solid var(--border);font-weight:800;color:var(--muted);vertical-align:top;white-space:nowrap;text-transform:none;letter-spacing:0;font-size:12px}
.kv-table td{padding:13px 16px;border-bottom:1px solid var(--border);vertical-align:top;word-break:break-all}
.kv-empty{color:var(--muted);opacity:.6}
.detail-actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:22px}
.detail-title-row{display:flex;align-items:center;gap:14px;flex-wrap:wrap}
.back-link{font-size:13px;color:var(--muted);margin-bottom:18px;display:inline-flex;align-items:center;gap:6px;font-weight:700}
.back-link:hover{color:var(--oshi);text-decoration:none}
.uri-list{margin:0;padding-left:18px;font-size:12px}
.uri-list li{margin-bottom:4px}

@media (max-width:1080px){
  :root{--sw:240px}
  .metric-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
}
@media (max-width:780px){
  .sidebar{position:static;width:100%;height:auto;border-right:0;border-bottom:2px solid var(--border);padding:16px}
  .main{margin-left:0;padding:22px 18px 40px}
  .layout{flex-direction:column;padding-top:64px}
  .topnav{padding:0 16px}
  .topnav-email{display:none}
  .metric-grid{grid-template-columns:1fr;gap:12px}
  .page-header{flex-direction:column;align-items:stretch}
  .form-row{padding:14px 16px}
  th,td{padding:12px 16px}
}
</style>
{{end}}
`
