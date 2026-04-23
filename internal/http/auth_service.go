package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var ErrNoActiveSession = errors.New("no active session")

type HydraLoginRequest struct {
	Skip    bool
	Subject string
}

type HydraConsentRequest struct {
	Subject                      string
	Skip                         bool
	RequestedScope               []string
	RequestedAccessTokenAudience []string
	Client                       HydraOAuthClient
}

type HydraOAuthClient struct {
	ClientID    string
	ClientName  string
	SkipConsent bool
}

type HydraLogoutRequest struct {
	Subject string
}

type KratosSession struct {
	Active                      bool
	IdentityID                  string
	Email                       string
	Roles                       []string
	Methods                     []string
	AuthenticatorAssuranceLevel string
}

// ConsentSessionClaims holds custom JWT claims injected into access and ID tokens
// when Hydra accepts a consent request.
type ConsentSessionClaims struct {
	AccessToken map[string]any
	IDToken     map[string]any
}

type HydraAuthClient interface {
	GetLoginRequest(ctx context.Context, loginChallenge string) (HydraLoginRequest, error)
	AcceptLoginRequest(ctx context.Context, loginChallenge, subject string) (string, error)
	GetConsentRequest(ctx context.Context, consentChallenge string) (HydraConsentRequest, error)
	AcceptConsentRequest(ctx context.Context, consentChallenge string, grantScope, grantAudience []string, session ConsentSessionClaims) (string, error)
	RejectConsentRequest(ctx context.Context, consentChallenge, reason, description string) (string, error)
	GetLogoutRequest(ctx context.Context, logoutChallenge string) (HydraLogoutRequest, error)
	AcceptLogoutRequest(ctx context.Context, logoutChallenge string) (string, error)
}

type KratosAuthClient interface {
	ToSession(ctx context.Context, r *http.Request) (KratosSession, error)
	BrowserLoginURL(returnTo string) string
	BrowserSettingsURL(returnTo string) string
}

type authService struct {
	baseURL string
	hydra   HydraAuthClient
	kratos  KratosAuthClient
}

func normalizeRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(roles))
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		normalized := strings.TrimSpace(strings.ToLower(role))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func NewAuthService(baseURL string, hydra HydraAuthClient, kratos KratosAuthClient) AuthService {
	return &authService{
		baseURL: strings.TrimRight(baseURL, "/"),
		hydra:   hydra,
		kratos:  kratos,
	}
}

func (s *authService) HandleLogin(ctx context.Context, r *http.Request, loginChallenge string) (LoginFlowResult, error) {
	if strings.TrimSpace(loginChallenge) == "" {
		return LoginFlowResult{}, ErrChallengeRequired
	}

	loginRequest, err := s.hydra.GetLoginRequest(ctx, loginChallenge)
	if err != nil {
		return LoginFlowResult{}, fmt.Errorf("get hydra login request: %w", err)
	}
	if loginRequest.Skip && loginRequest.Subject != "" {
		redirectTo, err := s.hydra.AcceptLoginRequest(ctx, loginChallenge, loginRequest.Subject)
		if err != nil {
			return LoginFlowResult{}, fmt.Errorf("accept hydra login request: %w", err)
		}
		return LoginFlowResult{Action: AuthActionRedirect, RedirectTo: redirectTo}, nil
	}

	session, err := s.kratos.ToSession(ctx, r)
	if err != nil {
		if !errors.Is(err, ErrNoActiveSession) {
			return LoginFlowResult{}, fmt.Errorf("resolve kratos session: %w", err)
		}
		return LoginFlowResult{
			Action:     AuthActionRedirect,
			RedirectTo: s.kratos.BrowserLoginURL(s.loginReturnURL(loginChallenge)),
		}, nil
	}
	if !session.Active {
		return LoginFlowResult{
			Action:     AuthActionRedirect,
			RedirectTo: s.kratos.BrowserLoginURL(s.loginReturnURL(loginChallenge)),
		}, nil
	}
	if !sessionMFASatisfied(session) {
		return LoginFlowResult{
			Action:     AuthActionRedirect,
			RedirectTo: s.kratos.BrowserSettingsURL(s.loginReturnURL(loginChallenge)),
		}, nil
	}

	redirectTo, err := s.hydra.AcceptLoginRequest(ctx, loginChallenge, session.IdentityID)
	if err != nil {
		return LoginFlowResult{}, fmt.Errorf("accept hydra login request: %w", err)
	}
	return LoginFlowResult{Action: AuthActionRedirect, RedirectTo: redirectTo}, nil
}

func (s *authService) HandleConsent(ctx context.Context, r *http.Request, consentChallenge string) (ConsentFlowResult, error) {
	if strings.TrimSpace(consentChallenge) == "" {
		return ConsentFlowResult{}, ErrChallengeRequired
	}
	consentRequest, err := s.hydra.GetConsentRequest(ctx, consentChallenge)
	if err != nil {
		return ConsentFlowResult{}, fmt.Errorf("get hydra consent request: %w", err)
	}
	if consentRequest.Skip || consentRequest.Client.SkipConsent {
		claims := s.buildConsentSessionClaims(ctx, r)
		redirectTo, err := s.hydra.AcceptConsentRequest(ctx, consentChallenge, consentRequest.RequestedScope, consentRequest.RequestedAccessTokenAudience, claims)
		if err != nil {
			return ConsentFlowResult{}, fmt.Errorf("accept hydra consent request: %w", err)
		}
		return ConsentFlowResult{RedirectTo: redirectTo}, nil
	}

	session, err := s.kratos.ToSession(ctx, r)
	if err != nil {
		if !errors.Is(err, ErrNoActiveSession) {
			return ConsentFlowResult{}, fmt.Errorf("resolve kratos session: %w", err)
		}
		return ConsentFlowResult{
			RedirectTo: s.kratos.BrowserLoginURL(s.consentReturnURL(consentChallenge)),
		}, nil
	}
	if !session.Active {
		return ConsentFlowResult{
			RedirectTo: s.kratos.BrowserLoginURL(s.consentReturnURL(consentChallenge)),
		}, nil
	}
	if !sessionMFASatisfied(session) {
		return ConsentFlowResult{
			RedirectTo: s.kratos.BrowserSettingsURL(s.consentReturnURL(consentChallenge)),
		}, nil
	}
	if consentRequest.Subject != "" && consentRequest.Subject != session.IdentityID {
		return ConsentFlowResult{}, ErrConsentSessionMismatch
	}

	return ConsentFlowResult{
		Prompt: &ConsentPrompt{
			Challenge:                    consentChallenge,
			ClientID:                     consentRequest.Client.ClientID,
			ClientName:                   consentRequest.Client.ClientName,
			RequestedScope:               consentRequest.RequestedScope,
			RequestedAccessTokenAudience: consentRequest.RequestedAccessTokenAudience,
		},
	}, nil
}

func (s *authService) SubmitConsent(ctx context.Context, r *http.Request, consentChallenge string, input ConsentDecisionInput) (AuthFlowResult, error) {
	if strings.TrimSpace(consentChallenge) == "" {
		return AuthFlowResult{}, ErrChallengeRequired
	}
	consentRequest, err := s.hydra.GetConsentRequest(ctx, consentChallenge)
	if err != nil {
		return AuthFlowResult{}, fmt.Errorf("get hydra consent request: %w", err)
	}
	if consentRequest.Skip || consentRequest.Client.SkipConsent {
		claims := s.buildConsentSessionClaims(ctx, r)
		redirectTo, err := s.hydra.AcceptConsentRequest(ctx, consentChallenge, consentRequest.RequestedScope, consentRequest.RequestedAccessTokenAudience, claims)
		if err != nil {
			return AuthFlowResult{}, fmt.Errorf("accept hydra consent request: %w", err)
		}
		return AuthFlowResult{RedirectTo: redirectTo}, nil
	}

	session, err := s.kratos.ToSession(ctx, r)
	if err != nil {
		if !errors.Is(err, ErrNoActiveSession) {
			return AuthFlowResult{}, fmt.Errorf("resolve kratos session: %w", err)
		}
		return AuthFlowResult{RedirectTo: s.kratos.BrowserLoginURL(s.consentReturnURL(consentChallenge))}, nil
	}
	if !session.Active {
		return AuthFlowResult{RedirectTo: s.kratos.BrowserLoginURL(s.consentReturnURL(consentChallenge))}, nil
	}
	if !sessionMFASatisfied(session) {
		return AuthFlowResult{RedirectTo: s.kratos.BrowserSettingsURL(s.consentReturnURL(consentChallenge))}, nil
	}
	if consentRequest.Subject != "" && consentRequest.Subject != session.IdentityID {
		return AuthFlowResult{}, ErrConsentSessionMismatch
	}

	if !input.Accept {
		redirectTo, err := s.hydra.RejectConsentRequest(ctx, consentChallenge, "access_denied", "resource owner denied the request")
		if err != nil {
			return AuthFlowResult{}, fmt.Errorf("reject hydra consent request: %w", err)
		}
		return AuthFlowResult{RedirectTo: redirectTo}, nil
	}

	roles := normalizeRoles(session.Roles)
	claims := ConsentSessionClaims{}
	if len(roles) > 0 {
		claims = ConsentSessionClaims{
			AccessToken: map[string]any{"roles": roles},
			IDToken:     map[string]any{"roles": roles},
		}
	}
	redirectTo, err := s.hydra.AcceptConsentRequest(ctx, consentChallenge, consentRequest.RequestedScope, consentRequest.RequestedAccessTokenAudience, claims)
	if err != nil {
		return AuthFlowResult{}, fmt.Errorf("accept hydra consent request: %w", err)
	}
	return AuthFlowResult{RedirectTo: redirectTo}, nil
}

func (s *authService) HandleLogout(ctx context.Context, logoutChallenge string) (AuthFlowResult, error) {
	if strings.TrimSpace(logoutChallenge) == "" {
		return AuthFlowResult{}, ErrChallengeRequired
	}
	if _, err := s.hydra.GetLogoutRequest(ctx, logoutChallenge); err != nil {
		return AuthFlowResult{}, fmt.Errorf("get hydra logout request: %w", err)
	}
	redirectTo, err := s.hydra.AcceptLogoutRequest(ctx, logoutChallenge)
	if err != nil {
		return AuthFlowResult{}, fmt.Errorf("accept hydra logout request: %w", err)
	}
	return AuthFlowResult{RedirectTo: redirectTo}, nil
}

func (s *authService) CurrentSession(ctx context.Context, r *http.Request) (SessionView, error) {
	session, err := s.kratos.ToSession(ctx, r)
	if err != nil {
		if errors.Is(err, ErrNoActiveSession) {
			return SessionView{Authenticated: false}, nil
		}
		return SessionView{}, fmt.Errorf("resolve kratos session: %w", err)
	}
	if !session.Active {
		return SessionView{Authenticated: false}, nil
	}
	return SessionView{
		Authenticated:               true,
		Subject:                     session.IdentityID,
		IdentityID:                  session.IdentityID,
		Email:                       session.Email,
		Roles:                       normalizeRoles(session.Roles),
		Methods:                     session.Methods,
		AuthenticatorAssuranceLevel: session.AuthenticatorAssuranceLevel,
	}, nil
}

func (s *authService) loginReturnURL(loginChallenge string) string {
	u, _ := url.Parse(s.baseURL + "/v1/auth/login")
	q := u.Query()
	q.Set("login_challenge", loginChallenge)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *authService) consentReturnURL(consentChallenge string) string {
	u, _ := url.Parse(s.baseURL + "/v1/auth/consent")
	q := u.Query()
	q.Set("consent_challenge", consentChallenge)
	u.RawQuery = q.Encode()
	return u.String()
}

// buildConsentSessionClaims fetches the active Kratos session and returns claims
// containing the session's normalized roles. Returns empty claims on any error or
// when the session is inactive, so callers always get a valid (possibly empty) value.
func (s *authService) buildConsentSessionClaims(ctx context.Context, r *http.Request) ConsentSessionClaims {
	session, err := s.kratos.ToSession(ctx, r)
	if err != nil || !session.Active {
		return ConsentSessionClaims{}
	}
	roles := normalizeRoles(session.Roles)
	if len(roles) == 0 {
		return ConsentSessionClaims{}
	}
	return ConsentSessionClaims{
		AccessToken: map[string]any{"roles": roles},
		IDToken:     map[string]any{"roles": roles},
	}
}

func sessionMFASatisfied(session KratosSession) bool {
	return strings.EqualFold(strings.TrimSpace(session.AuthenticatorAssuranceLevel), "aal2")
}
