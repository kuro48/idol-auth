package http

import "html/template"

var settingsTpl = template.Must(template.New("settings").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>設定 — アカウントセンター</title>
  <style>
    :root {
      --oshi: {{ .OshiColor }};
      --bg: #f5f6fa;
      --card: #fff;
      --text: #1d2040;
      --muted: #6f7394;
      --border: rgba(29,32,64,0.09);
      --radius: 18px;
      --shadow: 0 4px 24px rgba(29,32,64,0.08);
    }
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: "Hiragino Sans", "Yu Gothic", "Noto Sans JP", "Avenir Next", sans-serif;
      min-height: 100vh;
    }

    .page-header {
      background: linear-gradient(135deg, var(--oshi) 0%, color-mix(in srgb, var(--oshi) 60%, #fff) 100%);
      padding: 28px 24px 48px;
      position: relative;
      overflow: hidden;
    }
    .page-header::after {
      content: "";
      position: absolute;
      inset: 0;
      background: radial-gradient(circle at 80% 50%, rgba(255,255,255,0.22) 0%, transparent 60%);
      pointer-events: none;
    }
    .back-link {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      color: rgba(255,255,255,0.88);
      text-decoration: none;
      font-size: 13px;
      font-weight: 600;
      margin-bottom: 20px;
      position: relative;
      z-index: 1;
      transition: opacity 0.15s;
    }
    .back-link:hover { opacity: 0.75; }
    .header-identity {
      display: flex;
      align-items: center;
      gap: 14px;
      position: relative;
      z-index: 1;
    }
    .avatar {
      width: 52px;
      height: 52px;
      border-radius: 50%;
      background: rgba(255,255,255,0.28);
      border: 2px solid rgba(255,255,255,0.5);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 18px;
      font-weight: 800;
      color: #fff;
      flex-shrink: 0;
    }
    .header-name {
      font-size: 22px;
      font-weight: 800;
      color: #fff;
      letter-spacing: -0.03em;
    }
    .header-sub {
      font-size: 13px;
      color: rgba(255,255,255,0.75);
      margin-top: 2px;
    }

    .content {
      max-width: 680px;
      margin: -24px auto 48px;
      padding: 0 16px;
    }

    .flash {
      margin-bottom: 16px;
      padding: 14px 18px;
      border-radius: var(--radius);
      font-size: 14px;
      font-weight: 600;
    }
    .flash-error { background: #fff0f0; color: #c0392b; border: 1px solid #fcc; }
    .flash-info  { background: #f0f8ff; color: #1a5a8a; border: 1px solid #b3d9f7; }

    .section {
      background: var(--card);
      border-radius: var(--radius);
      box-shadow: var(--shadow);
      border: 1px solid var(--border);
      padding: 24px;
      margin-bottom: 16px;
    }
    .section-title {
      font-size: 13px;
      font-weight: 700;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 18px;
    }

    .field { margin-bottom: 16px; }
    .field label {
      display: block;
      font-size: 12px;
      font-weight: 600;
      color: var(--muted);
      margin-bottom: 6px;
      letter-spacing: 0.04em;
    }
    .field input {
      width: 100%;
      padding: 11px 14px;
      border: 1px solid var(--border);
      border-radius: 12px;
      font-size: 15px;
      color: var(--text);
      background: #fafbfc;
      outline: none;
      transition: border-color 0.15s, box-shadow 0.15s;
      font-family: inherit;
    }
    .field input:focus {
      border-color: var(--oshi);
      box-shadow: 0 0 0 3px color-mix(in srgb, var(--oshi) 20%, transparent);
      background: #fff;
    }
    .field input:disabled { opacity: 0.55; cursor: not-allowed; }
    .field-error { font-size: 12px; color: #c0392b; margin-top: 5px; }

    .btn-save {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 11px 22px;
      border-radius: 12px;
      font-size: 14px;
      font-weight: 700;
      cursor: pointer;
      border: none;
      background: var(--oshi);
      color: #fff;
      transition: opacity 0.15s, transform 0.1s;
      margin-top: 4px;
    }
    .btn-save:hover { opacity: 0.85; }
    .btn-save:active { transform: scale(0.98); }

    @media (max-width: 480px) {
      .page-header { padding: 20px 16px 40px; }
      .content { padding: 0 10px; }
      .section { padding: 18px; }
    }
  </style>
</head>
<body>

<div class="page-header">
  <a href="{{ .BackURL }}" class="back-link">← アカウントへ戻る</a>
  <div class="header-identity">
    <div class="avatar">{{ .Initials }}</div>
    <div>
      <div class="header-name">{{ if .DisplayName }}{{ .DisplayName }}{{ else }}アカウント設定{{ end }}</div>
      <div class="header-sub">{{ if .Email }}{{ .Email }}{{ else if .Phone }}{{ .Phone }}{{ end }}</div>
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
</body>
</html>`))
