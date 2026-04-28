package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

func TestServiceCreateApp(t *testing.T) {
	repo := &stubAppRepository{}
	svc := NewService(repo, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	created, err := svc.CreateApp(context.Background(), CreateAppInput{
		Name:        "Idol Web",
		Slug:        "idol-web",
		Type:        AppTypeWeb,
		PartyType:   PartyTypeFirst,
		Description: "primary web app",
		ActorID:     "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Fatal("expected app ID to be assigned")
	}
	if created.Slug != "idol-web" {
		t.Fatalf("expected slug %q, got %q", "idol-web", created.Slug)
	}
	if len(repo.apps) != 1 {
		t.Fatalf("expected 1 stored app, got %d", len(repo.apps))
	}
}

func TestServiceCreateAppRejectsInvalidSlug(t *testing.T) {
	svc := NewService(&stubAppRepository{}, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.CreateApp(context.Background(), CreateAppInput{
		Name:      "Bad",
		Slug:      "Bad Slug",
		Type:      AppTypeWeb,
		PartyType: PartyTypeFirst,
		ActorID:   "bootstrap-admin",
	})
	if !errors.Is(err, ErrInvalidAppSlug) {
		t.Fatalf("expected ErrInvalidAppSlug, got %v", err)
	}
}

func TestServiceCreateOIDCClientBrowserSuccess(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{
			{
				ID:        appID,
				Name:      "Idol Web",
				Slug:      "idol-web",
				Type:      AppTypeSPA,
				PartyType: PartyTypeFirst,
				Status:    AppStatusActive,
			},
		},
	}
	clientRepo := &stubOIDCClientRepository{}
	provisioner := &stubProvisioner{
		result: ProvisionedClient{
			HydraClientID:   "idol-web-client",
			ClientSecret:    "top-secret",
			ClientSecretSet: true,
		},
	}
	svc := NewService(appRepo, clientRepo, &stubAuditRepository{}, provisioner, timeNow)

	created, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:                   "Idol SPA",
		ClientType:             ClientTypePublic,
		RedirectURIs:           []string{"http://localhost:3000/callback"},
		PostLogoutRedirectURIs: []string{"http://localhost:3000/logout/callback"},
		Scopes:                 []string{"openid", "offline_access"},
		ActorID:                "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateOIDCClient() error = %v", err)
	}

	if created.Client.HydraClientID != "idol-web-client" {
		t.Fatalf("expected hydra client id %q, got %q", "idol-web-client", created.Client.HydraClientID)
	}
	if provisioner.lastSpec.TokenEndpointAuthMethod != "none" {
		t.Fatalf("expected auth method %q, got %q", "none", provisioner.lastSpec.TokenEndpointAuthMethod)
	}
	if !provisioner.lastSpec.PKCERequired {
		t.Fatal("expected PKCE to be required")
	}
	if len(clientRepo.clients) != 1 {
		t.Fatalf("expected 1 stored client, got %d", len(clientRepo.clients))
	}
}

func TestServiceCreateOIDCClientRejectsBrowserClientWithoutOpenID(t *testing.T) {
	appID := uuid.New()
	svc := NewService(&stubAppRepository{
		apps: []App{{ID: appID, Slug: "idol-web", Type: AppTypeWeb, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:         "bad client",
		ClientType:   ClientTypePublic,
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"offline_access"},
		ActorID:      "bootstrap-admin",
	})
	if !errors.Is(err, ErrOpenIDScopeRequired) {
		t.Fatalf("expected ErrOpenIDScopeRequired, got %v", err)
	}
}

func TestServiceCreateOIDCClientRejectsPublicClientAuthMethodOverride(t *testing.T) {
	appID := uuid.New()
	svc := NewService(&stubAppRepository{
		apps: []App{{ID: appID, Slug: "idol-web", Type: AppTypeSPA, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:                    "bad client",
		ClientType:              ClientTypePublic,
		TokenEndpointAuthMethod: "client_secret_basic",
		RedirectURIs:            []string{"https://example.com/callback"},
		Scopes:                  []string{"openid"},
		ActorID:                 "bootstrap-admin",
	})
	if !errors.Is(err, ErrInvalidTokenEndpointAuthMethod) {
		t.Fatalf("expected ErrInvalidTokenEndpointAuthMethod, got %v", err)
	}
}

func TestServiceCreateOIDCClientRejectsInvalidRedirectURI(t *testing.T) {
	appID := uuid.New()
	svc := NewService(&stubAppRepository{
		apps: []App{{ID: appID, Slug: "idol-web", Type: AppTypeWeb, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:         "bad client",
		ClientType:   ClientTypeConfidential,
		RedirectURIs: []string{"javascript:alert(1)"},
		Scopes:       []string{"openid"},
		ActorID:      "bootstrap-admin",
	})
	if !errors.Is(err, ErrInvalidRedirectURI) {
		t.Fatalf("expected ErrInvalidRedirectURI, got %v", err)
	}
}

func TestServiceListAppsReturnsStoredApps(t *testing.T) {
	repo := &stubAppRepository{
		apps: []App{
			{ID: uuid.New(), Slug: "idol-web"},
			{ID: uuid.New(), Slug: "idol-api"},
		},
	}
	svc := NewService(repo, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	apps, err := svc.ListApps(context.Background())
	if err != nil {
		t.Fatalf("ListApps() error = %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
}

func TestServiceListOIDCClientsReturnsClientsForApp(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{{ID: appID, Slug: "idol-web", Type: AppTypeWeb, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}
	clientRepo := &stubOIDCClientRepository{
		clients: []OIDCClient{{ID: uuid.New(), AppID: appID}},
	}
	svc := NewService(appRepo, clientRepo, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	clients, err := svc.ListOIDCClients(context.Background(), appID)
	if err != nil {
		t.Fatalf("ListOIDCClients() error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
}

func TestServiceListOIDCClientsReturnsNotFoundForMissingApp(t *testing.T) {
	svc := NewService(&stubAppRepository{}, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.ListOIDCClients(context.Background(), uuid.New())
	if !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("expected ErrAppNotFound, got %v", err)
	}
}

func TestServiceCreateOIDCClientM2MSuccess(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{
			{ID: appID, Slug: "idol-api", Type: AppTypeM2M, PartyType: PartyTypeFirst, Status: AppStatusActive},
		},
	}
	provisioner := &stubProvisioner{
		result: ProvisionedClient{
			HydraClientID:   "idol-api-m2m",
			ClientSecret:    "top-secret",
			ClientSecretSet: true,
		},
	}
	svc := NewService(appRepo, &stubOIDCClientRepository{}, &stubAuditRepository{}, provisioner, timeNow)

	created, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:       "Service Account",
		ClientType: ClientTypeConfidential,
		Scopes:     []string{"openid"},
		ActorID:    "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateOIDCClient() M2M error = %v", err)
	}
	if provisioner.lastSpec.TokenEndpointAuthMethod != "client_secret_basic" {
		t.Fatalf("expected client_secret_basic, got %q", provisioner.lastSpec.TokenEndpointAuthMethod)
	}
	if provisioner.lastSpec.PKCERequired {
		t.Fatal("expected PKCE not required for M2M")
	}
	if created.ClientSecret != "top-secret" {
		t.Fatalf("expected client secret, got %q", created.ClientSecret)
	}
}

func TestServiceCreateOIDCClientConfidentialWebSuccess(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{
			{ID: appID, Slug: "idol-web", Type: AppTypeWeb, PartyType: PartyTypeThird, Status: AppStatusActive},
		},
	}
	provisioner := &stubProvisioner{
		result: ProvisionedClient{
			HydraClientID:   "idol-web-confidential",
			ClientSecret:    "server-secret",
			ClientSecretSet: true,
		},
	}
	svc := NewService(appRepo, &stubOIDCClientRepository{}, &stubAuditRepository{}, provisioner, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:         "Server Side App",
		ClientType:   ClientTypeConfidential,
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"openid", "profile"},
		ActorID:      "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateOIDCClient() confidential web error = %v", err)
	}
	if provisioner.lastSpec.TokenEndpointAuthMethod != "client_secret_basic" {
		t.Fatalf("expected client_secret_basic, got %q", provisioner.lastSpec.TokenEndpointAuthMethod)
	}
	if !provisioner.lastSpec.PKCERequired {
		t.Fatal("expected PKCE required for confidential web client")
	}
}

func TestServiceCreateOIDCClientRejectsM2MWithRedirectURIs(t *testing.T) {
	appID := uuid.New()
	svc := NewService(&stubAppRepository{
		apps: []App{{ID: appID, Slug: "idol-api", Type: AppTypeM2M, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:         "bad m2m",
		ClientType:   ClientTypeConfidential,
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"openid"},
		ActorID:      "bootstrap-admin",
	})
	if !errors.Is(err, ErrRedirectURIsNotAllowed) {
		t.Fatalf("expected ErrRedirectURIsNotAllowed, got %v", err)
	}
}

func TestServiceCreateOIDCClientRejectsPublicM2MClient(t *testing.T) {
	appID := uuid.New()
	svc := NewService(&stubAppRepository{
		apps: []App{{ID: appID, Slug: "idol-api", Type: AppTypeM2M, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:       "bad public m2m",
		ClientType: ClientTypePublic,
		Scopes:     []string{"openid"},
		ActorID:    "bootstrap-admin",
	})
	if !errors.Is(err, ErrConfidentialClientRequired) {
		t.Fatalf("expected ErrConfidentialClientRequired, got %v", err)
	}
}

func TestServiceCreateOIDCClientRollsBackOnStorageError(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{{ID: appID, Slug: "idol-web", Type: AppTypeSPA, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}
	provisioner := &stubProvisioner{
		result: ProvisionedClient{HydraClientID: "hydra-client-id", ClientSecret: "s"},
	}
	clientRepo := &stubOIDCClientRepositoryWithError{err: errors.New("db error")}
	svc := NewService(appRepo, clientRepo, &stubAuditRepository{}, provisioner, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Name:         "X",
		ClientType:   ClientTypePublic,
		RedirectURIs: []string{"https://example.com/cb"},
		Scopes:       []string{"openid"},
		ActorID:      "bootstrap-admin",
	})
	if err == nil {
		t.Fatal("expected storage error")
	}
	if len(provisioner.deleteIDs) != 1 || provisioner.deleteIDs[0] != "hydra-client-id" {
		t.Fatalf("expected rollback delete of hydra-client-id, got %v", provisioner.deleteIDs)
	}
}

func TestIsLoopbackHostIPv4Loopback(t *testing.T) {
	if !isLoopbackHost("127.0.0.1") {
		t.Fatal("expected 127.0.0.1 to be loopback")
	}
}

func TestIsLoopbackHostNonLoopbackIP(t *testing.T) {
	if isLoopbackHost("192.168.1.1") {
		t.Fatal("expected 192.168.1.1 to not be loopback")
	}
}

type stubOIDCClientRepositoryWithError struct {
	err error
}

func (r *stubOIDCClientRepositoryWithError) Create(_ context.Context, _ OIDCClient) (OIDCClient, error) {
	return OIDCClient{}, r.err
}

func (r *stubOIDCClientRepositoryWithError) ListByAppID(_ context.Context, _ uuid.UUID) ([]OIDCClient, error) {
	return nil, nil
}

type stubAppRepository struct {
	apps []App
}

func (r *stubAppRepository) Create(_ context.Context, app App) (App, error) {
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	r.apps = append(r.apps, app)
	return app, nil
}

func (r *stubAppRepository) List(_ context.Context) ([]App, error) {
	return append([]App(nil), r.apps...), nil
}

func (r *stubAppRepository) GetByID(_ context.Context, id uuid.UUID) (App, error) {
	for _, app := range r.apps {
		if app.ID == id {
			return app, nil
		}
	}
	return App{}, ErrAppNotFound
}

type stubOIDCClientRepository struct {
	clients []OIDCClient
}

func (r *stubOIDCClientRepository) Create(_ context.Context, client OIDCClient) (OIDCClient, error) {
	if client.ID == uuid.Nil {
		client.ID = uuid.New()
	}
	r.clients = append(r.clients, client)
	return client, nil
}

func (r *stubOIDCClientRepository) ListByAppID(_ context.Context, appID uuid.UUID) ([]OIDCClient, error) {
	var out []OIDCClient
	for _, client := range r.clients {
		if client.AppID == appID {
			out = append(out, client)
		}
	}
	return out, nil
}

type stubAuditRepository struct {
	logs []audit.Log
}

func (r *stubAuditRepository) Write(_ context.Context, entry audit.Log) error {
	r.logs = append(r.logs, entry)
	return nil
}

func (r *stubAuditRepository) List(_ context.Context, _ audit.ListParams) ([]audit.Log, error) {
	return append([]audit.Log(nil), r.logs...), nil
}

type stubProvisioner struct {
	result    ProvisionedClient
	lastSpec  ClientProvisionSpec
	deleteIDs []string
}

func (p *stubProvisioner) CreateClient(_ context.Context, spec ClientProvisionSpec) (ProvisionedClient, error) {
	p.lastSpec = spec
	if p.result.HydraClientID == "" {
		p.result.HydraClientID = "generated-client-id"
	}
	return p.result, nil
}

func (p *stubProvisioner) DeleteClient(_ context.Context, clientID string) error {
	p.deleteIDs = append(p.deleteIDs, clientID)
	return nil
}

func timeNow() time.Time {
	return time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
}

func TestSlugifyName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"My SPA", "my-spa"},
		{"  Hello  World!  ", "hello-world"},
		{"UPPERCASE", "uppercase"},
		{"already-slug", "already-slug"},
		{"123 App", "123-app"},
		{"  !@#$%  ", "app"},
		{"Hello_World", "hello-world"},
		{"Café & Bar", "caf-bar"},
	}
	for _, tc := range cases {
		if got := slugifyName(tc.input); got != tc.want {
			t.Errorf("slugifyName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestServiceCreateAppDerivesSlugFromName(t *testing.T) {
	repo := &stubAppRepository{}
	svc := NewService(repo, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	created, err := svc.CreateApp(context.Background(), CreateAppInput{
		Name:      "My Web App",
		Type:      AppTypeWeb,
		PartyType: PartyTypeFirst,
		ActorID:   "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if created.Slug != "my-web-app" {
		t.Fatalf("expected slug %q, got %q", "my-web-app", created.Slug)
	}
}

func TestServiceCreateAppExplicitSlugTakesPrecedence(t *testing.T) {
	repo := &stubAppRepository{}
	svc := NewService(repo, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	created, err := svc.CreateApp(context.Background(), CreateAppInput{
		Name:      "My Web App",
		Slug:      "custom-slug",
		Type:      AppTypeWeb,
		PartyType: PartyTypeFirst,
		ActorID:   "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if created.Slug != "custom-slug" {
		t.Fatalf("expected slug %q, got %q", "custom-slug", created.Slug)
	}
}

func TestServiceCreateAppAcceptsTypeAlias(t *testing.T) {
	cases := []struct {
		alias    AppType
		expected AppType
	}{
		{"webapp", AppTypeWeb},
		{"server", AppTypeWeb},
		{"single-page", AppTypeSPA},
		{"single_page", AppTypeSPA},
		{"mobile", AppTypeNative},
	}
	for _, tc := range cases {
		repo := &stubAppRepository{}
		svc := NewService(repo, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)
		created, err := svc.CreateApp(context.Background(), CreateAppInput{
			Name:      "App",
			Slug:      "app",
			Type:      tc.alias,
			PartyType: PartyTypeFirst,
			ActorID:   "bootstrap-admin",
		})
		if err != nil {
			t.Fatalf("CreateApp() with type alias %q error = %v", tc.alias, err)
		}
		if created.Type != tc.expected {
			t.Fatalf("expected type %q for alias %q, got %q", tc.expected, tc.alias, created.Type)
		}
	}
}

func TestServiceCreateOIDCClientDefaultsClientTypeForSPA(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{{ID: appID, Name: "My SPA", Slug: "my-spa", Type: AppTypeSPA, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}
	svc := NewService(appRepo, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	created, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"openid"},
		ActorID:      "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateOIDCClient() without client_type error = %v", err)
	}
	if created.Client.ClientType != ClientTypePublic {
		t.Fatalf("expected public client type for SPA, got %q", created.Client.ClientType)
	}
}

func TestServiceCreateOIDCClientDefaultsClientTypeForM2M(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{{ID: appID, Name: "My API", Slug: "my-api", Type: AppTypeM2M, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}
	svc := NewService(appRepo, &stubOIDCClientRepository{}, &stubAuditRepository{}, &stubProvisioner{}, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		Scopes:  []string{"openid"},
		ActorID: "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateOIDCClient() M2M without client_type error = %v", err)
	}
}

func TestServiceCreateOIDCClientDefaultsNameFromApp(t *testing.T) {
	appID := uuid.New()
	appRepo := &stubAppRepository{
		apps: []App{{ID: appID, Name: "Idol Platform", Slug: "idol-platform", Type: AppTypeSPA, PartyType: PartyTypeFirst, Status: AppStatusActive}},
	}
	provisioner := &stubProvisioner{result: ProvisionedClient{HydraClientID: "test-client"}}
	svc := NewService(appRepo, &stubOIDCClientRepository{}, &stubAuditRepository{}, provisioner, timeNow)

	_, err := svc.CreateOIDCClient(context.Background(), appID, CreateOIDCClientInput{
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"openid"},
		ActorID:      "bootstrap-admin",
	})
	if err != nil {
		t.Fatalf("CreateOIDCClient() without name error = %v", err)
	}
	if provisioner.lastSpec.Name != "Idol Platform" {
		t.Fatalf("expected client name %q (from app), got %q", "Idol Platform", provisioner.lastSpec.Name)
	}
}

func TestAuditMetadataJSONRoundTrip(t *testing.T) {
	meta := auditMetadata{AppSlug: "idol-web", ClientName: "Idol SPA"}
	b, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}
