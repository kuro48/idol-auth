package hydra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
)

type AdminClient struct {
	baseURL    string
	httpClient *http.Client
}

type createClientRequest struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	RedirectURIs            []string `json:"redirect_uris,omitempty"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	SkipConsent             bool     `json:"skip_consent"`
}

type createClientResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func NewAdminClient(baseURL string) *AdminClient {
	return &AdminClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *AdminClient) CreateClient(ctx context.Context, spec app.ClientProvisionSpec) (app.ProvisionedClient, error) {
	payload, err := json.Marshal(createClientRequest{
		ClientID:                spec.HydraClientID,
		ClientName:              spec.Name,
		GrantTypes:              spec.GrantTypes,
		ResponseTypes:           spec.ResponseTypes,
		Scope:                   strings.Join(spec.Scopes, " "),
		RedirectURIs:            spec.RedirectURIs,
		PostLogoutRedirectURIs:  spec.PostLogoutRedirectURIs,
		TokenEndpointAuthMethod: spec.TokenEndpointAuthMethod,
		SkipConsent:             spec.SkipConsent,
	})
	if err != nil {
		return app.ProvisionedClient{}, fmt.Errorf("marshal hydra client request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/admin/clients", bytes.NewReader(payload))
	if err != nil {
		return app.ProvisionedClient{}, fmt.Errorf("build hydra create client request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return app.ProvisionedClient{}, fmt.Errorf("call hydra create client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "hydra upstream error", "op", "create client", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return app.ProvisionedClient{}, fmt.Errorf("hydra create client returned status %d", resp.StatusCode)
	}

	var decoded createClientResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return app.ProvisionedClient{}, fmt.Errorf("decode hydra create client response: %w", err)
	}
	return app.ProvisionedClient{
		HydraClientID:   decoded.ClientID,
		ClientSecret:    decoded.ClientSecret,
		ClientSecretSet: decoded.ClientSecret != "",
	}, nil
}

func (c *AdminClient) DeleteClient(ctx context.Context, clientID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/admin/clients/"+url.PathEscape(clientID), nil)
	if err != nil {
		return fmt.Errorf("build hydra delete client request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call hydra delete client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		slog.WarnContext(ctx, "hydra upstream error", "op", "delete client", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return fmt.Errorf("hydra delete client returned status %d", resp.StatusCode)
	}
	return nil
}
