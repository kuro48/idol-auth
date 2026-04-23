package hydra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
)

type FlowClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewFlowClient(baseURL string) *FlowClient {
	return &FlowClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *FlowClient) GetLoginRequest(ctx context.Context, loginChallenge string) (apphttp.HydraLoginRequest, error) {
	var resp struct {
		Skip    bool   `json:"skip"`
		Subject string `json:"subject"`
	}
	if err := c.get(ctx, "/admin/oauth2/auth/requests/login", "login_challenge", loginChallenge, &resp); err != nil {
		return apphttp.HydraLoginRequest{}, err
	}
	return apphttp.HydraLoginRequest{Skip: resp.Skip, Subject: resp.Subject}, nil
}

func (c *FlowClient) AcceptLoginRequest(ctx context.Context, loginChallenge, subject string) (string, error) {
	var resp struct {
		RedirectTo string `json:"redirect_to"`
	}
	if err := c.put(ctx, "/admin/oauth2/auth/requests/login/accept", "login_challenge", loginChallenge, map[string]any{
		"subject": subject,
	}, &resp); err != nil {
		return "", err
	}
	return resp.RedirectTo, nil
}

func (c *FlowClient) GetConsentRequest(ctx context.Context, consentChallenge string) (apphttp.HydraConsentRequest, error) {
	var resp struct {
		Subject                      string   `json:"subject"`
		Skip                         bool     `json:"skip"`
		RequestedScope               []string `json:"requested_scope"`
		RequestedAccessTokenAudience []string `json:"requested_access_token_audience"`
		Client                       struct {
			ClientID    string `json:"client_id"`
			ClientName  string `json:"client_name"`
			SkipConsent bool   `json:"skip_consent"`
		} `json:"client"`
	}
	if err := c.get(ctx, "/admin/oauth2/auth/requests/consent", "consent_challenge", consentChallenge, &resp); err != nil {
		return apphttp.HydraConsentRequest{}, err
	}
	return apphttp.HydraConsentRequest{
		Subject:                      resp.Subject,
		Skip:                         resp.Skip,
		RequestedScope:               resp.RequestedScope,
		RequestedAccessTokenAudience: resp.RequestedAccessTokenAudience,
		Client: apphttp.HydraOAuthClient{
			ClientID:    resp.Client.ClientID,
			ClientName:  resp.Client.ClientName,
			SkipConsent: resp.Client.SkipConsent,
		},
	}, nil
}

func (c *FlowClient) AcceptConsentRequest(ctx context.Context, consentChallenge string, grantScope, grantAudience []string, session apphttp.ConsentSessionClaims) (string, error) {
	var resp struct {
		RedirectTo string `json:"redirect_to"`
	}
	body := map[string]any{
		"grant_scope":                 grantScope,
		"grant_access_token_audience": grantAudience,
	}
	if len(session.AccessToken) > 0 || len(session.IDToken) > 0 {
		sessionClaims := map[string]any{}
		if len(session.AccessToken) > 0 {
			sessionClaims["access_token"] = session.AccessToken
		}
		if len(session.IDToken) > 0 {
			sessionClaims["id_token"] = session.IDToken
		}
		body["session"] = sessionClaims
	}
	if err := c.put(ctx, "/admin/oauth2/auth/requests/consent/accept", "consent_challenge", consentChallenge, body, &resp); err != nil {
		return "", err
	}
	return resp.RedirectTo, nil
}

func (c *FlowClient) RejectConsentRequest(ctx context.Context, consentChallenge, reason, description string) (string, error) {
	var resp struct {
		RedirectTo string `json:"redirect_to"`
	}
	if err := c.put(ctx, "/admin/oauth2/auth/requests/consent/reject", "consent_challenge", consentChallenge, map[string]any{
		"error":             reason,
		"error_description": description,
	}, &resp); err != nil {
		return "", err
	}
	return resp.RedirectTo, nil
}

func (c *FlowClient) GetLogoutRequest(ctx context.Context, logoutChallenge string) (apphttp.HydraLogoutRequest, error) {
	var resp struct {
		Subject string `json:"subject"`
	}
	if err := c.get(ctx, "/admin/oauth2/auth/requests/logout", "logout_challenge", logoutChallenge, &resp); err != nil {
		return apphttp.HydraLogoutRequest{}, err
	}
	return apphttp.HydraLogoutRequest{Subject: resp.Subject}, nil
}

func (c *FlowClient) AcceptLogoutRequest(ctx context.Context, logoutChallenge string) (string, error) {
	var resp struct {
		RedirectTo string `json:"redirect_to"`
	}
	if err := c.put(ctx, "/admin/oauth2/auth/requests/logout/accept", "logout_challenge", logoutChallenge, map[string]any{}, &resp); err != nil {
		return "", err
	}
	return resp.RedirectTo, nil
}

func (c *FlowClient) get(ctx context.Context, path, queryName, queryValue string, dst any) error {
	endpoint := c.baseURL + path + "?" + url.Values{queryName: []string{queryValue}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build hydra get request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call hydra get request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("hydra get request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode hydra get response: %w", err)
	}
	return nil
}

func (c *FlowClient) put(ctx context.Context, path, queryName, queryValue string, payload any, dst any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal hydra put request: %w", err)
	}
	endpoint := c.baseURL + path + "?" + url.Values{queryName: []string{queryValue}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build hydra put request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call hydra put request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("hydra put request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode hydra put response: %w", err)
	}
	return nil
}
