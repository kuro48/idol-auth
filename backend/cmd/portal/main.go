package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kuro48/idol-auth/internal/demo"
	kratosinfra "github.com/kuro48/idol-auth/internal/infra/kratos"
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
	handler, err := newPortalHandler(cfg, http.DefaultClient)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("portal server starting", "addr", server.Addr)
	return server.ListenAndServe()
}

func newPortalHandler(cfg *demo.PortalConfig, httpClient *http.Client) (http.Handler, error) {
	kratosClient := demo.NewKratosFlowClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)
	sessionClient := kratosinfra.NewFrontendClient(cfg.KratosPublicURL, cfg.KratosBrowserURL)
	themeUpdater := kratosinfra.NewAdminClient(cfg.KratosAdminURL)

	kratosPublicURL, err := url.Parse(cfg.KratosPublicURL)
	if err != nil {
		return nil, fmt.Errorf("parse kratos public url: %w", err)
	}
	kratosProxy := httputil.NewSingleHostReverseProxy(kratosPublicURL)

	var turnstile *demo.TurnstileVerifier
	if strings.TrimSpace(cfg.TurnstileSecretKey) != "" {
		turnstile = demo.NewTurnstileVerifier(cfg.TurnstileSecretKey)
		slog.Info("turnstile bot protection enabled for registration")
	}

	mux := http.NewServeMux()
	mux.Handle("/self-service/", demo.ProtectRegistration(kratosProxy, turnstile))
	mux.Handle("/.well-known/", kratosProxy)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, accountCenterURL(cfg), http.StatusSeeOther)
	})
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handleLogout(w, r, cfg, httpClient)
	})
	mux.HandleFunc("/ui/theme", func(w http.ResponseWriter, r *http.Request) {
		demo.HandleThemePreference(w, r, sessionClient, themeUpdater)
	})
	legalBase := legalBaseURL(cfg)
	accountCenter := accountCenterURL(cfg)
	flowPage := flowPageConfig{
		kratosClient:     kratosClient,
		sessionClient:    sessionClient,
		legalBaseURL:     legalBase,
		accountCenterURL: accountCenter,
		turnstileSiteKey: cfg.TurnstileSiteKey,
	}
	registerFlow(mux, flowPage, "login", "ログイン", "メールアドレスとパスワードでログインします。")
	registerFlow(mux, flowPage, "registration", "新規登録", "推し活アカウントを作成します。")
	registerFlow(mux, flowPage, "recovery", "パスワード再設定", "アカウントを回復します。")
	registerFlow(mux, flowPage, "verification", "確認", "メールアドレスを確認します。")
	registerFlow(mux, flowPage, "settings", "セキュリティ設定", "パスワードや二段階認証を管理します。")
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		demo.HandleKratosError(w, r, kratosClient)
	})
	return mux, nil
}

func accountCenterURL(cfg *demo.PortalConfig) string {
	if target := strings.TrimSpace(cfg.AccountCenterURL); target != "" {
		return target
	}
	return strings.TrimRight(cfg.AppURL, "/") + "/account/"
}

type flowPageConfig struct {
	kratosClient     *demo.KratosFlowClient
	sessionClient    demo.SessionReader
	legalBaseURL     string
	accountCenterURL string
	turnstileSiteKey string
}

func registerFlow(mux *http.ServeMux, page flowPageConfig, flowType, title, description string) {
	mux.HandleFunc("/"+flowType, func(w http.ResponseWriter, r *http.Request) {
		flowID := r.URL.Query().Get("flow")
		if flowID == "" {
			http.Redirect(w, r, page.kratosClient.BrowserInitURL(flowType), http.StatusFound)
			return
		}
		flow, err := page.kratosClient.GetFlow(r.Context(), r, flowType, flowID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err := demo.RenderPage(w, demo.PageData{
			Title:            title,
			Description:      description,
			FlowType:         flowType,
			OshiColor:        demo.ResolveSessionOshiColor(r.Context(), page.sessionClient, r),
			LegalBaseURL:     page.legalBaseURL,
			AccountCenterURL: page.accountCenterURL,
			TurnstileSiteKey: page.turnstileSiteKey,
			Flow:             flow,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func legalBaseURL(cfg *demo.PortalConfig) string {
	accountCenter, err := url.Parse(accountCenterURL(cfg))
	if err != nil || accountCenter.Scheme == "" || accountCenter.Host == "" {
		return ""
	}
	return accountCenter.Scheme + "://" + accountCenter.Host
}

func handleLogout(w http.ResponseWriter, r *http.Request, cfg *demo.PortalConfig, client *http.Client) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	loginURL := legalBaseURL(cfg) + "/login"
	logoutBrowserURL, err := url.Parse(strings.TrimRight(cfg.KratosPublicURL, "/") + "/self-service/logout/browser")
	if err != nil {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	q := logoutBrowserURL.Query()
	q.Set("return_to", loginURL)
	logoutBrowserURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, logoutBrowserURL.String(), nil)
	if err != nil {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	if cookie := demo.FilterOryCookies(r.Header.Get("Cookie")); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var decoded struct {
		LogoutURL string `json:"logout_url"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil || strings.TrimSpace(decoded.LogoutURL) == "" {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, decoded.LogoutURL, http.StatusSeeOther)
}
