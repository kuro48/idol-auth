package main

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ryunosukekurokawa/idol-auth/internal/demo"
)

func main() {
	if err := run(); err != nil {
		slog.Error("portal server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := demo.LoadPortalConfig()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	kratosClient := demo.NewKratosFlowClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderHome(w)
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
	slog.Info("portal server starting", "addr", server.Addr)
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

func renderHome(w http.ResponseWriter) {
	const tpl = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Idol Auth Portal</title>
  <style>
    body { margin: 0; font-family: Georgia, serif; background: linear-gradient(140deg, #f7f1e8, #d7e5f0); color: #1f2933; }
    main { max-width: 860px; margin: 48px auto; padding: 32px; background: rgba(255,255,255,0.9); border-radius: 24px; box-shadow: 0 24px 60px rgba(31,41,51,0.12); }
    a.button { display: inline-block; background: #155e75; color: white; padding: 14px 18px; border-radius: 12px; text-decoration: none; margin-right: 12px; margin-bottom: 12px; }
  </style>
</head>
<body>
  <main>
    <h1>Idol Auth Portal</h1>
    <p>User-facing authentication portal for registration, login, MFA enrollment, recovery, and verification.</p>
    <a class="button" href="/registration">Create Account</a>
    <a class="button" href="/login">Open Login</a>
    <a class="button" href="/settings">Security Settings</a>
    <a class="button" href="/recovery">Recovery</a>
    <a class="button" href="/verification">Verification</a>
  </main>
</body>
</html>`
	t := template.Must(template.New("home").Parse(tpl))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.Execute(w, nil)
}
