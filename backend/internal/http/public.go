package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrIdentityAlreadyExists is returned by Register when the email is already taken.
var ErrIdentityAlreadyExists = errors.New("identity already exists")

// PublicSessionView is the response shape for GET /v1/public/api/session.
type PublicSessionView struct {
	Active      bool     `json:"active"`
	Subject     string   `json:"subject,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	OshiColor   string   `json:"oshi_color,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
}

// AuthResult is returned by headless Register and Login.
type AuthResult struct {
	SessionToken string `json:"session_token,omitempty"`
	IdentityID   string `json:"identity_id"`
	Email        string `json:"email"`
}

// RegisterInput is the request body for POST /v1/public/api/register.
type RegisterInput struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// LoginInput is the request body for POST /v1/public/api/login.
type LoginInput struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// HydraFacadeClient is implemented by infra/hydra.FacadeClient.
type HydraFacadeClient interface {
	Token(ctx context.Context, body []byte) ([]byte, int, error)
	Revoke(ctx context.Context, body []byte) ([]byte, int, error)
	Introspect(ctx context.Context, body []byte) ([]byte, int, error)
}

// KratosNativeClient is implemented by infra/kratos.NativeClient.
type KratosNativeClient interface {
	Register(ctx context.Context, email, displayName, password string) (AuthResult, error)
	Login(ctx context.Context, identifier, password string) (AuthResult, error)
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
	Register(ctx context.Context, input RegisterInput) (AuthResult, error)
	Login(ctx context.Context, input LoginInput) (AuthResult, error)
}

// PublicAuthServiceImpl implements PublicAuthService.
type PublicAuthServiceImpl struct {
	hydra            HydraFacadeClient
	kratosNative     KratosNativeClient
	hydraBrowserURL  string
	kratosBrowserURL string
	kratosPublicURL  string
	httpClient       *http.Client
}

func NewPublicAuthService(
	hydra HydraFacadeClient,
	kratosNative KratosNativeClient,
	hydraBrowserURL string,
	kratosBrowserURL string,
	kratosPublicURL string,
) *PublicAuthServiceImpl {
	return &PublicAuthServiceImpl{
		hydra:            hydra,
		kratosNative:     kratosNative,
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
		return base + "?return_to=" + url.QueryEscape(returnTo)
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
	// #nosec G704 -- kratosPublicURL comes from trusted server configuration,
	// never from the bearer token or request parameters.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.kratosPublicURL+"/sessions/whoami", nil)
	if err != nil {
		return PublicSessionView{}, fmt.Errorf("build kratos whoami request: %w", err)
	}
	req.Header.Set("X-Session-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req) // #nosec G704 -- request target is the fixed Kratos public URL above.
	if err != nil {
		return PublicSessionView{}, fmt.Errorf("call kratos whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
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

func (s *PublicAuthServiceImpl) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	return s.kratosNative.Register(ctx, input.Email, input.DisplayName, input.Password)
}

func (s *PublicAuthServiceImpl) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	return s.kratosNative.Login(ctx, input.Identifier, input.Password)
}

// buildQueryFromMap builds a URL query string from a map, omitting empty values.
// Uses url.Values to ensure proper encoding and stable key ordering.
func buildQueryFromMap(params map[string]string) string {
	v := url.Values{}
	for k, val := range params {
		if val != "" {
			v.Set(k, val)
		}
	}
	return v.Encode()
}
