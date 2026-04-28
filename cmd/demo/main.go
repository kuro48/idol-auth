package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ryunosukekurokawa/idol-auth/internal/demo"
)

const (
	stateCookieName    = "idol_demo_state"
	verifierCookieName = "idol_demo_verifier"
	clientKindCookie   = "idol_demo_client_kind"
)

func main() {
	if err := run(); err != nil {
		slog.Error("demo server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := demo.LoadConfig()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	controlPlane := demo.NewControlPlaneClient(cfg.AuthInternalURL, cfg.AdminToken)
	kratosClient := demo.NewKratosFlowClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderHome(w, cfg)
	})
	mux.HandleFunc("/oauth/start", func(w http.ResponseWriter, r *http.Request) {
		spec := selectedAppSpec(cfg, r.URL.Query().Get("app"))
		clientID, err := controlPlane.EnsureDemoClient(r.Context(), cfg, spec)
		if err != nil {
			http.Error(w, "デモクライアントの初期化に失敗しました: "+err.Error(), http.StatusBadGateway)
			return
		}
		verifier, err := demo.GeneratePKCEVerifier()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		state, err := demo.GenerateState()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		secure := r.TLS != nil
		http.SetCookie(w, &http.Cookie{Name: verifierCookieName, Value: verifier, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
		http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: state, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
		http.SetCookie(w, &http.Cookie{Name: clientKindCookie, Value: spec.Key, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})

		redirectURI := strings.TrimRight(cfg.AppURL, "/") + "/oauth/callback"
		authURL, err := demo.BuildAuthorizationURL(demo.AuthorizationParams{
			HydraBrowserURL: cfg.HydraBrowserURL,
			ClientID:        clientID,
			RedirectURI:     redirectURI,
			State:           state,
			CodeChallenge:   demo.ComputeS256Challenge(verifier),
			Scopes:          []string{"openid", "offline_access"},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
	})
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		if errValue := r.URL.Query().Get("error"); errValue != "" {
			http.Error(w, "OAuth エラー: "+errValue, http.StatusBadRequest)
			return
		}
		stateCookie, err := r.Cookie(stateCookieName)
		if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "OAuth の state が不正です", http.StatusBadRequest)
			return
		}
		verifierCookie, err := r.Cookie(verifierCookieName)
		if err != nil || verifierCookie.Value == "" {
			http.Error(w, "PKCE verifier がありません", http.StatusBadRequest)
			return
		}
		clientKind := ""
		if clientKindCookieValue, err := r.Cookie(clientKindCookie); err == nil {
			clientKind = clientKindCookieValue.Value
		}

		clientID, err := controlPlane.EnsureDemoClient(r.Context(), cfg, selectedAppSpec(cfg, clientKind))
		if err != nil {
			http.Error(w, "デモクライアントの初期化に失敗しました: "+err.Error(), http.StatusBadGateway)
			return
		}

		tokenResp, err := exchangeCode(r.Context(), cfg, clientID, verifierCookie.Value, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: "", Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{Name: verifierCookieName, Value: "", Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{Name: clientKindCookie, Value: "", Path: "/", MaxAge: -1})
		renderToken(w, tokenResp)
	})

	registerFlow(mux, kratosClient, "login", "ログイン", "共有アカウントでサインインします。")
	registerFlow(mux, kratosClient, "registration", "新規登録", "共有アカウントを作成します。")
	registerFlow(mux, kratosClient, "recovery", "アカウント復旧", "アカウント復旧フローを開始します。")
	registerFlow(mux, kratosClient, "verification", "確認", "メールアドレスや識別子を確認します。")
	registerFlow(mux, kratosClient, "settings", "セキュリティ設定", "セキュリティ設定や MFA を管理します。")

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("demo server starting", "addr", server.Addr)
	return server.ListenAndServe()
}

func registerFlow(mux *http.ServeMux, kratosClient *demo.KratosFlowClient, flowType, title, description string) {
	mux.HandleFunc("/"+flowType, func(w http.ResponseWriter, r *http.Request) {
		flowID := r.URL.Query().Get("flow")
		if flowID == "" {
			http.Redirect(w, r, kratosClient.BrowserInitURL(flowType), http.StatusFound)
			return
		}
		flow, err := kratosClient.GetFlow(r.Context(), r, flowType, flowID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err := demo.RenderPage(w, demo.PageData{
			Title:       title,
			Description: description,
			FlowType:    flowType,
			Flow:        flow,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func renderHome(w http.ResponseWriter, cfg *demo.Config) {
	const tpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Idol Auth — デモ</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: #0a0c12;
      background-image: radial-gradient(ellipse at 20% 50%, rgba(108,99,255,0.08) 0%, transparent 60%),
                        radial-gradient(ellipse at 80% 20%, rgba(99,179,237,0.05) 0%, transparent 50%);
      color: #e8eaf0;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
      padding: 56px 24px;
    }
    .container { max-width: 560px; margin: 0 auto; }
    .tag {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: rgba(108,99,255,0.12);
      border: 1px solid rgba(108,99,255,0.25);
      border-radius: 100px;
      padding: 4px 12px;
      font-size: 11px;
      font-weight: 700;
      color: #a5b4fc;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      margin-bottom: 20px;
    }
    h1 { margin: 0 0 12px; font-size: 32px; font-weight: 700; letter-spacing: -0.03em; line-height: 1.2; }
    .subtitle { color: #7c8394; font-size: 15px; line-height: 1.7; margin: 0 0 40px; }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
    .card {
      background: #13161f;
      border: 1px solid rgba(255,255,255,0.07);
      border-radius: 12px;
      padding: 20px;
      text-decoration: none;
      color: #e8eaf0;
      display: flex;
      flex-direction: column;
      gap: 6px;
      transition: border-color 0.15s, background 0.15s;
    }
    .card:hover { border-color: rgba(108,99,255,0.4); background: rgba(108,99,255,0.07); }
    .card-icon { font-size: 18px; margin-bottom: 4px; }
    .card-title { font-size: 14px; font-weight: 600; }
    .card-desc { font-size: 13px; color: #7c8394; line-height: 1.5; }
    .card-primary { border-color: rgba(108,99,255,0.3); background: rgba(108,99,255,0.09); }
  </style>
</head>
<body>
  <div class="container">
    <div class="tag">✦ Demo</div>
    <h1>Idol Auth</h1>
    <p class="subtitle">ローカル OIDC デモクライアントと Kratos ブラウザフロー確認用の最小 UI。</p>
    <div class="grid">
      <a class="card card-primary" href="/oauth/start">
        <div class="card-icon">→</div>
        <div class="card-title">ファーストパーティでログイン</div>
        <div class="card-desc">メインアプリで OAuth2 フローを開始</div>
      </a>
      <a class="card" href="/oauth/start?app=partner">
        <div class="card-icon">⇄</div>
        <div class="card-title">パートナーアプリでログイン</div>
        <div class="card-desc">パートナー用クライアントでフローを開始</div>
      </a>
      <a class="card" href="/registration">
        <div class="card-icon">+</div>
        <div class="card-title">アカウントを作成</div>
        <div class="card-desc">新規ユーザー登録フロー</div>
      </a>
      <a class="card" href="/login">
        <div class="card-icon">◉</div>
        <div class="card-title">ログイン画面を開く</div>
        <div class="card-desc">Kratos ブラウザログインフロー</div>
      </a>
      <a class="card" href="/settings">
        <div class="card-icon">◈</div>
        <div class="card-title">セキュリティ設定</div>
        <div class="card-desc">MFA・パスワード設定</div>
      </a>
    </div>
  </div>
</body>
</html>`
	t := template.Must(template.New("home").Parse(tpl))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.Execute(w, nil)
}

func selectedAppSpec(cfg *demo.Config, key string) demo.DemoAppSpec {
	if key == "partner" {
		return demo.PartnerAppSpec(cfg)
	}
	return demo.PrimaryAppSpec(cfg)
}

func exchangeCode(ctx context.Context, cfg *demo.Config, clientID, verifier, code string) (map[string]any, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("client_id", clientID)
	values.Set("redirect_uri", strings.TrimRight(cfg.AppURL, "/")+"/oauth/callback")
	values.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.HydraPublicURL, "/")+"/oauth2/token", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return decoded, nil
}

func renderToken(w http.ResponseWriter, tokenResp map[string]any) {
	const tpl = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>OIDC トークン — Idol Auth</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: #0a0c12;
      color: #e8eaf0;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
      padding: 48px 24px;
    }
    .container { max-width: 600px; margin: 0 auto; }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: rgba(34,197,94,0.12);
      border: 1px solid rgba(34,197,94,0.25);
      border-radius: 100px;
      padding: 4px 12px;
      font-size: 11px;
      font-weight: 700;
      color: #86efac;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      margin-bottom: 20px;
    }
    h1 { margin: 0 0 8px; font-size: 24px; font-weight: 700; letter-spacing: -0.02em; }
    .back {
      display: inline-block;
      color: #7c8394;
      font-size: 14px;
      text-decoration: none;
      margin-bottom: 32px;
      transition: color 0.15s;
    }
    .back:hover { color: #e8eaf0; }
    pre {
      background: #13161f;
      border: 1px solid rgba(255,255,255,0.08);
      border-radius: 12px;
      padding: 24px;
      font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
      font-size: 13px;
      line-height: 1.7;
      color: #a5f3fc;
      overflow-x: auto;
      white-space: pre-wrap;
      word-break: break-word;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="badge">✓ 認証完了</div>
    <h1>OIDC コールバック完了</h1>
    <a class="back" href="/">← デモのホームに戻る</a>
    <pre>{{ . }}</pre>
  </div>
</body>
</html>`
	b, _ := json.MarshalIndent(sanitizeTokenResponse(tokenResp), "", "  ")
	t := template.Must(template.New("token").Parse(tpl))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = t.Execute(w, string(b))
}

func sanitizeTokenResponse(tokenResp map[string]any) map[string]any {
	if tokenResp == nil {
		return nil
	}
	sanitized := make(map[string]any, len(tokenResp))
	for key, value := range tokenResp {
		switch key {
		case "access_token", "refresh_token", "id_token":
			sanitized[key] = "<redacted>"
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}
