package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PublicSessionView is the response shape for GET /v1/public/api/session.
type PublicSessionView struct {
	Active      bool     `json:"active"`
	Subject     string   `json:"subject,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	OshiColor   string   `json:"oshi_color,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
}

// HydraFacadeClient is implemented by infra/hydra.FacadeClient.
type HydraFacadeClient interface {
	Token(ctx context.Context, body []byte) ([]byte, int, error)
	Revoke(ctx context.Context, body []byte) ([]byte, int, error)
	Introspect(ctx context.Context, body []byte) ([]byte, int, error)
}

// PublicAuthService is the interface the router uses for the public facade.
type PublicAuthService interface {
	LoginURL(params map[string]string) string
	LogoutURL(params map[string]string) string
	RegistrationURL(returnTo string) string
	Token(ctx context.Context, body []byte) ([]byte, int, error)
	Revoke(ctx context.Context, body []byte) ([]byte, int, error)
	Introspect(ctx context.Context, body []byte) ([]byte, int, error)
	GetSession(ctx context.Context, token string) (PublicSessionView, error)
}

// PublicAuthServiceImpl implements PublicAuthService.
type PublicAuthServiceImpl struct {
	hydra            HydraFacadeClient
	hydraBrowserURL  string
	kratosBrowserURL string
	kratosPublicURL  string
	httpClient       *http.Client
}

func NewPublicAuthService(
	hydra HydraFacadeClient,
	hydraBrowserURL string,
	kratosBrowserURL string,
	kratosPublicURL string,
) *PublicAuthServiceImpl {
	return &PublicAuthServiceImpl{
		hydra:            hydra,
		hydraBrowserURL:  strings.TrimRight(hydraBrowserURL, "/"),
		kratosBrowserURL: strings.TrimRight(kratosBrowserURL, "/"),
		kratosPublicURL:  strings.TrimRight(kratosPublicURL, "/"),
		httpClient:       &http.Client{Timeout: 10 * time.Second},
	}
}

// LoginURL builds the Hydra authorization URL for a browser-initiated login.
func (s *PublicAuthServiceImpl) LoginURL(params map[string]string) string {
	return s.hydraBrowserURL + "/oauth2/auth?" + buildQueryFromMap(params)
}

// LogoutURL builds the Hydra end-session URL for a browser-initiated logout.
func (s *PublicAuthServiceImpl) LogoutURL(params map[string]string) string {
	base := s.hydraBrowserURL + "/oauth2/sessions/logout"
	if q := buildQueryFromMap(params); q != "" {
		return base + "?" + q
	}
	return base
}

// RegistrationURL builds the Kratos browser registration URL.
func (s *PublicAuthServiceImpl) RegistrationURL(returnTo string) string {
	base := s.kratosBrowserURL + "/self-service/registration/browser"
	if returnTo != "" {
		return base + "?return_to=" + returnTo
	}
	return base
}

func (s *PublicAuthServiceImpl) Token(ctx context.Context, body []byte) ([]byte, int, error) {
	return s.hydra.Token(ctx, body)
}

func (s *PublicAuthServiceImpl) Revoke(ctx context.Context, body []byte) ([]byte, int, error) {
	return s.hydra.Revoke(ctx, body)
}

func (s *PublicAuthServiceImpl) Introspect(ctx context.Context, body []byte) ([]byte, int, error) {
	return s.hydra.Introspect(ctx, body)
}

// GetSession calls Kratos /sessions/whoami with X-Session-Token to look up
// the session for the provided bearer token.
func (s *PublicAuthServiceImpl) GetSession(ctx context.Context, token string) (PublicSessionView, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.kratosPublicURL+"/sessions/whoami", nil)
	if err != nil {
		return PublicSessionView{}, fmt.Errorf("build kratos whoami request: %w", err)
	}
	req.Header.Set("X-Session-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return PublicSessionView{}, fmt.Errorf("call kratos whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return PublicSessionView{Active: false}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return PublicSessionView{}, fmt.Errorf("kratos whoami returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded struct {
		Active   bool `json:"active"`
		Identity struct {
			ID     string `json:"id"`
			Traits struct {
				Email       string `json:"email"`
				DisplayName string `json:"display_name"`
			} `json:"traits"`
			MetadataPublic struct {
				Roles     []string `json:"roles"`
				OshiColor string   `json:"oshi_color"`
			} `json:"metadata_public"`
		} `json:"identity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return PublicSessionView{}, fmt.Errorf("decode kratos whoami response: %w", err)
	}

	return PublicSessionView{
		Active:      decoded.Active,
		Subject:     decoded.Identity.ID,
		Email:       decoded.Identity.Traits.Email,
		DisplayName: decoded.Identity.Traits.DisplayName,
		Roles:       decoded.Identity.MetadataPublic.Roles,
		OshiColor:   decoded.Identity.MetadataPublic.OshiColor,
	}, nil
}

// buildQueryFromMap builds a URL query string from a map, omitting empty values.
func buildQueryFromMap(params map[string]string) string {
	parts := make([]string, 0, len(params))
	for k, v := range params {
		if v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, "&")
}
