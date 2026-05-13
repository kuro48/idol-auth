package kratos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apphttp "github.com/kuro48/idol-auth/internal/http"
)

// NativeClient drives Kratos headless (API-mode) self-service flows.
// It initiates a flow, then submits credentials in a single call,
// hiding the two-step Kratos protocol from callers.
type NativeClient struct {
	apiBaseURL string
	httpClient *http.Client
}

func NewNativeClient(apiBaseURL string) *NativeClient {
	return &NativeClient{
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Register creates a new Kratos identity via the headless registration flow.
func (c *NativeClient) Register(ctx context.Context, email, displayName, password string) (apphttp.AuthResult, error) {
	flowID, err := c.initFlow(ctx, "/self-service/registration/api")
	if err != nil {
		return apphttp.AuthResult{}, fmt.Errorf("init registration flow: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"method":   "password",
		"password": password,
		"traits": map[string]string{
			"email":        email,
			"display_name": displayName,
		},
	})

	var decoded struct {
		SessionToken string `json:"session_token"`
		Identity     struct {
			ID     string `json:"id"`
			Traits struct {
				Email string `json:"email"`
			} `json:"traits"`
		} `json:"identity"`
	}
	if err := c.submit(ctx, "/self-service/registration", flowID, payload, &decoded); err != nil {
		return apphttp.AuthResult{}, fmt.Errorf("submit registration: %w", err)
	}

	return apphttp.AuthResult{
		SessionToken: decoded.SessionToken,
		IdentityID:   decoded.Identity.ID,
		Email:        decoded.Identity.Traits.Email,
	}, nil
}

// Login authenticates an existing Kratos identity via the headless login flow.
func (c *NativeClient) Login(ctx context.Context, identifier, password string) (apphttp.AuthResult, error) {
	flowID, err := c.initFlow(ctx, "/self-service/login/api")
	if err != nil {
		return apphttp.AuthResult{}, fmt.Errorf("init login flow: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"method":     "password",
		"identifier": identifier,
		"password":   password,
	})

	var decoded struct {
		SessionToken string `json:"session_token"`
		Identity     struct {
			ID     string `json:"id"`
			Traits struct {
				Email string `json:"email"`
			} `json:"traits"`
		} `json:"identity"`
	}
	if err := c.submit(ctx, "/self-service/login", flowID, payload, &decoded); err != nil {
		return apphttp.AuthResult{}, fmt.Errorf("submit login: %w", err)
	}

	return apphttp.AuthResult{
		SessionToken: decoded.SessionToken,
		IdentityID:   decoded.Identity.ID,
		Email:        decoded.Identity.Traits.Email,
	}, nil
}

func (c *NativeClient) initFlow(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call kratos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("kratos returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var flow struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&flow); err != nil {
		return "", fmt.Errorf("decode flow: %w", err)
	}
	return flow.ID, nil
}

func (c *NativeClient) submit(ctx context.Context, path, flowID string, payload []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.apiBaseURL+path+"?flow="+flowID,
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call kratos: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return extractKratosError(body, resp.StatusCode)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// extractKratosError returns a descriptive error from a non-2xx Kratos response.
// Returns apphttp.ErrIdentityAlreadyExists for duplicate-identity conditions.
func extractKratosError(body []byte, status int) error {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
		UI struct {
			Messages []struct {
				Text string `json:"text"`
			} `json:"messages"`
		} `json:"ui"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		if status == 409 {
			return apphttp.ErrIdentityAlreadyExists
		}
		return fmt.Errorf("kratos returned status %d", status)
	}
	if envelope.Error.Message != "" {
		msg := envelope.Error.Message
		if envelope.Error.Reason != "" {
			msg += ": " + envelope.Error.Reason
		}
		if isAlreadyExistsMsg(msg) || status == 409 {
			return apphttp.ErrIdentityAlreadyExists
		}
		return fmt.Errorf("%s (status %d)", msg, status)
	}
	for _, m := range envelope.UI.Messages {
		if m.Text != "" {
			if isAlreadyExistsMsg(m.Text) {
				return apphttp.ErrIdentityAlreadyExists
			}
			return fmt.Errorf("%s (status %d)", m.Text, status)
		}
	}
	if status == 409 {
		return apphttp.ErrIdentityAlreadyExists
	}
	return fmt.Errorf("kratos returned status %d", status)
}

func isAlreadyExistsMsg(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "already exists") ||
		strings.Contains(lower, "already registered") ||
		strings.Contains(lower, "account with the same")
}
