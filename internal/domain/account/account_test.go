package account_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/account"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

// ---- mock implementations ----

type mockMembershipRepo struct {
	upsertFn                 func(ctx context.Context, m account.AppMembership) (account.AppMembership, error)
	listByIdentityFn         func(ctx context.Context, id string) ([]account.AppMembership, error)
	listByAppIDFn            func(ctx context.Context, appID uuid.UUID) ([]account.AppMembership, error)
	updateStatusFn           func(ctx context.Context, appID uuid.UUID, identityID string, status account.MembershipStatus, actorID string, now time.Time) error
	updateStatusByIdentityFn func(ctx context.Context, identityID string, status account.MembershipStatus, actorID string, now time.Time) error
}

func (m *mockMembershipRepo) Upsert(ctx context.Context, mem account.AppMembership) (account.AppMembership, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, mem)
	}
	return mem, nil
}
func (m *mockMembershipRepo) ListByIdentity(ctx context.Context, id string) ([]account.AppMembership, error) {
	if m.listByIdentityFn != nil {
		return m.listByIdentityFn(ctx, id)
	}
	return nil, nil
}
func (m *mockMembershipRepo) ListByAppID(ctx context.Context, appID uuid.UUID) ([]account.AppMembership, error) {
	if m.listByAppIDFn != nil {
		return m.listByAppIDFn(ctx, appID)
	}
	return nil, nil
}
func (m *mockMembershipRepo) UpdateStatus(ctx context.Context, appID uuid.UUID, identityID string, status account.MembershipStatus, actorID string, now time.Time) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, appID, identityID, status, actorID, now)
	}
	return nil
}
func (m *mockMembershipRepo) UpdateStatusByIdentity(ctx context.Context, identityID string, status account.MembershipStatus, actorID string, now time.Time) error {
	if m.updateStatusByIdentityFn != nil {
		return m.updateStatusByIdentityFn(ctx, identityID, status, actorID, now)
	}
	return nil
}

type mockDeletionRepo struct {
	upsertScheduledFn func(ctx context.Context, req account.DeletionRequest) (account.DeletionRequest, error)
	getByIdentityFn   func(ctx context.Context, id string) (account.DeletionRequest, error)
	cancelFn          func(ctx context.Context, identityID, actorID string, t time.Time) error
	listDueFn         func(ctx context.Context, dueBefore time.Time, limit int) ([]account.DeletionRequest, error)
	markCompletedFn   func(ctx context.Context, identityID, actorID string, t time.Time) error
}

func (m *mockDeletionRepo) UpsertScheduled(ctx context.Context, req account.DeletionRequest) (account.DeletionRequest, error) {
	if m.upsertScheduledFn != nil {
		return m.upsertScheduledFn(ctx, req)
	}
	return req, nil
}
func (m *mockDeletionRepo) GetByIdentity(ctx context.Context, id string) (account.DeletionRequest, error) {
	if m.getByIdentityFn != nil {
		return m.getByIdentityFn(ctx, id)
	}
	return account.DeletionRequest{}, account.ErrDeletionRequestNotFound
}
func (m *mockDeletionRepo) Cancel(ctx context.Context, identityID, actorID string, t time.Time) error {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, identityID, actorID, t)
	}
	return nil
}
func (m *mockDeletionRepo) ListDue(ctx context.Context, dueBefore time.Time, limit int) ([]account.DeletionRequest, error) {
	if m.listDueFn != nil {
		return m.listDueFn(ctx, dueBefore, limit)
	}
	return nil, nil
}
func (m *mockDeletionRepo) MarkCompleted(ctx context.Context, identityID, actorID string, t time.Time) error {
	if m.markCompletedFn != nil {
		return m.markCompletedFn(ctx, identityID, actorID, t)
	}
	return nil
}

type mockAppDirectory struct {
	getByHydraClientIDFn func(ctx context.Context, id string) (app.App, error)
	getByIDFn            func(ctx context.Context, id uuid.UUID) (app.App, error)
}

func (m *mockAppDirectory) GetByHydraClientID(ctx context.Context, id string) (app.App, error) {
	if m.getByHydraClientIDFn != nil {
		return m.getByHydraClientIDFn(ctx, id)
	}
	return app.App{}, app.ErrAppNotFound
}
func (m *mockAppDirectory) GetByID(ctx context.Context, id uuid.UUID) (app.App, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return app.App{}, app.ErrAppNotFound
}

type mockIdentityLifecycle struct {
	revokeSessionsFn func(ctx context.Context, id string) error
	deleteIdentityFn func(ctx context.Context, id string) error
}

func (m *mockIdentityLifecycle) RevokeIdentitySessions(ctx context.Context, id string) error {
	if m.revokeSessionsFn != nil {
		return m.revokeSessionsFn(ctx, id)
	}
	return nil
}
func (m *mockIdentityLifecycle) DeleteIdentity(ctx context.Context, id string) error {
	if m.deleteIdentityFn != nil {
		return m.deleteIdentityFn(ctx, id)
	}
	return nil
}

type mockTokenResolver struct {
	resolveAppByTokenFn func(ctx context.Context, rawToken string) (app.App, error)
}

func (m *mockTokenResolver) ResolveAppByToken(ctx context.Context, rawToken string) (app.App, error) {
	if m.resolveAppByTokenFn != nil {
		return m.resolveAppByTokenFn(ctx, rawToken)
	}
	return app.App{}, errors.New("token not found")
}

type mockAuditRepo struct {
	logs []audit.Log
}

func (m *mockAuditRepo) Write(_ context.Context, entry audit.Log) error {
	m.logs = append(m.logs, entry)
	return nil
}

func (m *mockAuditRepo) List(_ context.Context, _ audit.ListParams) ([]audit.Log, error) {
	return m.logs, nil
}

// ---- helpers ----

var fixedNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func fixedClock() time.Time { return fixedNow }

func sampleApp() app.App {
	return app.App{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name:      "Test App",
		Slug:      "test-app",
		PartyType: app.PartyTypeFirst,
	}
}

func newService(memberships *mockMembershipRepo, deletions *mockDeletionRepo, apps *mockAppDirectory, identities *mockIdentityLifecycle, tokens *mockTokenResolver, auditLogs *mockAuditRepo) *account.Service {
	return account.NewService(memberships, deletions, apps, identities, tokens, auditLogs, fixedClock, 7*24*time.Hour)
}

// ---- tests ----

func TestEnsureMembershipForHydraClient_CreatesActiveMembership(t *testing.T) {
	sampleA := sampleApp()
	apps := &mockAppDirectory{
		getByHydraClientIDFn: func(_ context.Context, id string) (app.App, error) {
			if id == "hydra-client-1" {
				return sampleA, nil
			}
			return app.App{}, app.ErrAppNotFound
		},
	}
	var capturedMembership account.AppMembership
	memberships := &mockMembershipRepo{
		upsertFn: func(_ context.Context, m account.AppMembership) (account.AppMembership, error) {
			capturedMembership = m
			return m, nil
		},
	}
	svc := newService(memberships, &mockDeletionRepo{}, apps, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	err := svc.EnsureMembershipForHydraClient(context.Background(), "hydra-client-1", "identity-abc", "identity-abc")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMembership.AppID != sampleA.ID {
		t.Errorf("expected AppID %v, got %v", sampleA.ID, capturedMembership.AppID)
	}
	if capturedMembership.IdentityID != "identity-abc" {
		t.Errorf("expected IdentityID identity-abc, got %v", capturedMembership.IdentityID)
	}
	if capturedMembership.Status != account.MembershipStatusActive {
		t.Errorf("expected status active, got %v", capturedMembership.Status)
	}
}

func TestEnsureMembershipForHydraClient_IgnoresBlankInputs(t *testing.T) {
	cases := []struct {
		name       string
		clientID   string
		identityID string
	}{
		{"empty clientID", "", "identity-1"},
		{"whitespace clientID", "   ", "identity-1"},
		{"empty identityID", "client-1", ""},
		{"whitespace identityID", "client-1", "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			memberships := &mockMembershipRepo{
				upsertFn: func(_ context.Context, _ account.AppMembership) (account.AppMembership, error) {
					called = true
					return account.AppMembership{}, nil
				},
			}
			svc := newService(memberships, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})
			_ = svc.EnsureMembershipForHydraClient(context.Background(), tc.clientID, tc.identityID, "actor")
			if called {
				t.Error("expected no upsert call for blank inputs")
			}
		})
	}
}

func TestEnsureMembershipForHydraClient_PropagatesAppLookupError(t *testing.T) {
	apps := &mockAppDirectory{
		getByHydraClientIDFn: func(_ context.Context, _ string) (app.App, error) {
			return app.App{}, errors.New("db error")
		},
	}
	svc := newService(&mockMembershipRepo{}, &mockDeletionRepo{}, apps, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})
	err := svc.EnsureMembershipForHydraClient(context.Background(), "client-1", "identity-1", "actor")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveAppByToken_DelegatesToResolver(t *testing.T) {
	expected := sampleApp()
	tokens := &mockTokenResolver{
		resolveAppByTokenFn: func(_ context.Context, rawToken string) (app.App, error) {
			if rawToken == "valid-token" {
				return expected, nil
			}
			return app.App{}, errors.New("not found")
		},
	}
	svc := newService(&mockMembershipRepo{}, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, tokens, &mockAuditRepo{})

	got, err := svc.ResolveAppByToken(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected app ID %v, got %v", expected.ID, got.ID)
	}
}

func TestResolveAppByToken_NilResolverReturnsError(t *testing.T) {
	svc := account.NewService(&mockMembershipRepo{}, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, nil, &mockAuditRepo{}, fixedClock, 0)
	_, err := svc.ResolveAppByToken(context.Background(), "any-token")
	if err == nil {
		t.Fatal("expected error for nil token resolver")
	}
}

func TestListMembershipsForIdentity_TrimsWhitespaceAndDelegates(t *testing.T) {
	var capturedID string
	memberships := &mockMembershipRepo{
		listByIdentityFn: func(_ context.Context, id string) ([]account.AppMembership, error) {
			capturedID = id
			return []account.AppMembership{{IdentityID: id}}, nil
		},
	}
	svc := newService(memberships, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	results, err := svc.ListMembershipsForIdentity(context.Background(), "  identity-1  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "identity-1" {
		t.Errorf("expected trimmed ID, got %q", capturedID)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestListMembershipsForApp_Delegates(t *testing.T) {
	appID := uuid.New()
	memberships := &mockMembershipRepo{
		listByAppIDFn: func(_ context.Context, id uuid.UUID) ([]account.AppMembership, error) {
			return []account.AppMembership{{AppID: id}}, nil
		},
	}
	svc := newService(memberships, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	results, err := svc.ListMembershipsForApp(context.Background(), appID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].AppID != appID {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestDisconnectIdentityFromApp_RevokesAndWritesAudit(t *testing.T) {
	appID := uuid.New()
	var capturedStatus account.MembershipStatus
	memberships := &mockMembershipRepo{
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ string, status account.MembershipStatus, _ string, _ time.Time) error {
			capturedStatus = status
			return nil
		},
	}
	auditRepo := &mockAuditRepo{}
	svc := newService(memberships, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, auditRepo)

	err := svc.DisconnectIdentityFromApp(context.Background(), "identity-1", appID, "identity-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStatus != account.MembershipStatusRevoked {
		t.Errorf("expected revoked, got %v", capturedStatus)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].ActorType != audit.ActorTypeIdentity {
		t.Errorf("expected actor type identity, got %v", auditRepo.logs[0].ActorType)
	}
}

func TestDisconnectIdentityFromApp_PropagatesRepoError(t *testing.T) {
	memberships := &mockMembershipRepo{
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ account.MembershipStatus, _ string, _ time.Time) error {
			return errors.New("db error")
		},
	}
	svc := newService(memberships, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	err := svc.DisconnectIdentityFromApp(context.Background(), "identity-1", uuid.New(), "actor")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRevokeAppUser_RevokesWithAppClientActor(t *testing.T) {
	auditRepo := &mockAuditRepo{}
	memberships := &mockMembershipRepo{}
	svc := newService(memberships, &mockDeletionRepo{}, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, auditRepo)

	err := svc.RevokeAppUser(context.Background(), uuid.New(), "identity-1", "app-actor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].ActorType != audit.ActorTypeAppClient {
		t.Errorf("expected actor type app_client, got %v", auditRepo.logs[0].ActorType)
	}
}

func TestScheduleDeletion_SetsGracePeriodAndWritesAudit(t *testing.T) {
	gracePeriod := 7 * 24 * time.Hour
	var capturedRequest account.DeletionRequest
	deletions := &mockDeletionRepo{
		upsertScheduledFn: func(_ context.Context, req account.DeletionRequest) (account.DeletionRequest, error) {
			capturedRequest = req
			return req, nil
		},
	}
	auditRepo := &mockAuditRepo{}
	svc := account.NewService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, auditRepo, fixedClock, gracePeriod)

	result, err := svc.ScheduleDeletion(context.Background(), "  identity-1  ", "identity-1", "some reason")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRequest.IdentityID != "identity-1" {
		t.Errorf("expected trimmed identity ID, got %q", capturedRequest.IdentityID)
	}
	expectedScheduled := fixedNow.Add(gracePeriod)
	if !result.ScheduledFor.Equal(expectedScheduled) {
		t.Errorf("expected scheduled_for %v, got %v", expectedScheduled, result.ScheduledFor)
	}
	if result.Status != account.DeletionStatusScheduled {
		t.Errorf("expected status scheduled, got %v", result.Status)
	}
	if len(auditRepo.logs) != 1 || auditRepo.logs[0].EventType != "identity.deletion.scheduled" {
		t.Errorf("expected audit log for deletion.scheduled, got %v", auditRepo.logs)
	}
}

func TestScheduleDeletion_PropagatesRepoError(t *testing.T) {
	deletions := &mockDeletionRepo{
		upsertScheduledFn: func(_ context.Context, _ account.DeletionRequest) (account.DeletionRequest, error) {
			return account.DeletionRequest{}, errors.New("db error")
		},
	}
	svc := newService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	_, err := svc.ScheduleDeletion(context.Background(), "identity-1", "actor", "reason")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCancelDeletion_CancelsAndWritesAudit(t *testing.T) {
	var capturedID string
	deletions := &mockDeletionRepo{
		cancelFn: func(_ context.Context, identityID, _ string, _ time.Time) error {
			capturedID = identityID
			return nil
		},
	}
	auditRepo := &mockAuditRepo{}
	svc := newService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, auditRepo)

	err := svc.CancelDeletion(context.Background(), "identity-1", "identity-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "identity-1" {
		t.Errorf("expected trimmed ID, got %q", capturedID)
	}
	if len(auditRepo.logs) != 1 || auditRepo.logs[0].EventType != "identity.deletion.cancelled" {
		t.Errorf("expected audit log for deletion.cancelled, got %v", auditRepo.logs)
	}
}

func TestGetDeletionRequest_ReturnsNilWhenNotFound(t *testing.T) {
	deletions := &mockDeletionRepo{
		getByIdentityFn: func(_ context.Context, _ string) (account.DeletionRequest, error) {
			return account.DeletionRequest{}, account.ErrDeletionRequestNotFound
		},
	}
	svc := newService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	result, err := svc.GetDeletionRequest(context.Background(), "identity-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestGetDeletionRequest_ReturnsDeletionRequest(t *testing.T) {
	expected := account.DeletionRequest{
		ID:         uuid.New(),
		IdentityID: "identity-1",
		Status:     account.DeletionStatusScheduled,
	}
	deletions := &mockDeletionRepo{
		getByIdentityFn: func(_ context.Context, _ string) (account.DeletionRequest, error) {
			return expected, nil
		},
	}
	svc := newService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	result, err := svc.GetDeletionRequest(context.Background(), "identity-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.ID != expected.ID {
		t.Errorf("expected deletion request ID %v, got %v", expected.ID, result)
	}
}

func TestGetDeletionRequest_PropagatesUnknownError(t *testing.T) {
	deletions := &mockDeletionRepo{
		getByIdentityFn: func(_ context.Context, _ string) (account.DeletionRequest, error) {
			return account.DeletionRequest{}, errors.New("unexpected db error")
		},
	}
	svc := newService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{})

	_, err := svc.GetDeletionRequest(context.Background(), "identity-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProcessDueDeletionRequests_ExecutesFullLifecycle(t *testing.T) {
	identityID := "identity-to-delete"
	requests := []account.DeletionRequest{{ID: uuid.New(), IdentityID: identityID}}

	deletions := &mockDeletionRepo{
		listDueFn: func(_ context.Context, _ time.Time, _ int) ([]account.DeletionRequest, error) {
			return requests, nil
		},
	}

	var revokedSessionID, deletedIdentityID, completedID string
	identities := &mockIdentityLifecycle{
		revokeSessionsFn: func(_ context.Context, id string) error {
			revokedSessionID = id
			return nil
		},
		deleteIdentityFn: func(_ context.Context, id string) error {
			deletedIdentityID = id
			return nil
		},
	}

	deletions.markCompletedFn = func(_ context.Context, id, _ string, _ time.Time) error {
		completedID = id
		return nil
	}
	auditRepo := &mockAuditRepo{}
	svc := newService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, identities, &mockTokenResolver{}, auditRepo)

	err := svc.ProcessDueDeletionRequests(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revokedSessionID != identityID {
		t.Errorf("expected sessions revoked for %q, got %q", identityID, revokedSessionID)
	}
	if deletedIdentityID != identityID {
		t.Errorf("expected identity deleted %q, got %q", identityID, deletedIdentityID)
	}
	if completedID != identityID {
		t.Errorf("expected deletion marked completed for %q, got %q", identityID, completedID)
	}
	if len(auditRepo.logs) != 1 || auditRepo.logs[0].EventType != "identity.deletion.completed" {
		t.Errorf("expected audit log for deletion.completed")
	}
}

func TestProcessDueDeletionRequests_SkipsBlankIdentityIDs(t *testing.T) {
	deletions := &mockDeletionRepo{
		listDueFn: func(_ context.Context, _ time.Time, _ int) ([]account.DeletionRequest, error) {
			return []account.DeletionRequest{{ID: uuid.New(), IdentityID: "   "}}, nil
		},
	}
	deleted := false
	identities := &mockIdentityLifecycle{
		deleteIdentityFn: func(_ context.Context, _ string) error {
			deleted = true
			return nil
		},
	}
	svc := newService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, identities, &mockTokenResolver{}, &mockAuditRepo{})

	_ = svc.ProcessDueDeletionRequests(context.Background(), 10)
	if deleted {
		t.Error("expected blank identity ID to be skipped")
	}
}

func TestProcessDueDeletionRequests_NilDependenciesAreNoOp(t *testing.T) {
	svc := account.NewService(nil, nil, nil, nil, nil, nil, fixedClock, 0)
	err := svc.ProcessDueDeletionRequests(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error with nil dependencies: %v", err)
	}
}

func TestNewService_DefaultsGracePeriodTo7Days(t *testing.T) {
	var capturedScheduledFor time.Time
	deletions := &mockDeletionRepo{
		upsertScheduledFn: func(_ context.Context, req account.DeletionRequest) (account.DeletionRequest, error) {
			capturedScheduledFor = req.ScheduledFor
			return req, nil
		},
	}
	// gracePeriod=0 should default to 7 days
	svc := account.NewService(&mockMembershipRepo{}, deletions, &mockAppDirectory{}, &mockIdentityLifecycle{}, &mockTokenResolver{}, &mockAuditRepo{}, fixedClock, 0)

	_, err := svc.ScheduleDeletion(context.Background(), "identity-1", "actor", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := fixedNow.Add(7 * 24 * time.Hour)
	if !capturedScheduledFor.Equal(expected) {
		t.Errorf("expected default grace period 7d, got scheduled_for %v", capturedScheduledFor)
	}
}
