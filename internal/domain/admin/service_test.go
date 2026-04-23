package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

// ---------------------------------------------------------------------------
// SetIdentityRoles
// ---------------------------------------------------------------------------

func TestServiceSetIdentityRolesNormalizesAndDeduplicates(t *testing.T) {
	mgr := &stubIdentityRoleManager{}
	svc := NewService(&stubAppManager{}, mgr, &stubAuditRepository{}, testNow)

	roles, err := svc.SetIdentityRoles(context.Background(), SetIdentityRolesInput{
		IdentityID: "id-1",
		Roles:      []string{" Admin ", "platform-operator", "admin", ""},
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("SetIdentityRoles() error = %v", err)
	}
	want := []string{"admin", "platform-operator"}
	if len(roles) != len(want) || roles[0] != want[0] || roles[1] != want[1] {
		t.Fatalf("expected %v, got %v", want, roles)
	}
	if mgr.receivedID != "id-1" {
		t.Fatalf("expected identity id %q, got %q", "id-1", mgr.receivedID)
	}
}

func TestServiceSetIdentityRolesWritesAuditLog(t *testing.T) {
	auditRepo := &stubAuditRepository{}
	svc := NewService(&stubAppManager{}, &stubIdentityRoleManager{}, auditRepo, testNow)

	ctx := WithRequestMetadata(context.Background(), RequestMetadata{
		IPAddress: "203.0.113.10",
		UserAgent: "test-agent",
		RequestID: "req-123",
	})
	_, err := svc.SetIdentityRoles(ctx, SetIdentityRolesInput{
		IdentityID: "id-1",
		Roles:      []string{"admin"},
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("SetIdentityRoles() error = %v", err)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(auditRepo.logs))
	}
	entry := auditRepo.logs[0]
	if entry.EventType != "identity.roles.updated" {
		t.Fatalf("unexpected event type: %q", entry.EventType)
	}
	if entry.ActorID != "actor-1" {
		t.Fatalf("unexpected actor id: %q", entry.ActorID)
	}
	if entry.TargetID != "id-1" {
		t.Fatalf("unexpected target id: %q", entry.TargetID)
	}
	if entry.IPAddress != "203.0.113.10" || entry.UserAgent != "test-agent" || entry.RequestID != "req-123" {
		t.Fatalf("unexpected request metadata: %+v", entry)
	}
	if entry.Result != audit.ResultSuccess {
		t.Fatalf("unexpected result: %q", entry.Result)
	}
}

func TestServiceSetIdentityRolesSkipsAuditWhenNil(t *testing.T) {
	svc := NewService(&stubAppManager{}, &stubIdentityRoleManager{}, nil, testNow)

	_, err := svc.SetIdentityRoles(context.Background(), SetIdentityRolesInput{
		IdentityID: "id-1",
		Roles:      []string{"admin"},
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("SetIdentityRoles() should not error with nil audit repo, got %v", err)
	}
}

func TestServiceSetIdentityRolesForwardsUpstreamError(t *testing.T) {
	expected := errors.New("kratos unavailable")
	svc := NewService(&stubAppManager{}, &stubIdentityRoleManager{err: expected}, &stubAuditRepository{}, testNow)

	_, err := svc.SetIdentityRoles(context.Background(), SetIdentityRolesInput{
		IdentityID: "id-1",
		Roles:      []string{"admin"},
		ActorID:    "actor-1",
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestServiceSearchIdentitiesDelegatesToIdentityManager(t *testing.T) {
	mgr := &stubIdentityRoleManager{
		identities: []Identity{{ID: "id-1", Email: "user@example.com", State: IdentityStateActive}},
	}
	svc := NewService(&stubAppManager{}, mgr, nil, testNow)

	identities, err := svc.SearchIdentities(context.Background(), SearchIdentitiesInput{
		CredentialsIdentifier: "user@example.com",
		State:                 IdentityStateActive,
	})
	if err != nil {
		t.Fatalf("SearchIdentities() error = %v", err)
	}
	if len(identities) != 1 || identities[0].ID != "id-1" {
		t.Fatalf("unexpected identities: %+v", identities)
	}
	if mgr.searchFilter.CredentialsIdentifier != "user@example.com" {
		t.Fatalf("expected filter to be forwarded, got %+v", mgr.searchFilter)
	}
}

func TestServiceDisableIdentityWritesAuditLog(t *testing.T) {
	auditRepo := &stubAuditRepository{}
	mgr := &stubIdentityRoleManager{
		disabledIdentity: Identity{ID: "id-1", State: IdentityStateInactive},
	}
	svc := NewService(&stubAppManager{}, mgr, auditRepo, testNow)

	identity, err := svc.DisableIdentity(context.Background(), DisableIdentityInput{
		IdentityID: "id-1",
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("DisableIdentity() error = %v", err)
	}
	if identity.State != IdentityStateInactive {
		t.Fatalf("expected inactive identity, got %+v", identity)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].EventType != "identity.disabled" {
		t.Fatalf("unexpected event type: %q", auditRepo.logs[0].EventType)
	}
}

func TestServiceEnableIdentityWritesAuditLog(t *testing.T) {
	auditRepo := &stubAuditRepository{}
	mgr := &stubIdentityRoleManager{
		enabledIdentity: Identity{ID: "id-1", State: IdentityStateActive},
	}
	svc := NewService(&stubAppManager{}, mgr, auditRepo, testNow)

	identity, err := svc.EnableIdentity(context.Background(), EnableIdentityInput{
		IdentityID: "id-1",
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("EnableIdentity() error = %v", err)
	}
	if identity.State != IdentityStateActive {
		t.Fatalf("expected active identity, got %+v", identity)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].EventType != "identity.enabled" {
		t.Fatalf("unexpected event type: %q", auditRepo.logs[0].EventType)
	}
}

func TestServiceRevokeIdentitySessionsWritesAuditLog(t *testing.T) {
	auditRepo := &stubAuditRepository{}
	mgr := &stubIdentityRoleManager{}
	svc := NewService(&stubAppManager{}, mgr, auditRepo, testNow)

	if err := svc.RevokeIdentitySessions(context.Background(), RevokeIdentitySessionsInput{
		IdentityID: "id-1",
		ActorID:    "actor-1",
	}); err != nil {
		t.Fatalf("RevokeIdentitySessions() error = %v", err)
	}
	if mgr.revokedIdentityID != "id-1" {
		t.Fatalf("expected revoked identity id to be recorded, got %q", mgr.revokedIdentityID)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].EventType != "identity.sessions.revoked" {
		t.Fatalf("unexpected event type: %q", auditRepo.logs[0].EventType)
	}
}

func TestServiceDeleteIdentityWritesAuditLog(t *testing.T) {
	auditRepo := &stubAuditRepository{}
	mgr := &stubIdentityRoleManager{}
	svc := NewService(&stubAppManager{}, mgr, auditRepo, testNow)

	err := svc.DeleteIdentity(context.Background(), DeleteIdentityInput{
		IdentityID: "id-1",
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("DeleteIdentity() error = %v", err)
	}
	if mgr.deletedIdentityID != "id-1" {
		t.Fatalf("expected deleted identity id to be recorded, got %q", mgr.deletedIdentityID)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].EventType != "identity.deleted" {
		t.Fatalf("unexpected event type: %q", auditRepo.logs[0].EventType)
	}
}

func TestServiceListAuditLogsDelegatesToRepository(t *testing.T) {
	auditRepo := &stubAuditRepository{
		logs: []audit.Log{{EventType: "identity.disabled", ActorID: "actor-1"}},
	}
	svc := NewService(&stubAppManager{}, &stubIdentityRoleManager{}, auditRepo, testNow)

	logs, err := svc.ListAuditLogs(context.Background(), ListAuditLogsInput{
		ActorID: "actor-1",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}
	if len(logs) != 1 || logs[0].ActorID != "actor-1" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
	if auditRepo.lastListParams.ActorID != "actor-1" || auditRepo.lastListParams.Limit != 10 {
		t.Fatalf("unexpected list params: %+v", auditRepo.lastListParams)
	}
}

// ---------------------------------------------------------------------------
// normalizeRoles
// ---------------------------------------------------------------------------

func TestNormalizeRolesDeduplicatesAndSorts(t *testing.T) {
	got := normalizeRoles([]string{" Admin ", "platform-operator", "admin", ""})
	want := []string{"admin", "platform-operator"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestNormalizeRolesReturnsEmptyForBlankOnly(t *testing.T) {
	got := normalizeRoles([]string{"", "  ", "\t"})
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestNormalizeRolesReturnsEmptyForNilInput(t *testing.T) {
	got := normalizeRoles(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Delegation: CreateApp / ListApps / CreateOIDCClient / ListOIDCClients
// ---------------------------------------------------------------------------

func TestServiceCreateAppDelegatesToAppManager(t *testing.T) {
	mgr := &stubAppManager{}
	svc := NewService(mgr, &stubIdentityRoleManager{}, nil, testNow)

	created, err := svc.CreateApp(context.Background(), app.CreateAppInput{
		Name:      "Idol Web",
		Slug:      "idol-web",
		Type:      app.AppTypeWeb,
		PartyType: app.PartyTypeFirst,
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if created.Slug != "idol-web" {
		t.Fatalf("expected slug %q, got %q", "idol-web", created.Slug)
	}
}

func TestServiceCreateAppForwardsUpstreamError(t *testing.T) {
	expected := errors.New("storage error")
	svc := NewService(&stubAppManager{err: expected}, &stubIdentityRoleManager{}, nil, testNow)

	_, err := svc.CreateApp(context.Background(), app.CreateAppInput{
		Name:      "X",
		Slug:      "x",
		Type:      app.AppTypeWeb,
		PartyType: app.PartyTypeFirst,
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestServiceListAppsDelegatesToAppManager(t *testing.T) {
	mgr := &stubAppManager{
		apps: []app.App{{ID: uuid.New(), Slug: "idol-web"}},
	}
	svc := NewService(mgr, &stubIdentityRoleManager{}, nil, testNow)

	apps, err := svc.ListApps(context.Background())
	if err != nil {
		t.Fatalf("ListApps() error = %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
}

func TestServiceCreateOIDCClientDelegatesToAppManager(t *testing.T) {
	appID := uuid.New()
	mgr := &stubAppManager{
		clientReg: app.ClientRegistration{
			Client: app.OIDCClient{
				ID:            uuid.New(),
				HydraClientID: "client-id",
				AppID:         appID,
			},
		},
	}
	svc := NewService(mgr, &stubIdentityRoleManager{}, nil, testNow)

	reg, err := svc.CreateOIDCClient(context.Background(), appID, app.CreateOIDCClientInput{Name: "My Client"})
	if err != nil {
		t.Fatalf("CreateOIDCClient() error = %v", err)
	}
	if reg.Client.AppID != appID {
		t.Fatalf("expected app id %v, got %v", appID, reg.Client.AppID)
	}
}

func TestServiceListOIDCClientsDelegatesToAppManager(t *testing.T) {
	appID := uuid.New()
	mgr := &stubAppManager{
		clients: []app.OIDCClient{{ID: uuid.New(), AppID: appID}},
	}
	svc := NewService(mgr, &stubIdentityRoleManager{}, nil, testNow)

	clients, err := svc.ListOIDCClients(context.Background(), appID)
	if err != nil {
		t.Fatalf("ListOIDCClients() error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
}

// ---------------------------------------------------------------------------
// Stubs
// ---------------------------------------------------------------------------

type stubAppManager struct {
	apps      []app.App
	clients   []app.OIDCClient
	clientReg app.ClientRegistration
	err       error
}

func (m *stubAppManager) CreateApp(_ context.Context, input app.CreateAppInput) (app.App, error) {
	if m.err != nil {
		return app.App{}, m.err
	}
	return app.App{
		ID:        uuid.New(),
		Name:      input.Name,
		Slug:      input.Slug,
		Type:      input.Type,
		PartyType: input.PartyType,
		Status:    app.AppStatusActive,
	}, nil
}

func (m *stubAppManager) ListApps(_ context.Context) ([]app.App, error) {
	if m.err != nil {
		return nil, m.err
	}
	return append([]app.App(nil), m.apps...), nil
}

func (m *stubAppManager) CreateOIDCClient(_ context.Context, _ uuid.UUID, _ app.CreateOIDCClientInput) (app.ClientRegistration, error) {
	if m.err != nil {
		return app.ClientRegistration{}, m.err
	}
	return m.clientReg, nil
}

func (m *stubAppManager) ListOIDCClients(_ context.Context, _ uuid.UUID) ([]app.OIDCClient, error) {
	if m.err != nil {
		return nil, m.err
	}
	return append([]app.OIDCClient(nil), m.clients...), nil
}

type stubIdentityRoleManager struct {
	receivedID        string
	receivedRoles     []string
	searchFilter      SearchIdentitiesInput
	identities        []Identity
	disabledInput     DisableIdentityInput
	disabledIdentity  Identity
	enabledInput      EnableIdentityInput
	enabledIdentity   Identity
	revokedIdentityID string
	deletedIdentityID string
	err               error
}

func (m *stubIdentityRoleManager) SetIdentityRoles(_ context.Context, identityID string, roles []string) error {
	m.receivedID = identityID
	m.receivedRoles = append([]string(nil), roles...)
	return m.err
}

func (m *stubIdentityRoleManager) SearchIdentities(_ context.Context, input SearchIdentitiesInput) ([]Identity, error) {
	m.searchFilter = input
	if m.err != nil {
		return nil, m.err
	}
	return append([]Identity(nil), m.identities...), nil
}

func (m *stubIdentityRoleManager) DisableIdentity(_ context.Context, input DisableIdentityInput) (Identity, error) {
	m.disabledInput = input
	if m.err != nil {
		return Identity{}, m.err
	}
	if m.disabledIdentity.ID != "" {
		return m.disabledIdentity, nil
	}
	return Identity{ID: input.IdentityID, State: IdentityStateInactive}, nil
}

func (m *stubIdentityRoleManager) EnableIdentity(_ context.Context, input EnableIdentityInput) (Identity, error) {
	m.enabledInput = input
	if m.err != nil {
		return Identity{}, m.err
	}
	if m.enabledIdentity.ID != "" {
		return m.enabledIdentity, nil
	}
	return Identity{ID: input.IdentityID, State: IdentityStateActive}, nil
}

func (m *stubIdentityRoleManager) RevokeIdentitySessions(_ context.Context, identityID string) error {
	m.revokedIdentityID = identityID
	return m.err
}

func (m *stubIdentityRoleManager) DeleteIdentity(_ context.Context, identityID string) error {
	m.deletedIdentityID = identityID
	return m.err
}

type stubAuditRepository struct {
	logs           []audit.Log
	lastListParams audit.ListParams
}

func (r *stubAuditRepository) Write(_ context.Context, entry audit.Log) error {
	r.logs = append(r.logs, entry)
	return nil
}

func (r *stubAuditRepository) List(_ context.Context, params audit.ListParams) ([]audit.Log, error) {
	r.lastListParams = params
	return append([]audit.Log(nil), r.logs...), nil
}

func testNow() time.Time {
	return time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
}
