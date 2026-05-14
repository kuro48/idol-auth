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

	setAccountCenterHeaders(w)
	_ = settingsTpl.Execute(w, settingsData{
		Action:      flow.Action,
		Method:      flow.Method,
		Email:       session.Email,
		Phone:       session.Phone,
		DisplayName: session.DisplayName,
		OshiColor:   oshiColor,
		Initials:    initials(session.DisplayName, session.Email),
		BackURL:     "/account",
		Sections:    buildSettingsSections(flow),
		Messages:    flow.Messages,
	})
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
