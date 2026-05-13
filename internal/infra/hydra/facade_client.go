package hydra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseBodyBytes = 1 << 20 // 1 MB

// FacadeClient proxies OAuth2 token operations to Hydra on behalf of external apps.
// Token and Revoke use the Hydra public port; Introspect uses the admin port.
type FacadeClient struct {
	publicURL  string
	adminURL   string
	httpClient *http.Client
}

func NewFacadeClient(publicURL, adminURL string) *FacadeClient {
	return &FacadeClient{
		publicURL:  strings.TrimRight(publicURL, "/"),
		adminURL:   strings.TrimRight(adminURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Token forwards a form-encoded body to Hydra's /oauth2/token endpoint.
// Returns the raw response body and HTTP status code.
func (c *FacadeClient) Token(ctx context.Context, body []byte) ([]byte, int, error) {
	return c.post(ctx, c.publicURL+"/oauth2/token", body)
}

// Revoke forwards a form-encoded body to Hydra's /oauth2/revoke endpoint.
func (c *FacadeClient) Revoke(ctx context.Context, body []byte) ([]byte, int, error) {
	return c.post(ctx, c.publicURL+"/oauth2/revoke", body)
}

// Introspect forwards a form-encoded body to Hydra's admin /admin/oauth2/introspect endpoint.
func (c *FacadeClient) Introspect(ctx context.Context, body []byte) ([]byte, int, error) {
	return c.post(ctx, c.adminURL+"/admin/oauth2/introspect", body)
}

// OAuthClientExists reports whether the given client ID is registered in Hydra.
// Returns false (not error) on 404; returns error on network failure or unexpected status.
func (c *FacadeClient) OAuthClientExists(ctx context.Context, clientID string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.adminURL+"/admin/clients/"+url.PathEscape(clientID), nil)
	if err != nil {
		return false, fmt.Errorf("build hydra client check request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("call hydra admin clients: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("hydra admin clients returned %d", resp.StatusCode)
	}
	return true, nil
}

func (c *FacadeClient) post(ctx context.Context, endpoint string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build hydra facade request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call hydra facade: %w", err)
	}
	defer resp.Body.Close()

	out, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read hydra facade response: %w", err)
	}
	return out, resp.StatusCode, nil
}
