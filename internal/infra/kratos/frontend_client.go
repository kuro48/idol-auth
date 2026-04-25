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

	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
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
				Email string `json:"email"`
			} `json:"traits"`
			MetadataPublic struct {
				Roles []string `json:"roles"`
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
		Roles:                       decoded.Identity.MetadataPublic.Roles,
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
