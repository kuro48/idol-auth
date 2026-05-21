package http

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

type settingsFlowService interface {
	GetSettingsFlow(ctx context.Context, r *http.Request, flowID string) (*KratosSettingsFlow, error)
	SubmitSettingsFlow(ctx context.Context, r *http.Request, flowID string, form url.Values) (KratosSettingsSubmitResult, error)
}

type settingsSection struct {
	Group  string
	Title  string
	Inputs []KratosSettingsNode
	Hidden []KratosSettingsNode
	Submit *KratosSettingsNode
}

type settingsData struct {
	Action      string
	Method      string
	Email       string
	Phone       string
	DisplayName string
	OshiColor   template.CSS
	Initials    string
	BackURL     string
	Sections    []settingsSection
	Messages    []KratosSettingsMessage
	Nonce       string
}

func (s *server) handleSettings(w http.ResponseWriter, r *http.Request) {
	flowID := strings.TrimSpace(r.URL.Query().Get("flow"))
	returnTo := strings.TrimRight(s.config.App.BaseURL, "/") + "/account"
	kratosSettingsURL := strings.TrimRight(s.config.Ory.KratosBrowserURL, "/") + "/self-service/settings/browser?" +
		url.Values{"return_to": {returnTo}}.Encode()

	if flowID == "" {
		http.Redirect(w, r, kratosSettingsURL, http.StatusFound)
		return
	}

	svc, ok := s.authSvc.(settingsFlowService)
	if !ok {
		http.Error(w, "settings service unavailable", http.StatusServiceUnavailable)
		return
	}

	flow, err := svc.GetSettingsFlow(r.Context(), r, flowID)
	if err != nil {
		if errors.Is(err, ErrSettingsFlowExpired) || errors.Is(err, ErrNoActiveSession) {
			http.Redirect(w, r, kratosSettingsURL, http.StatusFound)
			return
		}
		http.Error(w, "failed to load settings flow", http.StatusBadGateway)
		return
	}

	session, _ := accountSessionFromContext(r.Context())
	oshiColor := template.CSS("#1740c9")
	if c := session.OshiColor; c != "" {
		oshiColor = template.CSS(c)
	}

	nonce, err := newCSRFToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setAccountCenterHeaders(w, nonce)
	_ = settingsTpl.Execute(w, settingsData{
		Action:      "/v1/settings/flow?flow=" + url.QueryEscape(flow.ID),
		Method:      http.MethodPost,
		Email:       session.Email,
		Phone:       session.Phone,
		DisplayName: session.DisplayName,
		OshiColor:   oshiColor,
		Initials:    initials(session.DisplayName, session.Email),
		BackURL:     "/account",
		Sections:    buildSettingsSections(flow),
		Messages:    flow.Messages,
		Nonce:       nonce,
	})
}

func (s *server) handleSettingsFlowGet(w http.ResponseWriter, r *http.Request) {
	flowID := strings.TrimSpace(r.URL.Query().Get("id"))
	if flowID == "" {
		writeError(w, http.StatusBadRequest, "settings flow id is required")
		return
	}
	svc, ok := s.authSvc.(settingsFlowService)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "settings service unavailable")
		return
	}
	flow, err := svc.GetSettingsFlow(r.Context(), r, flowID)
	if err != nil {
		writeSettingsFlowError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flow)
}

func (s *server) handleSettingsFlowSubmit(w http.ResponseWriter, r *http.Request) {
	flowID := strings.TrimSpace(r.URL.Query().Get("flow"))
	if flowID == "" {
		flowID = strings.TrimSpace(r.URL.Query().Get("id"))
	}
	if flowID == "" {
		writeError(w, http.StatusBadRequest, "settings flow id is required")
		return
	}
	svc, ok := s.authSvc.(settingsFlowService)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "settings service unavailable")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	result, err := svc.SubmitSettingsFlow(r.Context(), r, flowID, r.PostForm)
	if err != nil {
		writeSettingsFlowError(w, err)
		return
	}
	for _, cookie := range result.SetCookies {
		w.Header().Add("Set-Cookie", cookie)
	}
	if result.RedirectTo != "" {
		http.Redirect(w, r, s.safeSettingsRedirect(result.RedirectTo), http.StatusSeeOther)
		return
	}
	if result.Flow != nil {
		session, _ := accountSessionFromContext(r.Context())
		oshiColor := template.CSS("#1740c9")
		if c := session.OshiColor; c != "" {
			oshiColor = template.CSS(c)
		}
		nonce, nonceErr := newCSRFToken()
		if nonceErr != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		setAccountCenterHeaders(w, nonce)
		w.WriteHeader(http.StatusBadRequest)
		_ = settingsTpl.Execute(w, settingsData{
			Action:      "/v1/settings/flow?flow=" + url.QueryEscape(result.Flow.ID),
			Method:      http.MethodPost,
			Email:       session.Email,
			Phone:       session.Phone,
			DisplayName: session.DisplayName,
			OshiColor:   oshiColor,
			Initials:    initials(session.DisplayName, session.Email),
			BackURL:     "/account",
			Sections:    buildSettingsSections(result.Flow),
			Messages:    result.Flow.Messages,
			Nonce:       nonce,
		})
		return
	}
	http.Redirect(w, r, "/account", http.StatusSeeOther)
}

func (s *server) safeSettingsRedirect(raw string) string {
	target := strings.TrimSpace(raw)
	if target == "" {
		return "/account"
	}
	if strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//") {
		return target
	}
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return "/account"
	}
	if sameOrigin(u, s.config.App.BaseURL) || sameOrigin(u, s.config.Ory.KratosBrowserURL) {
		return target
	}
	return "/account"
}

func sameOrigin(u *url.URL, allowedRaw string) bool {
	allowed, err := url.Parse(strings.TrimSpace(allowedRaw))
	if err != nil || allowed.Host == "" {
		return false
	}
	return strings.EqualFold(u.Scheme, allowed.Scheme) && strings.EqualFold(u.Host, allowed.Host)
}

func writeSettingsFlowError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSettingsFlowExpired):
		writeError(w, http.StatusNotFound, "settings flow expired or not found")
	case errors.Is(err, ErrNoActiveSession):
		writeError(w, http.StatusUnauthorized, "authentication required")
	default:
		writeError(w, http.StatusBadGateway, "failed to proxy settings flow")
	}
}

func buildSettingsSections(flow *KratosSettingsFlow) []settingsSection {
	var csrfValue string
	groupNodes := make(map[string][]KratosSettingsNode)

	for _, node := range flow.Nodes {
		if node.Group == "default" && node.Name == "csrf_token" {
			csrfValue = node.Value
			continue
		}
		groupNodes[node.Group] = append(groupNodes[node.Group], node)
	}

	csrfNode := KratosSettingsNode{
		Group:     "default",
		Name:      "csrf_token",
		InputType: "hidden",
		Value:     csrfValue,
	}

	order := []struct{ group, title string }{
		{"profile", "連絡先・プロフィール"},
		{"password", "パスワード変更"},
		{"totp", "二段階認証（TOTP）"},
		{"lookup_secret", "バックアップコード"},
	}

	sections := make([]settingsSection, 0, len(order))
	for _, o := range order {
		nodes, ok := groupNodes[o.group]
		if !ok {
			continue
		}

		var inputs, hidden []KratosSettingsNode
		var submit *KratosSettingsNode

		for i := range nodes {
			n := nodes[i]
			switch n.InputType {
			case "submit":
				n2 := n
				submit = &n2
			case "hidden":
				hidden = append(hidden, n)
			default:
				inputs = append(inputs, n)
			}
		}
		hidden = append(hidden, csrfNode)

		sections = append(sections, settingsSection{
			Group:  o.group,
			Title:  o.title,
			Inputs: inputs,
			Hidden: hidden,
			Submit: submit,
		})
	}
	return sections
}
