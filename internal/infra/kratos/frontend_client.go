package kratos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	apphttp "github.com/kuro48/idol-auth/internal/http"
	"github.com/kuro48/idol-auth/internal/oshi"
)

type FrontendClient struct {
	apiBaseURL     string
	browserBaseURL string
	httpClient     *http.Client
}

func NewFrontendClient(apiBaseURL, browserBaseURL string) *FrontendClient {
	return &FrontendClient{
		apiBaseURL:     strings.TrimRight(apiBaseURL, "/"),
		browserBaseURL: strings.TrimRight(browserBaseURL, "/"),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *FrontendClient) ToSession(ctx context.Context, r *http.Request) (apphttp.KratosSession, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+"/sessions/whoami", nil)
	if err != nil {
		return apphttp.KratosSession{}, fmt.Errorf("build kratos whoami request: %w", err)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		if filtered := filterOryCookies(cookie); filtered != "" {
			req.Header.Set("Cookie", filtered)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return apphttp.KratosSession{}, fmt.Errorf("call kratos whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return apphttp.KratosSession{}, apphttp.ErrNoActiveSession
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "kratos upstream error", "op", "whoami", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return apphttp.KratosSession{}, fmt.Errorf("kratos whoami returned status %d", resp.StatusCode)
	}

	var decoded struct {
		Active                      bool   `json:"active"`
		AuthenticatorAssuranceLevel string `json:"authenticator_assurance_level"`
		AuthenticationMethods       []struct {
			Method string `json:"method"`
		} `json:"authentication_methods"`
		Identity struct {
			ID     string `json:"id"`
			Traits struct {
				Email       string `json:"email"`
				Phone       string `json:"phone"`
				DisplayName string `json:"display_name"`
			} `json:"traits"`
			MetadataPublic struct {
				Roles     []string `json:"roles"`
				OshiColor string   `json:"oshi_color"`
				OshiIDs   []string `json:"oshi_ids"`
				FanSince  string   `json:"fan_since"`
			} `json:"metadata_public"`
		} `json:"identity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return apphttp.KratosSession{}, fmt.Errorf("decode kratos whoami response: %w", err)
	}
	methods := make([]string, 0, len(decoded.AuthenticationMethods))
	for _, method := range decoded.AuthenticationMethods {
		if method.Method != "" {
			methods = append(methods, method.Method)
		}
	}
	return apphttp.KratosSession{
		Active:                      decoded.Active,
		IdentityID:                  decoded.Identity.ID,
		Email:                       decoded.Identity.Traits.Email,
		Phone:                       decoded.Identity.Traits.Phone,
		DisplayName:                 decoded.Identity.Traits.DisplayName,
		Roles:                       decoded.Identity.MetadataPublic.Roles,
		OshiColor:                   oshi.NormalizeColor(decoded.Identity.MetadataPublic.OshiColor),
		OshiIDs:                     decoded.Identity.MetadataPublic.OshiIDs,
		FanSince:                    decoded.Identity.MetadataPublic.FanSince,
		Methods:                     methods,
		AuthenticatorAssuranceLevel: decoded.AuthenticatorAssuranceLevel,
	}, nil
}

// filterOryCookies returns only the semicolon-separated cookie pairs whose
// name starts with "ory_", preventing unrelated session cookies from being
// forwarded to the Kratos API.
func filterOryCookies(cookieHeader string) string {
	parts := strings.Split(cookieHeader, ";")
	ory := parts[:0]
	for _, part := range parts {
		if strings.HasPrefix(strings.TrimSpace(part), "ory_") {
			ory = append(ory, strings.TrimSpace(part))
		}
	}
	return strings.Join(ory, "; ")
}

func (c *FrontendClient) BrowserLoginURL(returnTo string) string {
	values := url.Values{}
	if returnTo != "" {
		values.Set("return_to", returnTo)
	}
	return c.browserBaseURL + "/self-service/login/browser?" + values.Encode()
}

func (c *FrontendClient) BrowserSettingsURL(returnTo string) string {
	values := url.Values{}
	if returnTo != "" {
		values.Set("return_to", returnTo)
	}
	return c.browserBaseURL + "/self-service/settings/browser?" + values.Encode()
}

func (c *FrontendClient) GetSettingsFlow(ctx context.Context, r *http.Request, flowID string) (*apphttp.KratosSettingsFlow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+"/self-service/settings/flows?id="+url.QueryEscape(flowID), nil)
	if err != nil {
		return nil, fmt.Errorf("build kratos settings flow request: %w", err)
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		if filtered := filterOryCookies(cookie); filtered != "" {
			req.Header.Set("Cookie", filtered)
		}
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call kratos settings flow: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound, http.StatusGone:
		return nil, apphttp.ErrSettingsFlowExpired
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, apphttp.ErrNoActiveSession
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "kratos upstream error", "op", "settings_flow", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return nil, fmt.Errorf("kratos settings flow returned status %d", resp.StatusCode)
	}

	var decoded struct {
		ID string `json:"id"`
		UI struct {
			Action   string `json:"action"`
			Method   string `json:"method"`
			Nodes    []struct {
				Type  string `json:"type"`
				Group string `json:"group"`
				Attributes struct {
					Name     string `json:"name"`
					Type     string `json:"type"`
					Value    any    `json:"value"`
					Required bool   `json:"required"`
					Disabled bool   `json:"disabled"`
				} `json:"attributes"`
				Messages []struct {
					ID   int    `json:"id"`
					Text string `json:"text"`
					Type string `json:"type"`
				} `json:"messages"`
				Meta struct {
					Label *struct {
						Text string `json:"text"`
					} `json:"label"`
				} `json:"meta"`
			} `json:"nodes"`
			Messages []struct {
				ID   int    `json:"id"`
				Text string `json:"text"`
				Type string `json:"type"`
			} `json:"messages"`
		} `json:"ui"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode kratos settings flow response: %w", err)
	}

	nodes := make([]apphttp.KratosSettingsNode, 0, len(decoded.UI.Nodes))
	for _, n := range decoded.UI.Nodes {
		if n.Type != "input" {
			continue
		}
		valueStr := ""
		if n.Attributes.Value != nil {
			valueStr = fmt.Sprintf("%v", n.Attributes.Value)
		}
		label := ""
		if n.Meta.Label != nil {
			label = n.Meta.Label.Text
		}
		msgs := make([]apphttp.KratosSettingsMessage, 0, len(n.Messages))
		for _, m := range n.Messages {
			msgs = append(msgs, apphttp.KratosSettingsMessage{ID: m.ID, Text: m.Text, Type: m.Type})
		}
		nodes = append(nodes, apphttp.KratosSettingsNode{
			Type:      n.Type,
			Group:     n.Group,
			Name:      n.Attributes.Name,
			InputType: n.Attributes.Type,
			Value:     valueStr,
			Label:     label,
			Required:  n.Attributes.Required,
			Disabled:  n.Attributes.Disabled,
			Messages:  msgs,
		})
	}

	flowMsgs := make([]apphttp.KratosSettingsMessage, 0, len(decoded.UI.Messages))
	for _, m := range decoded.UI.Messages {
		flowMsgs = append(flowMsgs, apphttp.KratosSettingsMessage{ID: m.ID, Text: m.Text, Type: m.Type})
	}

	return &apphttp.KratosSettingsFlow{
		ID:       decoded.ID,
		Action:   decoded.UI.Action,
		Method:   decoded.UI.Method,
		Nodes:    nodes,
		Messages: flowMsgs,
	}, nil
}
