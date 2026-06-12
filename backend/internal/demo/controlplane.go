package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ControlPlaneClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type AppRecord struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type ClientRecord struct {
	ID                     string   `json:"id"`
	HydraClientID          string   `json:"hydra_client_id"`
	RedirectURIs           []string `json:"redirect_uris"`
	PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris"`
}

type DemoAppSpec struct {
	Key         string
	Name        string
	Slug        string
	Description string
	PartyType   string
	ClientName  string
}

func NewControlPlaneClient(baseURL, token string) *ControlPlaneClient {
	return &ControlPlaneClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *ControlPlaneClient) EnsureDemoClient(ctx context.Context, cfg *Config, spec DemoAppSpec) (string, error) {
	apps, err := c.listApps(ctx)
	if err != nil {
		return "", err
	}

	var app AppRecord
	found := false
	for _, item := range apps {
		if item.Slug == spec.Slug {
			app = item
			found = true
			break
		}
	}
	if !found {
		app, err = c.createApp(ctx, spec)
		if err != nil {
			return "", err
		}
	}

	clients, err := c.listClients(ctx, app.ID)
	if err != nil {
		return "", err
	}
	callbackURL := strings.TrimRight(cfg.AppURL, "/") + "/oauth/callback"
	for _, client := range clients {
		for _, uri := range client.RedirectURIs {
			if uri == callbackURL {
				return client.HydraClientID, nil
			}
		}
	}

	client, err := c.createClient(ctx, app.ID, spec, callbackURL, strings.TrimRight(cfg.AppURL, "/")+"/")
	if err != nil {
		return "", err
	}
	return client.HydraClientID, nil
}

func (c *ControlPlaneClient) listApps(ctx context.Context) ([]AppRecord, error) {
	var resp struct {
		Items []AppRecord `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/admin/apps", nil, &resp); err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}
	return resp.Items, nil
}

func (c *ControlPlaneClient) createApp(ctx context.Context, spec DemoAppSpec) (AppRecord, error) {
	var resp AppRecord
	payload := map[string]any{
		"name":        spec.Name,
		"slug":        spec.Slug,
		"type":        "spa",
		"party_type":  spec.PartyType,
		"description": spec.Description,
	}
	if err := c.doJSON(ctx, http.MethodPost, "/v1/admin/apps", payload, &resp); err != nil {
		return AppRecord{}, fmt.Errorf("create app: %w", err)
	}
	return resp, nil
}

func (c *ControlPlaneClient) listClients(ctx context.Context, appID string) ([]ClientRecord, error) {
	var resp struct {
		Items []ClientRecord `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/admin/apps/"+appID+"/clients", nil, &resp); err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	return resp.Items, nil
}

func (c *ControlPlaneClient) createClient(ctx context.Context, appID string, spec DemoAppSpec, callbackURL, postLogoutURL string) (ClientRecord, error) {
	var resp struct {
		Client ClientRecord `json:"client"`
	}
	payload := map[string]any{
		"name":                      spec.ClientName,
		"client_type":               "public",
		"redirect_uris":             []string{callbackURL},
		"post_logout_redirect_uris": []string{postLogoutURL},
		"scopes":                    []string{"openid", "offline_access"},
	}
	if err := c.doJSON(ctx, http.MethodPost, "/v1/admin/apps/"+appID+"/clients", payload, &resp); err != nil {
		return ClientRecord{}, fmt.Errorf("create client: %w", err)
	}
	return resp.Client, nil
}

func PrimaryAppSpec(cfg *Config) DemoAppSpec {
	return DemoAppSpec{
		Key:         "first_party",
		Name:        cfg.AppName,
		Slug:        cfg.AppSlug,
		Description: cfg.AppDescription,
		PartyType:   "first_party",
		ClientName:  "Idol Demo Browser Client",
	}
}

func PartnerAppSpec(cfg *Config) DemoAppSpec {
	return DemoAppSpec{
		Key:         "partner",
		Name:        cfg.PartnerAppName,
		Slug:        cfg.PartnerAppSlug,
		Description: cfg.PartnerAppDescription,
		PartyType:   "third_party",
		ClientName:  "Idol Partner Demo Client",
	}
}

func (c *ControlPlaneClient) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
