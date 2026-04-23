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
			http.Error(w, "bootstrap demo client: "+err.Error(), http.StatusBadGateway)
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
			http.Error(w, "oauth error: "+errValue, http.StatusBadRequest)
			return
		}
		stateCookie, err := r.Cookie(stateCookieName)
		if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "invalid oauth state", http.StatusBadRequest)
			return
		}
		verifierCookie, err := r.Cookie(verifierCookieName)
		if err != nil || verifierCookie.Value == "" {
			http.Error(w, "missing pkce verifier", http.StatusBadRequest)
			return
		}
		clientKind := ""
		if clientKindCookieValue, err := r.Cookie(clientKindCookie); err == nil {
			clientKind = clientKindCookieValue.Value
		}

		clientID, err := controlPlane.EnsureDemoClient(r.Context(), cfg, selectedAppSpec(cfg, clientKind))
		if err != nil {
			http.Error(w, "bootstrap demo client: "+err.Error(), http.StatusBadGateway)
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

	registerFlow(mux, kratosClient, "login", "Login", "Sign in with the shared account.")
	registerFlow(mux, kratosClient, "registration", "Registration", "Create a shared account.")
	registerFlow(mux, kratosClient, "recovery", "Recovery", "Recover your account.")
	registerFlow(mux, kratosClient, "verification", "Verification", "Verify your identifier.")
	registerFlow(mux, kratosClient, "settings", "Settings", "Manage security settings and MFA.")

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
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Idol Demo Client</title>
  <style>
    body { margin: 0; font-family: Georgia, serif; background: radial-gradient(circle at top left, #f4d9c6, #dbe8ef 55%, #eef3f6); color: #14232e; }
    main { max-width: 900px; margin: 48px auto; padding: 32px; background: rgba(255,255,255,0.88); border-radius: 24px; box-shadow: 0 24px 60px rgba(20,35,46,0.14); }
    a.button { display: inline-block; background: #155e75; color: white; padding: 14px 18px; border-radius: 12px; text-decoration: none; margin-right: 12px; margin-bottom: 12px; }
  </style>
</head>
<body>
  <main>
    <h1>Idol Shared Account Demo</h1>
    <p>This app acts as both a local OIDC demo client and the minimal Kratos UI needed to exercise browser flows.</p>
    <a class="button" href="/oauth/start">Start First-Party Login</a>
    <a class="button" href="/oauth/start?app=partner">Start Partner Login</a>
    <a class="button" href="/registration">Create Account</a>
    <a class="button" href="/login">Open Login UI</a>
    <a class="button" href="/settings">Security Settings</a>
  </main>
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
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>OIDC Tokens</title></head>
<body style="font-family: monospace; background: #0f172a; color: #e2e8f0; margin: 0; padding: 24px;">
<h1>OIDC Callback Complete</h1>
<p><a href="/" style="color:#7dd3fc">Back to demo home</a></p>
<pre style="white-space: pre-wrap; word-break: break-word;">{{ . }}</pre>
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
