package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/config"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/account"
	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
	profiledomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/profile"
	apphttp "github.com/ryunosukekurokawa/idol-auth/internal/http"
)

func TestAdminCreateAppRequiresAuthorization(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAdminCreateAppReturnsCreated(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{
		"name":"Idol Web",
		"slug":"idol-web",
		"type":"web",
		"party_type":"first_party",
		"description":"main app"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"management_token":"mgmt-secret"`) {
		t.Fatalf("expected management token in response, got %s", w.Body.String())
	}
}

func TestAdminListAppsAllowsKratosAdminSession(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated:               true,
			IdentityID:                  "identity-admin",
			Email:                       "admin@example.com",
			AuthenticatorAssuranceLevel: "aal2",
		},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{AllowedEmails: []string{"admin@example.com"}},
	}, &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/apps", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAdminListAppsAllowsKratosAdminRole(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated:               true,
			IdentityID:                  "identity-admin",
			Email:                       "user@example.com",
			Roles:                       []string{"admin"},
			AuthenticatorAssuranceLevel: "aal2",
		},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{AllowedRoles: []string{"admin"}},
	}, &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/apps", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAdminCreateAppRejectsNonAdminSession(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated:               true,
			IdentityID:                  "identity-user",
			Email:                       "user@example.com",
			AuthenticatorAssuranceLevel: "aal2",
		},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{AllowedEmails: []string{"admin@example.com"}},
	}, &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAdminSearchUsersReturnsItems(t *testing.T) {
	adminSvc := &stubAdminService{
		searchResult: []admindomain.Identity{
			{ID: "identity-123", Email: "user@example.com", State: admindomain.IdentityStateActive},
		},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users?identifier=user@example.com&state=active", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if adminSvc.lastSearchFilter.CredentialsIdentifier != "user@example.com" || adminSvc.lastSearchFilter.State != admindomain.IdentityStateActive {
		t.Fatalf("unexpected search filter: %+v", adminSvc.lastSearchFilter)
	}
	if !strings.Contains(w.Body.String(), `"identity-123"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAdminSearchUsersRejectsInvalidState(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users?state=paused", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminDeleteUserReturnsNoContent(t *testing.T) {
	adminSvc := &stubAdminService{}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusNoContent, w.Code, w.Body.String())
	}
	if adminSvc.deletedIdentityID != "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81" {
		t.Fatalf("expected deleted identity id to be forwarded, got %q", adminSvc.deletedIdentityID)
	}
}

func TestAdminRevokeIdentitySessionsReturnsNoContent(t *testing.T) {
	adminSvc := &stubAdminService{}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81/revoke-sessions", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusNoContent, w.Code, w.Body.String())
	}
	if adminSvc.revokedIdentityID != "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81" {
		t.Fatalf("expected revoked identity id to be forwarded, got %q", adminSvc.revokedIdentityID)
	}
}

func TestAdminListAuditLogsReturnsItems(t *testing.T) {
	metadata, _ := json.Marshal(map[string]any{"roles": []string{"admin"}})
	adminSvc := &stubAdminService{
		auditLogs: []admindomain.AuditLog{{
			EventType: "identity.roles.updated",
			ActorID:   "bootstrap-admin",
			Metadata:  metadata,
		}},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-logs?actor_id=bootstrap-admin&limit=10", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if adminSvc.lastAuditFilter.ActorID != "bootstrap-admin" || adminSvc.lastAuditFilter.Limit != 10 {
		t.Fatalf("unexpected audit filter: %+v", adminSvc.lastAuditFilter)
	}
	if !strings.Contains(w.Body.String(), `"identity.roles.updated"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAdminListAuditLogsRejectsInvalidLimit(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-logs?limit=abc", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminListAppsReturnsOK(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/apps", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"items"`) {
		t.Fatalf("expected items key in response, got %s", w.Body.String())
	}
}

func TestAdminIssueManagementTokenReturnsToken(t *testing.T) {
	appID := uuid.New()
	adminSvc := &stubAdminService{}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps/"+appID.String()+"/management-token", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if adminSvc.issuedManagementTokenAppID != appID {
		t.Fatalf("expected app id %s, got %s", appID, adminSvc.issuedManagementTokenAppID)
	}
	if !strings.Contains(w.Body.String(), `"management_token":"mgmt-secret"`) {
		t.Fatalf("expected management token body, got %s", w.Body.String())
	}
}

func TestAccountOverviewReturnsMemberships(t *testing.T) {
	appID := uuid.New()
	accountSvc := &stubAccountService{
		membershipsForIdentity: []account.AppMembership{{
			ID:         uuid.New(),
			AppID:      appID,
			AppSlug:    "idol-web",
			AppName:    "Idol Web",
			IdentityID: "identity-123",
			Status:     account.MembershipStatusActive,
		}},
	}
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
			Email:         "user@example.com",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn, accountSvc)
	req := httptest.NewRequest(http.MethodGet, "/v1/account", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"memberships"`) || !strings.Contains(w.Body.String(), `"idol-web"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAccountDeletionScheduleReturnsAccepted(t *testing.T) {
	accountSvc := &stubAccountService{
		scheduledDeletion: account.DeletionRequest{
			ID:         uuid.New(),
			IdentityID: "identity-123",
			Status:     account.DeletionStatusScheduled,
		},
	}
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated: true,
			IdentityID:    "identity-123",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn, accountSvc)
	req := httptest.NewRequest(http.MethodPost, "/v1/account/deletion", bytes.NewBufferString(`{"reason":"user_requested"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusAccepted, w.Code, w.Body.String())
	}
	if accountSvc.lastDeletionReason != "user_requested" {
		t.Fatalf("expected deletion reason to be forwarded, got %q", accountSvc.lastDeletionReason)
	}
}

func TestAppScopedListUsersReturnsMemberships(t *testing.T) {
	appID := uuid.New()
	accountSvc := &stubAccountService{
		resolvedApp: app.App{
			ID:   appID,
			Slug: "idol-web",
			Name: "Idol Web",
		},
		membershipsForApp: []account.AppMembership{{
			ID:         uuid.New(),
			AppID:      appID,
			AppSlug:    "idol-web",
			IdentityID: "identity-123",
			Status:     account.MembershipStatusActive,
		}},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, accountSvc)
	req := httptest.NewRequest(http.MethodGet, "/v1/apps/self/users", nil)
	req.Header.Set("Authorization", "Bearer app-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"idol-web"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAppScopedDeleteUserRevokesMembership(t *testing.T) {
	appID := uuid.New()
	accountSvc := &stubAccountService{
		resolvedApp: app.App{
			ID:   appID,
			Slug: "idol-web",
			Name: "Idol Web",
		},
	}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, accountSvc)
	req := httptest.NewRequest(http.MethodDelete, "/v1/apps/self/users/identity-123", nil)
	req.Header.Set("Authorization", "Bearer app-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusNoContent, w.Code, w.Body.String())
	}
	if accountSvc.revokedIdentityID != "identity-123" {
		t.Fatalf("expected revoked identity id to be forwarded, got %q", accountSvc.revokedIdentityID)
	}
	if accountSvc.revokedAppID != appID {
		t.Fatalf("expected revoked app id %s, got %s", appID, accountSvc.revokedAppID)
	}
}

func TestAdminCreateOIDCClientReturnsCreated(t *testing.T) {
	appID := uuid.New()
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/apps/"+appID.String()+"/clients",
		bytes.NewBufferString(`{"name":"My Client","client_type":"public","redirect_uris":["https://example.com/callback"],"scopes":["openid"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestAdminCreateOIDCClientRejectsBadAppID(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps/not-a-uuid/clients", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminListOIDCClientsReturnsOK(t *testing.T) {
	appID := uuid.New()
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/apps/"+appID.String()+"/clients", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"items"`) {
		t.Fatalf("expected items key in response, got %s", w.Body.String())
	}
}

func TestAdminListOIDCClientsRejectsBadAppID(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/apps/not-a-uuid/clients", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminCreateOIDCClientReturnsNotFoundForMissingApp(t *testing.T) {
	appID := uuid.New()
	adminSvc := &stubAdminService{createClientErr: app.ErrAppNotFound}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/apps/"+appID.String()+"/clients",
		bytes.NewBufferString(`{"name":"X","client_type":"public","redirect_uris":["https://example.com/cb"],"scopes":["openid"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestAdminCreateOIDCClientReturnsBadRequestForDomainValidationError(t *testing.T) {
	appID := uuid.New()
	adminSvc := &stubAdminService{createClientErr: app.ErrInvalidRedirectURI}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/apps/"+appID.String()+"/clients",
		bytes.NewBufferString(`{"name":"X","client_type":"public","redirect_uris":["javascript:alert(1)"],"scopes":["openid"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminCreateAppReturnsBadRequestForDomainValidationError(t *testing.T) {
	adminSvc := &stubAdminService{createAppErr: app.ErrInvalidAppSlug}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{"name":"X","slug":"Bad Slug","type":"web","party_type":"first_party"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminCreateAppReturnsConflictForDisabledApp(t *testing.T) {
	adminSvc := &stubAdminService{createAppErr: app.ErrAppDisabled}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{"name":"X","slug":"x","type":"web","party_type":"first_party"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestHandleLoginReturnsBadRequestOnChallengeRequired(t *testing.T) {
	authn := &stubAuthService{loginErr: apphttp.ErrChallengeRequired}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/login", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleReadyzReturnsServiceUnavailableWhenNotReady(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, nil, &stubReadinessChecker{err: errors.New("db not ready")}, nil)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestAdminReadAccessRejectsSessionWithoutMFA(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated:               true,
			IdentityID:                  "identity-admin",
			Email:                       "admin@example.com",
			AuthenticatorAssuranceLevel: "aal1",
		},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{AllowedEmails: []string{"admin@example.com"}},
	}, &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/apps", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestAdminMutatingAccessRequiresBootstrapTokenForSessionAuth(t *testing.T) {
	authn := &stubAuthService{
		session: apphttp.SessionView{
			Authenticated:               true,
			IdentityID:                  "identity-admin",
			Email:                       "admin@example.com",
			AuthenticatorAssuranceLevel: "aal2",
		},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{AllowedEmails: []string{"admin@example.com"}},
	}, &stubAdminService{}, nil, authn)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{
		"name":"Idol Web",
		"slug":"idol-web",
		"type":"web",
		"party_type":"first_party",
		"description":"main app"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestAdminCreateAppWithClientReturnsCreatedWithSecret(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{
		"name":"Idol Web","slug":"idol-web","type":"web","party_type":"first_party",
		"client":{
			"name":"Idol Web Client","client_type":"confidential",
			"redirect_uris":["https://example.com/callback"],"scopes":["openid","email"]
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	for _, key := range []string{"app", "client", "client_secret", "management_token"} {
		if _, ok := resp[key]; !ok {
			t.Fatalf("expected %q in response, got %s", key, w.Body.String())
		}
	}
}

func TestAdminCreateAppWithClientPropagatesClientError(t *testing.T) {
	adminSvc := &stubAdminService{createClientErr: app.ErrInvalidRedirectURI}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{
		"name":"Idol Web","slug":"idol-web","type":"web","party_type":"first_party",
		"client":{"name":"Bad","client_type":"confidential","redirect_uris":["javascript:evil"],"scopes":["openid"]}
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminPatchUserStateInactiveDisablesIdentity(t *testing.T) {
	adminSvc := &stubAdminService{
		disableResult: admindomain.Identity{ID: "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81", State: admindomain.IdentityStateInactive},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81",
		bytes.NewBufferString(`{"state":"inactive"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if adminSvc.lastIdentityID != "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81" {
		t.Fatalf("expected identity id to be forwarded, got %q", adminSvc.lastIdentityID)
	}
	if !strings.Contains(w.Body.String(), `"state":"inactive"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAdminPatchUserStateActiveEnablesIdentity(t *testing.T) {
	adminSvc := &stubAdminService{
		enableResult: admindomain.Identity{ID: "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81", State: admindomain.IdentityStateActive},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81",
		bytes.NewBufferString(`{"state":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"state":"active"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAdminPatchUserStateRejectsInvalidState(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81",
		bytes.NewBufferString(`{"state":"disabled"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminPatchUserByEmailResolvesIdentity(t *testing.T) {
	targetID := "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81"
	adminSvc := &stubAdminService{
		searchResult: []admindomain.Identity{
			{ID: targetID, Email: "user@example.com", State: admindomain.IdentityStateActive},
		},
		enableResult: admindomain.Identity{ID: targetID, State: admindomain.IdentityStateActive},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/user%40example.com",
		bytes.NewBufferString(`{"state":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if adminSvc.lastIdentityID != targetID {
		t.Fatalf("expected resolved id %q, got %q", targetID, adminSvc.lastIdentityID)
	}
}

func TestAdminPatchUserUnknownEmailRefReturnsNotFound(t *testing.T) {
	adminSvc := &stubAdminService{searchResult: nil}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/unknown%40example.com",
		bytes.NewBufferString(`{"state":"inactive"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestAdminPatchUserWithRolesOnly(t *testing.T) {
	adminSvc := &stubAdminService{rolesResult: []string{"admin"}}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81",
		bytes.NewBufferString(`{"roles":["admin"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"roles":["admin"]`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestAdminPatchUserWithStateAndRoles(t *testing.T) {
	adminSvc := &stubAdminService{
		enableResult: admindomain.Identity{ID: "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81", State: admindomain.IdentityStateActive},
		rolesResult:  []string{"admin"},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81",
		bytes.NewBufferString(`{"state":"active","roles":["admin"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"state":"active"`) {
		t.Fatalf("expected state in body, got %s", body)
	}
}

func TestAdminPatchUserWithNeitherStateNorRolesReturnsBadRequest(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAdminCreateAppWithTopLevelRedirectURIsReturnsAppAndClient(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{
		"name":"My SPA","type":"spa","party_type":"first_party",
		"redirect_uris":["https://example.com/callback"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	for _, key := range []string{"app", "client", "client_secret", "management_token"} {
		if _, ok := resp[key]; !ok {
			t.Fatalf("expected %q in response, got %s", key, w.Body.String())
		}
	}
}

func TestAdminCreateAppSetsLocationHeader(t *testing.T) {
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, &stubAdminService{}, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/apps", bytes.NewBufferString(`{
		"name":"Idol Web","slug":"idol-web","type":"web","party_type":"first_party"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); !strings.HasPrefix(loc, "/v1/admin/apps/") {
		t.Fatalf("expected Location header starting with /v1/admin/apps/, got %q", loc)
	}
}

type stubReadinessChecker struct {
	err error
}

func (s *stubReadinessChecker) Ready(_ context.Context) error {
	return s.err
}

type stubAdminService struct {
	lastIdentityID             string
	lastRoles                  []string
	lastActorID                string
	rolesResult                []string
	lastSearchFilter           admindomain.SearchIdentitiesInput
	searchResult               []admindomain.Identity
	disableResult              admindomain.Identity
	enableResult               admindomain.Identity
	revokedIdentityID          string
	lastAuditFilter            admindomain.ListAuditLogsInput
	auditLogs                  []admindomain.AuditLog
	deletedIdentityID          string
	createAppErr               error
	createClientErr            error
	issuedManagementTokenAppID uuid.UUID
}

func (s *stubAdminService) CreateApp(_ context.Context, input app.CreateAppInput) (app.App, error) {
	if s.createAppErr != nil {
		return app.App{}, s.createAppErr
	}
	return app.App{
		ID:          uuid.New(),
		Name:        input.Name,
		Slug:        input.Slug,
		Type:        input.Type,
		PartyType:   input.PartyType,
		Status:      app.AppStatusActive,
		Description: input.Description,
	}, nil
}

func (s *stubAdminService) ListApps(_ context.Context) ([]app.App, error) {
	return nil, nil
}

func (s *stubAdminService) IssueManagementToken(_ context.Context, appID uuid.UUID, actorID string) (string, error) {
	s.issuedManagementTokenAppID = appID
	s.lastActorID = actorID
	return "mgmt-secret", nil
}

func (s *stubAdminService) CreateOIDCClient(_ context.Context, _ uuid.UUID, _ app.CreateOIDCClientInput) (app.ClientRegistration, error) {
	if s.createClientErr != nil {
		return app.ClientRegistration{}, s.createClientErr
	}
	return app.ClientRegistration{}, nil
}

func (s *stubAdminService) ListOIDCClients(_ context.Context, _ uuid.UUID) ([]app.OIDCClient, error) {
	return nil, nil
}

func (s *stubAdminService) SetIdentityRoles(_ context.Context, input admindomain.SetIdentityRolesInput) ([]string, error) {
	s.lastIdentityID = input.IdentityID
	s.lastRoles = append([]string(nil), input.Roles...)
	s.lastActorID = input.ActorID
	if s.rolesResult != nil {
		return append([]string(nil), s.rolesResult...), nil
	}
	return append([]string(nil), input.Roles...), nil
}

func (s *stubAdminService) SearchIdentities(_ context.Context, input admindomain.SearchIdentitiesInput) ([]admindomain.Identity, error) {
	s.lastSearchFilter = input
	return append([]admindomain.Identity(nil), s.searchResult...), nil
}

func (s *stubAdminService) DisableIdentity(_ context.Context, input admindomain.DisableIdentityInput) (admindomain.Identity, error) {
	s.lastIdentityID = input.IdentityID
	s.lastActorID = input.ActorID
	if s.disableResult.ID != "" {
		return s.disableResult, nil
	}
	return admindomain.Identity{ID: input.IdentityID, State: admindomain.IdentityStateInactive}, nil
}

func (s *stubAdminService) EnableIdentity(_ context.Context, input admindomain.EnableIdentityInput) (admindomain.Identity, error) {
	s.lastIdentityID = input.IdentityID
	s.lastActorID = input.ActorID
	if s.enableResult.ID != "" {
		return s.enableResult, nil
	}
	return admindomain.Identity{ID: input.IdentityID, State: admindomain.IdentityStateActive}, nil
}

func (s *stubAdminService) RevokeIdentitySessions(_ context.Context, input admindomain.RevokeIdentitySessionsInput) error {
	s.revokedIdentityID = input.IdentityID
	s.lastActorID = input.ActorID
	return nil
}

func (s *stubAdminService) DeleteIdentity(_ context.Context, input admindomain.DeleteIdentityInput) error {
	s.deletedIdentityID = input.IdentityID
	s.lastActorID = input.ActorID
	return nil
}

func (s *stubAdminService) ListAuditLogs(_ context.Context, input admindomain.ListAuditLogsInput) ([]admindomain.AuditLog, error) {
	s.lastAuditFilter = input
	return append([]audit.Log(nil), s.auditLogs...), nil
}

type stubAccountService struct {
	resolvedApp            app.App
	membershipsForIdentity []account.AppMembership
	membershipsForApp      []account.AppMembership
	scheduledDeletion      account.DeletionRequest
	deletionRequest        *account.DeletionRequest
	lastDeletionReason     string
	revokedIdentityID      string
	revokedAppID           uuid.UUID
	disconnectedIdentityID string
	disconnectedAppID      uuid.UUID
}

func (s *stubAccountService) ListMembershipsForIdentity(_ context.Context, _ string) ([]account.AppMembership, error) {
	return append([]account.AppMembership(nil), s.membershipsForIdentity...), nil
}

func (s *stubAccountService) ListMembershipsForApp(_ context.Context, _ uuid.UUID) ([]account.AppMembership, error) {
	return append([]account.AppMembership(nil), s.membershipsForApp...), nil
}

func (s *stubAccountService) DisconnectIdentityFromApp(_ context.Context, identityID string, appID uuid.UUID, _ string) error {
	s.disconnectedIdentityID = identityID
	s.disconnectedAppID = appID
	return nil
}

func (s *stubAccountService) RevokeAppUser(_ context.Context, appID uuid.UUID, identityID, _ string) error {
	s.revokedAppID = appID
	s.revokedIdentityID = identityID
	return nil
}

func (s *stubAccountService) ScheduleDeletion(_ context.Context, _ string, _ string, reason string) (account.DeletionRequest, error) {
	s.lastDeletionReason = reason
	return s.scheduledDeletion, nil
}

func (s *stubAccountService) CancelDeletion(_ context.Context, _ string, _ string) error {
	return nil
}

func (s *stubAccountService) GetDeletionRequest(_ context.Context, _ string) (*account.DeletionRequest, error) {
	return s.deletionRequest, nil
}

func (s *stubAccountService) ResolveAppByToken(_ context.Context, _ string) (app.App, error) {
	if s.resolvedApp.ID == uuid.Nil {
		return app.App{}, app.ErrAppNotFound
	}
	return s.resolvedApp, nil
}

func (s *stubAccountService) RegisterIdentityForApp(_ context.Context, _ app.App, _ account.RegisterIdentityInput, _ string) (account.RegisterForAppResult, error) {
	return account.RegisterForAppResult{IdentityID: "new-identity-id", CreatedSharedAccount: true, RecoveryLink: "https://auth.example.com/recovery?token=test"}, nil
}

func (s *stubAccountService) GetMembershipForApp(_ context.Context, _ uuid.UUID, _ string) (account.AppMembership, error) {
	return account.AppMembership{}, nil
}

func TestRegisterAppUser_RequiresAppToken(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/apps/self/users", bytes.NewBufferString(`{"email":"u@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d; body=%s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestRegisterAppUser_RejectsMissingEmail(t *testing.T) {
	appID := uuid.New()
	accountSvc := &stubAccountService{resolvedApp: app.App{ID: appID, Slug: "idol-web"}}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, accountSvc)
	req := httptest.NewRequest(http.MethodPost, "/v1/apps/self/users", bytes.NewBufferString(`{"display_name":"Test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer app-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d; body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestRegisterAppUser_ReturnsCreatedWithRecoveryLink(t *testing.T) {
	appID := uuid.New()
	accountSvc := &stubAccountService{resolvedApp: app.App{ID: appID, Slug: "idol-web"}}
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, accountSvc)
	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "display_name": "Test User"})
	req := httptest.NewRequest(http.MethodPost, "/v1/apps/self/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer app-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d; body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"identity_id"`) {
		t.Fatalf("expected identity_id in response, got %s", w.Body.String())
	}
}

func TestGetAppUserProfile_RequiresAppToken(t *testing.T) {
	router := apphttp.NewRouter(testConfig(), &stubAdminService{}, nil, &stubAuthService{}, &stubAccountService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/apps/self/users/identity-123/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetAppUserProfile_ReturnsPublicProfileForMember(t *testing.T) {
	appID := uuid.New()
	accountSvc := &stubAccountService{resolvedApp: app.App{ID: appID, Slug: "idol-web"}}
	profileSvc := &stubProfileService{
		profile: profiledomain.Profile{
			IdentityID:  "identity-123",
			DisplayName: "推し活太郎",
			OshiColor:   "#ffb2d8",
			Email:       "user@example.com",
		},
	}
	cfg := testConfig()
	cfg.ProfileSvc = profileSvc
	router := apphttp.NewRouter(cfg, &stubAdminService{}, nil, &stubAuthService{}, accountSvc)
	req := httptest.NewRequest(http.MethodGet, "/v1/apps/self/users/identity-123/profile", nil)
	req.Header.Set("Authorization", "Bearer app-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"user@example.com"`) {
		t.Fatal("email (PII) must not appear in public profile response")
	}
	if !strings.Contains(w.Body.String(), "推し活太郎") {
		t.Fatalf("expected display_name in response, got %s", w.Body.String())
	}
}
