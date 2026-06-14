package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/kuro48/idol-auth/internal/domain/profile"
	"github.com/kuro48/idol-auth/internal/oshi"
)

var ErrNoActiveSession = errors.New("no active session")

var ErrSettingsFlowExpired = errors.New("settings flow expired or not found")

var (
	ErrInvalidOshiColor           = errors.New("invalid oshi color")
	ErrThemePreferenceUnavailable = errors.New("theme preference unavailable")
)

type KratosSettingsMessage struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Type string `json:"type"`
}

type KratosSettingsNode struct {
	Type      string                  `json:"type"`
	Group     string                  `json:"group"`
	Name      string                  `json:"name"`
	InputType string                  `json:"input_type"`
	Value     string                  `json:"value"`
	Label     string                  `json:"label"`
	Required  bool                    `json:"required"`
	Disabled  bool                    `json:"disabled"`
	Messages  []KratosSettingsMessage `json:"messages,omitempty"`
}

type KratosSettingsFlow struct {
	ID       string                  `json:"id"`
	Action   string                  `json:"action"`
	Method   string                  `json:"method"`
	Nodes    []KratosSettingsNode    `json:"nodes"`
	Messages []KratosSettingsMessage `json:"messages,omitempty"`
}

type KratosSettingsSubmitResult struct {
	RedirectTo string
	Flow       *KratosSettingsFlow
	SetCookies []string
}

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
	Phone                       string
	DisplayName                 string
	OshiColor                   string
	Oshis                       []profile.OshiEntry
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
	LogoutBrowser(ctx context.Context, r *http.Request) error
	BrowserLoginURL(returnTo string) string
	BrowserSettingsURL(returnTo string) string
	GetSettingsFlow(ctx context.Context, r *http.Request, flowID string) (*KratosSettingsFlow, error)
	SubmitSettingsFlow(ctx context.Context, r *http.Request, flowID string, form url.Values) (KratosSettingsSubmitResult, error)
}

type ThemePreferenceUpdater interface {
	SetIdentityOshiColor(ctx context.Context, identityID, color string) error
}

type MembershipRecorder interface {
	EnsureMembershipForHydraClient(ctx context.Context, hydraClientID, identityID, actorID string) error
}

type authService struct {
	baseURL      string
	hydra        HydraAuthClient
	kratos       KratosAuthClient
	themeUpdater ThemePreferenceUpdater
	memberships  MembershipRecorder
}

func scopeContains(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
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

func NewAuthService(baseURL string, hydra HydraAuthClient, kratos KratosAuthClient, themeUpdaters ...ThemePreferenceUpdater) AuthService {
	return NewAuthServiceWithOptions(baseURL, hydra, kratos, nil, themeUpdaters...)
}

func NewAuthServiceWithOptions(baseURL string, hydra HydraAuthClient, kratos KratosAuthClient, memberships MembershipRecorder, themeUpdaters ...ThemePreferenceUpdater) AuthService {
	var themeUpdater ThemePreferenceUpdater
	if len(themeUpdaters) > 0 {
		themeUpdater = themeUpdaters[0]
	}
	return &authService{
		baseURL:      strings.TrimRight(baseURL, "/"),
		hydra:        hydra,
		kratos:       kratos,
		themeUpdater: themeUpdater,
		memberships:  memberships,
	}
}

func (s *authService) UpdateThemePreference(ctx context.Context, r *http.Request, color string) (SessionView, error) {
	if s.themeUpdater == nil {
		return SessionView{}, ErrThemePreferenceUnavailable
	}
	normalizedColor := oshi.NormalizeColor(color)
	if normalizedColor == "" {
		return SessionView{}, ErrInvalidOshiColor
	}

	session, err := s.kratos.ToSession(ctx, r)
	if err != nil {
		return SessionView{}, err
	}
	if !session.Active || strings.TrimSpace(session.IdentityID) == "" {
		return SessionView{}, ErrNoActiveSession
	}
	if err := s.themeUpdater.SetIdentityOshiColor(ctx, session.IdentityID, normalizedColor); err != nil {
		return SessionView{}, fmt.Errorf("persist oshi color: %w", err)
	}
	session.OshiColor = normalizedColor

	return SessionView{
		Authenticated:               true,
		Subject:                     session.IdentityID,
		IdentityID:                  session.IdentityID,
		Email:                       session.Email,
		Roles:                       normalizeRoles(session.Roles),
		OshiColor:                   normalizedColor,
		Methods:                     session.Methods,
		AuthenticatorAssuranceLevel: session.AuthenticatorAssuranceLevel,
	}, nil
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
		if err := s.validateConsentSessionSubject(ctx, r, consentRequest.Subject); err != nil {
			return ConsentFlowResult{}, err
		}
		claims := s.buildConsentSessionClaims(ctx, r, consentRequest.RequestedScope)
		redirectTo, err := s.hydra.AcceptConsentRequest(ctx, consentChallenge, consentRequest.RequestedScope, consentRequest.RequestedAccessTokenAudience, claims)
		if err != nil {
			return ConsentFlowResult{}, fmt.Errorf("accept hydra consent request: %w", err)
		}
		if session, err := s.kratos.ToSession(ctx, r); err == nil && session.Active {
			_ = s.ensureMembership(ctx, consentRequest.Client.ClientID, session.IdentityID)
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
			OshiColor:                    oshi.NormalizeColor(session.OshiColor),
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
		if err := s.validateConsentSessionSubject(ctx, r, consentRequest.Subject); err != nil {
			return AuthFlowResult{}, err
		}
		claims := s.buildConsentSessionClaims(ctx, r, consentRequest.RequestedScope)
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

	claims := buildClaims(session, consentRequest.RequestedScope)
	redirectTo, err := s.hydra.AcceptConsentRequest(ctx, consentChallenge, consentRequest.RequestedScope, consentRequest.RequestedAccessTokenAudience, claims)
	if err != nil {
		return AuthFlowResult{}, fmt.Errorf("accept hydra consent request: %w", err)
	}
	_ = s.ensureMembership(ctx, consentRequest.Client.ClientID, session.IdentityID)
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

func (s *authService) LogoutSession(ctx context.Context, r *http.Request) error {
	return s.kratos.LogoutBrowser(ctx, r)
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
		Phone:                       session.Phone,
		DisplayName:                 session.DisplayName,
		Roles:                       normalizeRoles(session.Roles),
		OshiColor:                   oshi.NormalizeColor(session.OshiColor),
		Methods:                     session.Methods,
		AuthenticatorAssuranceLevel: session.AuthenticatorAssuranceLevel,
	}, nil
}

func (s *authService) GetSettingsFlow(ctx context.Context, r *http.Request, flowID string) (*KratosSettingsFlow, error) {
	return s.kratos.GetSettingsFlow(ctx, r, flowID)
}

func (s *authService) SubmitSettingsFlow(ctx context.Context, r *http.Request, flowID string, form url.Values) (KratosSettingsSubmitResult, error) {
	return s.kratos.SubmitSettingsFlow(ctx, r, flowID, form)
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

func (s *authService) buildConsentSessionClaims(ctx context.Context, r *http.Request, scopes []string) ConsentSessionClaims {
	session, err := s.kratos.ToSession(ctx, r)
	if err != nil || !session.Active {
		return ConsentSessionClaims{}
	}
	return buildClaims(session, scopes)
}

func buildClaims(session KratosSession, scopes []string) ConsentSessionClaims {
	atClaims := map[string]any{}
	idClaims := map[string]any{}
	roles := normalizeRoles(session.Roles)
	if len(roles) > 0 {
		atClaims["roles"] = roles
		idClaims["roles"] = roles
	}
	if scopeContains(scopes, "email") && session.Email != "" {
		idClaims["email"] = session.Email
	}
	if scopeContains(scopes, "profile") {
		if session.DisplayName != "" {
			idClaims["display_name"] = session.DisplayName
		}
		if session.OshiColor != "" {
			idClaims["oshi_color"] = session.OshiColor
		}
		if len(session.Oshis) > 0 {
			idClaims["oshis"] = session.Oshis
		}
	}
	if len(atClaims) == 0 && len(idClaims) == 0 {
		return ConsentSessionClaims{}
	}
	return ConsentSessionClaims{AccessToken: atClaims, IDToken: idClaims}
}

func (s *authService) validateConsentSessionSubject(ctx context.Context, r *http.Request, subject string) error {
	if strings.TrimSpace(subject) == "" {
		return nil
	}
	session, err := s.kratos.ToSession(ctx, r)
	if err != nil || !session.Active {
		return nil
	}
	if session.IdentityID != subject {
		return ErrConsentSessionMismatch
	}
	return nil
}

func sessionMFASatisfied(session KratosSession) bool {
	return strings.EqualFold(strings.TrimSpace(session.AuthenticatorAssuranceLevel), "aal2")
}

func (s *authService) ensureMembership(ctx context.Context, hydraClientID, identityID string) error {
	if s.memberships == nil {
		return nil
	}
	return s.memberships.EnsureMembershipForHydraClient(ctx, hydraClientID, identityID, identityID)
}
