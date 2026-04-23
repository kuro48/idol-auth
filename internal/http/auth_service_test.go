package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthServiceHandleLoginRedirectsOnNoActiveSession(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		loginRequest: HydraLoginRequest{},
	}, &stubKratosAuthClient{
		sessionErr: ErrNoActiveSession,
	})

	result, err := svc.HandleLogin(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleLogin() error = %v", err)
	}
	if result.RedirectTo == "" {
		t.Fatal("expected redirect url")
	}
}

func TestAuthServiceHandleLoginFailsOnKratosUpstreamError(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		loginRequest: HydraLoginRequest{},
	}, &stubKratosAuthClient{
		sessionErr: errors.New("kratos unavailable"),
	})

	_, err := svc.HandleLogin(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthServiceHandleConsentReturnsPromptForInteractiveConsent(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject:        "identity-123",
			RequestedScope: []string{"openid"},
			Client: HydraOAuthClient{
				ClientID:    "third-party",
				ClientName:  "Third Party App",
				SkipConsent: false,
			},
		},
	}, &stubKratosAuthClient{
		session: KratosSession{
			Active:                      true,
			IdentityID:                  "identity-123",
			AuthenticatorAssuranceLevel: "aal2",
			Methods:                     []string{"password", "totp"},
		},
	})

	result, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleConsent() error = %v", err)
	}
	if result.Prompt == nil {
		t.Fatal("expected consent prompt")
	}
	if result.Prompt.ClientName != "Third Party App" {
		t.Fatalf("expected client name to be propagated, got %q", result.Prompt.ClientName)
	}
}

func TestAuthServiceSubmitConsentRejectsDeniedRequest(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject: "identity-123",
			Client: HydraOAuthClient{
				ClientID: "third-party",
			},
		},
		rejectConsentRedirect: "http://hydra/rejected",
	}, &stubKratosAuthClient{
		session: KratosSession{
			Active:                      true,
			IdentityID:                  "identity-123",
			AuthenticatorAssuranceLevel: "aal2",
			Methods:                     []string{"password", "totp"},
		},
	})

	result, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: false})
	if err != nil {
		t.Fatalf("SubmitConsent() error = %v", err)
	}
	if result.RedirectTo != "http://hydra/rejected" {
		t.Fatalf("expected reject redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceHandleConsentRedirectsToLoginWhenSessionMissing(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Client: HydraOAuthClient{
				ClientID: "third-party",
			},
		},
	}, &stubKratosAuthClient{
		sessionErr: ErrNoActiveSession,
	})

	result, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleConsent() error = %v", err)
	}
	if result.RedirectTo == "" {
		t.Fatal("expected login redirect")
	}
}

func TestAuthServiceHandleLoginSkipsWhenHydraIndicatesSkip(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		loginRequest:  HydraLoginRequest{Skip: true, Subject: "identity-123"},
		loginRedirect: "http://hydra/auth?login_verifier=skip123",
	}, &stubKratosAuthClient{})

	result, err := svc.HandleLogin(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleLogin() error = %v", err)
	}
	if result.RedirectTo != "http://hydra/auth?login_verifier=skip123" {
		t.Fatalf("expected hydra skip redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceHandleLoginRedirectsInactiveSession(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		loginRequest: HydraLoginRequest{},
	}, &stubKratosAuthClient{
		session: KratosSession{Active: false},
	})

	result, err := svc.HandleLogin(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleLogin() error = %v", err)
	}
	if result.RedirectTo == "" {
		t.Fatal("expected kratos login redirect for inactive session")
	}
}

func TestAuthServiceHandleLoginAcceptsActiveSession(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		loginRequest:  HydraLoginRequest{},
		loginRedirect: "http://hydra/auth?login_verifier=accepted",
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal2", Methods: []string{"password", "totp"}},
	})

	result, err := svc.HandleLogin(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleLogin() error = %v", err)
	}
	if result.RedirectTo != "http://hydra/auth?login_verifier=accepted" {
		t.Fatalf("expected hydra accept redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceHandleLoginRedirectsToSettingsWhenMFANotSatisfied(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		loginRequest: HydraLoginRequest{},
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal1", Methods: []string{"password"}},
	})

	result, err := svc.HandleLogin(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleLogin() error = %v", err)
	}
	if got := result.RedirectTo; got != "http://kratos/settings?return_to=http://localhost:8080/v1/auth/login?login_challenge=challenge-1" {
		t.Fatalf("expected settings redirect, got %q", got)
	}
}

func TestAuthServiceHandleLoginFailsOnEmptyChallenge(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{})

	_, err := svc.HandleLogin(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "")
	if !errors.Is(err, ErrChallengeRequired) {
		t.Fatalf("expected ErrChallengeRequired, got %v", err)
	}
}

func TestAuthServiceHandleConsentAutoSkipsWhenHydraIndicatesSkip(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest:  HydraConsentRequest{Skip: true, RequestedScope: []string{"openid"}},
		consentRedirect: "http://hydra/auth?consent_verifier=skipped",
	}, &stubKratosAuthClient{})

	result, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleConsent() error = %v", err)
	}
	if result.RedirectTo != "http://hydra/auth?consent_verifier=skipped" {
		t.Fatalf("expected skip redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceHandleConsentAutoSkipsForFirstPartyClient(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Client: HydraOAuthClient{ClientID: "first-party", SkipConsent: true},
		},
		consentRedirect: "http://hydra/auth?consent_verifier=first-party",
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal2", Methods: []string{"password", "totp"}},
	})

	result, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleConsent() error = %v", err)
	}
	if result.RedirectTo != "http://hydra/auth?consent_verifier=first-party" {
		t.Fatalf("expected first-party redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceHandleConsentRedirectsToSettingsWhenMFANotSatisfied(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject: "identity-123",
			Client:  HydraOAuthClient{ClientID: "third-party"},
		},
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal1", Methods: []string{"password"}},
	})

	result, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleConsent() error = %v", err)
	}
	if got := result.RedirectTo; got != "http://kratos/settings?return_to=http://localhost:8080/v1/auth/consent?consent_challenge=challenge-1" {
		t.Fatalf("expected settings redirect, got %q", got)
	}
}

func TestAuthServiceHandleConsentFailsOnSessionSubjectMismatch(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject: "identity-other",
			Client:  HydraOAuthClient{ClientID: "third-party"},
		},
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal2", Methods: []string{"password", "totp"}},
	})

	_, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if !errors.Is(err, ErrConsentSessionMismatch) {
		t.Fatalf("expected ErrConsentSessionMismatch, got %v", err)
	}
}

func TestAuthServiceSubmitConsentRedirectsToSettingsWhenMFANotSatisfied(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject: "identity-123",
			Client:  HydraOAuthClient{ClientID: "third-party"},
		},
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal1", Methods: []string{"password"}},
	})

	result, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: true})
	if err != nil {
		t.Fatalf("SubmitConsent() error = %v", err)
	}
	if got := result.RedirectTo; got != "http://kratos/settings?return_to=http://localhost:8080/v1/auth/consent?consent_challenge=challenge-1" {
		t.Fatalf("expected settings redirect, got %q", got)
	}
}

func TestAuthServiceHandleConsentFailsOnEmptyChallenge(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{})

	_, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "")
	if !errors.Is(err, ErrChallengeRequired) {
		t.Fatalf("expected ErrChallengeRequired, got %v", err)
	}
}

func TestAuthServiceSubmitConsentAcceptsRequest(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject: "identity-123",
			Client:  HydraOAuthClient{ClientID: "third-party"},
		},
		consentRedirect: "http://hydra/auth?consent_verifier=accepted",
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal2", Methods: []string{"password", "totp"}},
	})

	result, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: true})
	if err != nil {
		t.Fatalf("SubmitConsent() error = %v", err)
	}
	if result.RedirectTo != "http://hydra/auth?consent_verifier=accepted" {
		t.Fatalf("expected accept redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceSubmitConsentAutoAcceptsWhenSkip(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest:  HydraConsentRequest{Skip: true},
		consentRedirect: "http://hydra/auth?consent_verifier=auto",
	}, &stubKratosAuthClient{})

	result, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: false})
	if err != nil {
		t.Fatalf("SubmitConsent() error = %v", err)
	}
	if result.RedirectTo != "http://hydra/auth?consent_verifier=auto" {
		t.Fatalf("expected auto-accept redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceSubmitConsentRedirectsToLoginWhenSessionMissing(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Client: HydraOAuthClient{ClientID: "third-party"},
		},
	}, &stubKratosAuthClient{
		sessionErr: ErrNoActiveSession,
	})

	result, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: true})
	if err != nil {
		t.Fatalf("SubmitConsent() error = %v", err)
	}
	if result.RedirectTo == "" {
		t.Fatal("expected login redirect when session missing")
	}
}

func TestAuthServiceSubmitConsentRedirectsToLoginForInactiveSession(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Client: HydraOAuthClient{ClientID: "third-party"},
		},
	}, &stubKratosAuthClient{
		session: KratosSession{Active: false},
	})

	result, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: true})
	if err != nil {
		t.Fatalf("SubmitConsent() error = %v", err)
	}
	if result.RedirectTo == "" {
		t.Fatal("expected login redirect for inactive session")
	}
}

func TestAuthServiceSubmitConsentFailsOnSessionSubjectMismatch(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject: "identity-other",
			Client:  HydraOAuthClient{ClientID: "third-party"},
		},
	}, &stubKratosAuthClient{
		session: KratosSession{Active: true, IdentityID: "identity-123", AuthenticatorAssuranceLevel: "aal2", Methods: []string{"password", "totp"}},
	})

	_, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: true})
	if !errors.Is(err, ErrConsentSessionMismatch) {
		t.Fatalf("expected ErrConsentSessionMismatch, got %v", err)
	}
}

func TestAuthServiceSubmitConsentFailsOnEmptyChallenge(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{})

	_, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "", ConsentDecisionInput{Accept: true})
	if !errors.Is(err, ErrChallengeRequired) {
		t.Fatalf("expected ErrChallengeRequired, got %v", err)
	}
}

func TestAuthServiceHandleLogoutAcceptsValidChallenge(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{
		logoutRedirect: "http://localhost:3000/logged-out",
	}, &stubKratosAuthClient{})

	result, err := svc.HandleLogout(context.Background(), "logout-challenge-1")
	if err != nil {
		t.Fatalf("HandleLogout() error = %v", err)
	}
	if result.RedirectTo != "http://localhost:3000/logged-out" {
		t.Fatalf("expected logout redirect, got %q", result.RedirectTo)
	}
}

func TestAuthServiceHandleLogoutFailsOnEmptyChallenge(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{})

	_, err := svc.HandleLogout(context.Background(), "")
	if !errors.Is(err, ErrChallengeRequired) {
		t.Fatalf("expected ErrChallengeRequired, got %v", err)
	}
}

func TestAuthServiceCurrentSessionReturnsAuthenticatedView(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{
		session: KratosSession{
			Active:                      true,
			IdentityID:                  "identity-123",
			Email:                       "user@example.com",
			Roles:                       []string{"Admin", "admin"},
			AuthenticatorAssuranceLevel: "aal2",
			Methods:                     []string{"password"},
		},
	})

	view, err := svc.CurrentSession(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("CurrentSession() error = %v", err)
	}
	if !view.Authenticated {
		t.Fatal("expected authenticated=true")
	}
	if view.IdentityID != "identity-123" {
		t.Fatalf("expected identity id, got %q", view.IdentityID)
	}
	if view.Email != "user@example.com" {
		t.Fatalf("expected email, got %q", view.Email)
	}
	if len(view.Roles) != 1 || view.Roles[0] != "admin" {
		t.Fatalf("expected deduplicated roles [admin], got %v", view.Roles)
	}
}

func TestAuthServiceCurrentSessionReturnsUnauthenticatedWhenNoSession(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{
		sessionErr: ErrNoActiveSession,
	})

	view, err := svc.CurrentSession(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("CurrentSession() error = %v", err)
	}
	if view.Authenticated {
		t.Fatal("expected authenticated=false when no session")
	}
}

func TestAuthServiceCurrentSessionReturnsUnauthenticatedForInactiveSession(t *testing.T) {
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{
		session: KratosSession{Active: false},
	})

	view, err := svc.CurrentSession(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("CurrentSession() error = %v", err)
	}
	if view.Authenticated {
		t.Fatal("expected authenticated=false for inactive session")
	}
}

func TestAuthServiceCurrentSessionForwardsUpstreamError(t *testing.T) {
	upstreamErr := errors.New("kratos unavailable")
	svc := NewAuthService("http://localhost:8080", &stubHydraAuthClient{}, &stubKratosAuthClient{
		sessionErr: upstreamErr,
	})

	_, err := svc.CurrentSession(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("expected upstream error, got %v", err)
	}
}

func TestNormalizeRolesDeduplicatesAndLowercases(t *testing.T) {
	got := normalizeRoles([]string{" Admin ", "platform-OPERATOR", "admin", ""})
	want := []string{"admin", "platform-operator"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestNormalizeRolesReturnsEmptyForEmptyInput(t *testing.T) {
	got := normalizeRoles([]string{})
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestAuthServiceSubmitConsentInjectsSessionRolesIntoClaims(t *testing.T) {
	stub := &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Subject: "identity-123",
			Client:  HydraOAuthClient{ClientID: "third-party"},
		},
		consentRedirect: "http://hydra/auth?consent_verifier=accepted",
	}
	svc := NewAuthService("http://localhost:8080", stub, &stubKratosAuthClient{
		session: KratosSession{
			Active:                      true,
			IdentityID:                  "identity-123",
			Roles:                       []string{"admin", "editor"},
			AuthenticatorAssuranceLevel: "aal2",
			Methods:                     []string{"password", "totp"},
		},
	})

	_, err := svc.SubmitConsent(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil), "challenge-1", ConsentDecisionInput{Accept: true})
	if err != nil {
		t.Fatalf("SubmitConsent() error = %v", err)
	}
	roles, ok := stub.receivedSession.AccessToken["roles"]
	if !ok {
		t.Fatal("expected roles in access_token claims")
	}
	roleList, ok := roles.([]string)
	if !ok {
		t.Fatalf("expected []string roles, got %T", roles)
	}
	if len(roleList) != 2 || roleList[0] != "admin" || roleList[1] != "editor" {
		t.Fatalf("expected [admin editor] in access_token claims, got %v", roleList)
	}
	idRoles, ok := stub.receivedSession.IDToken["roles"]
	if !ok {
		t.Fatal("expected roles in id_token claims")
	}
	if idRoleList, ok := idRoles.([]string); !ok || len(idRoleList) != 2 {
		t.Fatalf("expected 2 roles in id_token claims, got %v", idRoles)
	}
}

func TestAuthServiceHandleConsentSkipInjectsSessionRolesIntoClaims(t *testing.T) {
	stub := &stubHydraAuthClient{
		consentRequest: HydraConsentRequest{
			Skip:           true,
			RequestedScope: []string{"openid"},
		},
		consentRedirect: "http://hydra/auth?consent_verifier=skipped",
	}
	svc := NewAuthService("http://localhost:8080", stub, &stubKratosAuthClient{
		session: KratosSession{
			Active:                      true,
			Roles:                       []string{"admin"},
			AuthenticatorAssuranceLevel: "aal2",
			Methods:                     []string{"password", "totp"},
		},
	})

	_, err := svc.HandleConsent(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), "challenge-1")
	if err != nil {
		t.Fatalf("HandleConsent() error = %v", err)
	}
	roles, ok := stub.receivedSession.AccessToken["roles"]
	if !ok {
		t.Fatal("expected roles in access_token claims for skip path")
	}
	roleList, ok := roles.([]string)
	if !ok || len(roleList) != 1 || roleList[0] != "admin" {
		t.Fatalf("expected [admin] in access_token claims, got %v", roles)
	}
}

type stubHydraAuthClient struct {
	loginRequest          HydraLoginRequest
	loginRedirect         string
	consentRequest        HydraConsentRequest
	consentRedirect       string
	rejectConsentRedirect string
	logoutRequest         HydraLogoutRequest
	logoutRedirect        string
	receivedSession       ConsentSessionClaims
}

func (s *stubHydraAuthClient) GetLoginRequest(_ context.Context, _ string) (HydraLoginRequest, error) {
	return s.loginRequest, nil
}

func (s *stubHydraAuthClient) AcceptLoginRequest(_ context.Context, _, _ string) (string, error) {
	if s.loginRedirect == "" {
		return "http://hydra/login", nil
	}
	return s.loginRedirect, nil
}

func (s *stubHydraAuthClient) GetConsentRequest(_ context.Context, _ string) (HydraConsentRequest, error) {
	return s.consentRequest, nil
}

func (s *stubHydraAuthClient) AcceptConsentRequest(_ context.Context, _ string, _, _ []string, session ConsentSessionClaims) (string, error) {
	s.receivedSession = session
	if s.consentRedirect == "" {
		return "http://hydra/consent", nil
	}
	return s.consentRedirect, nil
}

func (s *stubHydraAuthClient) RejectConsentRequest(_ context.Context, _ string, _, _ string) (string, error) {
	if s.rejectConsentRedirect == "" {
		return "http://hydra/rejected", nil
	}
	return s.rejectConsentRedirect, nil
}

func (s *stubHydraAuthClient) GetLogoutRequest(_ context.Context, _ string) (HydraLogoutRequest, error) {
	return s.logoutRequest, nil
}

func (s *stubHydraAuthClient) AcceptLogoutRequest(_ context.Context, _ string) (string, error) {
	if s.logoutRedirect == "" {
		return "http://hydra/logout", nil
	}
	return s.logoutRedirect, nil
}

type stubKratosAuthClient struct {
	session    KratosSession
	sessionErr error
}

func (s *stubKratosAuthClient) ToSession(_ context.Context, _ *http.Request) (KratosSession, error) {
	if s.sessionErr != nil {
		return KratosSession{}, s.sessionErr
	}
	return s.session, nil
}

func (s *stubKratosAuthClient) BrowserLoginURL(returnTo string) string {
	return "http://kratos/login?return_to=" + returnTo
}

func (s *stubKratosAuthClient) BrowserSettingsURL(returnTo string) string {
	return "http://kratos/settings?return_to=" + returnTo
}
