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
	admindomain "github.com/ryunosukekurokawa/idol-auth/internal/domain/admin"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
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

func TestAdminSetIdentityRolesReturnsUpdatedRoles(t *testing.T) {
	adminSvc := &stubAdminService{rolesResult: []string{"admin", "platform-operator"}}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/identities/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81/roles", bytes.NewBufferString(`{"roles":["Admin","platform-operator","admin"]}`))
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
	if adminSvc.lastActorID != "bootstrap-admin" {
		t.Fatalf("expected actor id to use bootstrap actor, got %q", adminSvc.lastActorID)
	}
	if got := strings.TrimSpace(w.Body.String()); !strings.Contains(got, `"roles":["admin","platform-operator"]`) {
		t.Fatalf("unexpected body: %s", got)
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

func TestAdminDisableUserReturnsUpdatedIdentity(t *testing.T) {
	adminSvc := &stubAdminService{
		disableResult: admindomain.Identity{ID: "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81", State: admindomain.IdentityStateInactive},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81/disable", nil)
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

func TestAdminEnableUserReturnsUpdatedIdentity(t *testing.T) {
	adminSvc := &stubAdminService{
		enableResult: admindomain.Identity{ID: "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81", State: admindomain.IdentityStateActive},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81/enable", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if adminSvc.lastIdentityID != "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81" {
		t.Fatalf("expected identity id to be forwarded, got %q", adminSvc.lastIdentityID)
	}
	if !strings.Contains(w.Body.String(), `"state":"active"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
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
	for _, key := range []string{"app", "client", "client_secret"} {
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

func TestAdminDisableUserByEmailResolvesIdentity(t *testing.T) {
	targetID := "f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81"
	adminSvc := &stubAdminService{
		searchResult: []admindomain.Identity{
			{ID: targetID, Email: "user@example.com", State: admindomain.IdentityStateActive},
		},
	}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/users/user%40example.com/disable", nil)
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

func TestAdminUserNotFoundForUnknownEmailRef(t *testing.T) {
	adminSvc := &stubAdminService{searchResult: nil}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/users/unknown%40example.com/disable", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestAdminSetIdentityRolesViaUsersPath(t *testing.T) {
	adminSvc := &stubAdminService{rolesResult: []string{"admin"}}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/users/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81/roles", bytes.NewBufferString(`{"roles":["admin"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if dep := w.Header().Get("Deprecation"); dep != "" {
		t.Fatalf("canonical /users/ path must not set Deprecation header, got %q", dep)
	}
}

func TestAdminSetIdentityRolesDeprecatedPathHasHeaders(t *testing.T) {
	adminSvc := &stubAdminService{rolesResult: []string{"admin"}}
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Admin: config.AdminConfig{BootstrapToken: "secret"},
	}, adminSvc, nil, &stubAuthService{})
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/identities/f17dc6e2-b3e1-4f1a-8a23-b0c73a1f9d81/roles", bytes.NewBufferString(`{"roles":["admin"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	if got := w.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("expected Deprecation: true header, got %q", got)
	}
	if got := w.Header().Get("Sunset"); got == "" {
		t.Fatal("expected Sunset header to be set")
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
	for _, key := range []string{"app", "client", "client_secret"} {
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
	lastIdentityID    string
	lastRoles         []string
	lastActorID       string
	rolesResult       []string
	lastSearchFilter  admindomain.SearchIdentitiesInput
	searchResult      []admindomain.Identity
	disableResult     admindomain.Identity
	enableResult      admindomain.Identity
	revokedIdentityID string
	lastAuditFilter   admindomain.ListAuditLogsInput
	auditLogs         []admindomain.AuditLog
	deletedIdentityID string
	createAppErr      error
	createClientErr   error
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
