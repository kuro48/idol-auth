package appreg_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

// --- in-memory stub ---

type stubRepo struct {
	requests map[uuid.UUID]appreg.Request
	events   []appreg.Event
}

func newStubRepo() *stubRepo {
	return &stubRepo{requests: make(map[uuid.UUID]appreg.Request)}
}

func (r *stubRepo) Create(_ context.Context, req appreg.Request) (appreg.Request, error) {
	r.requests[req.ID] = req
	return req, nil
}

func (r *stubRepo) Update(_ context.Context, req appreg.Request) (appreg.Request, error) {
	existing, ok := r.requests[req.ID]
	if !ok {
		return appreg.Request{}, appreg.ErrRequestNotFound
	}
	if existing.Version != req.Version-1 {
		return appreg.Request{}, appreg.ErrAlreadyDecided
	}
	r.requests[req.ID] = req
	return req, nil
}

func (r *stubRepo) GetByID(_ context.Context, id uuid.UUID) (appreg.Request, error) {
	req, ok := r.requests[id]
	if !ok {
		return appreg.Request{}, appreg.ErrRequestNotFound
	}
	return req, nil
}

func (r *stubRepo) ListByIdentity(_ context.Context, identityID string) ([]appreg.Request, error) {
	var out []appreg.Request
	for _, req := range r.requests {
		if req.IdentityID == identityID {
			out = append(out, req)
		}
	}
	return out, nil
}

func (r *stubRepo) ListForAdmin(_ context.Context, statusFilter string) ([]appreg.Request, error) {
	var out []appreg.Request
	for _, req := range r.requests {
		if statusFilter == "" || string(req.Status) == statusFilter {
			out = append(out, req)
		}
	}
	return out, nil
}

func (r *stubRepo) HasPendingByNameAndIdentity(_ context.Context, name, identityID string, excludeID uuid.UUID) (bool, error) {
	for _, req := range r.requests {
		if req.IdentityID == identityID && req.Name == name && req.ID != excludeID {
			switch req.Status {
			case appreg.StatusPending, appreg.StatusUnderReview, appreg.StatusChangesRequested:
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *stubRepo) AppendEvent(_ context.Context, event appreg.Event) error {
	r.events = append(r.events, event)
	return nil
}

// --- helpers ---

func validInput() appreg.SubmitInput {
	return appreg.SubmitInput{
		Name:         "My App",
		Type:         "web",
		Description:  "A test application",
		Purpose:      strings.Repeat("x", 200),
		ContactEmail: "dev@example.com",
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"openid"},
	}
}

func newSvc(repo appreg.Repository) *appreg.Service {
	return appreg.NewService(repo, appreg.NoopNotifier{}, func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})
}

// --- tests ---

func TestSubmit_Success(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, err := svc.Submit(context.Background(), "identity-1", validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Status != appreg.StatusPending {
		t.Errorf("want status=pending, got %s", req.Status)
	}
	if req.IdentityID != "identity-1" {
		t.Errorf("want identity_id=identity-1, got %s", req.IdentityID)
	}
	if len(repo.events) != 1 || repo.events[0].EventType != "submitted" {
		t.Errorf("expected submitted event, got %+v", repo.events)
	}
}

func TestSubmit_DuplicatePending(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)
	input := validInput()

	if _, err := svc.Submit(context.Background(), "identity-1", input); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != appreg.ErrDuplicatePending {
		t.Errorf("want ErrDuplicatePending, got %v", err)
	}
}

func TestSubmit_ValidationError(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	input := validInput()
	input.Name = ""
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != appreg.ErrInvalidName {
		t.Errorf("want ErrInvalidName, got %v", err)
	}
}

func TestWithdraw_Success(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	withdrawn, err := svc.Withdraw(context.Background(), req.ID, "identity-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if withdrawn.Status != appreg.StatusWithdrawn {
		t.Errorf("want status=withdrawn, got %s", withdrawn.Status)
	}
}

func TestWithdraw_WrongOwner(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	_, err := svc.Withdraw(context.Background(), req.ID, "identity-2")
	if err != appreg.ErrNotOwner {
		t.Errorf("want ErrNotOwner, got %v", err)
	}
}

func TestWithdraw_AlreadyApproved(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	approved := req
	approved.Status = appreg.StatusApproved
	approved.Version = 1
	repo.requests[req.ID] = approved

	_, err := svc.Withdraw(context.Background(), req.ID, "identity-1")
	if err != appreg.ErrCannotWithdraw {
		t.Errorf("want ErrCannotWithdraw, got %v", err)
	}
}

func TestResubmit_OnlyFromChangesRequested(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	_, err := svc.Resubmit(context.Background(), req.ID, "identity-1", validInput())
	if err != appreg.ErrCannotResubmit {
		t.Errorf("want ErrCannotResubmit from pending state, got %v", err)
	}
}

func TestResubmit_Success(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	cr := req
	cr.Status = appreg.StatusChangesRequested
	cr.Version = 1
	repo.requests[req.ID] = cr

	newInput := validInput()
	newInput.Name = "My App Updated"
	resubmitted, err := svc.Resubmit(context.Background(), req.ID, "identity-1", newInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resubmitted.Status != appreg.StatusPending {
		t.Errorf("want status=pending after resubmit, got %s", resubmitted.Status)
	}
	if resubmitted.Name != "My App Updated" {
		t.Errorf("want updated name, got %s", resubmitted.Name)
	}
}

func TestApprove_AlreadyDecided(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	if _, err := svc.BeginReview(context.Background(), req.ID, "admin-1"); err != nil {
		t.Fatalf("begin review failed: %v", err)
	}
	appID := uuid.New()
	clientID := uuid.New()

	if _, err := svc.Approve(context.Background(), req.ID, "admin-1", appID, clientID); err != nil {
		t.Fatalf("first approve failed: %v", err)
	}
	_, err := svc.Approve(context.Background(), req.ID, "admin-1", appID, clientID)
	if err != appreg.ErrAlreadyDecided {
		t.Errorf("want ErrAlreadyDecided on double-approve, got %v", err)
	}
}

func TestBeginReview_LocksRequest(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	reviewing, err := svc.BeginReview(context.Background(), req.ID, "admin-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reviewing.Status != appreg.StatusUnderReview {
		t.Fatalf("want status=under_review, got %s", reviewing.Status)
	}
	if reviewing.ReviewerID != "admin-1" {
		t.Fatalf("want reviewer_id admin-1, got %q", reviewing.ReviewerID)
	}
	_, err = svc.Reject(context.Background(), req.ID, "admin-2", "late reject")
	if err != appreg.ErrInvalidTransition {
		t.Fatalf("want ErrInvalidTransition after review lock, got %v", err)
	}
}

func TestApprove_RequiresBeginReview(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	_, err := svc.Approve(context.Background(), req.ID, "admin-1", uuid.New(), uuid.New())
	if err != appreg.ErrInvalidTransition {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

func TestReject_SetsStatusAndNote(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	rejected, err := svc.Reject(context.Background(), req.ID, "admin-1", "policy violation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rejected.Status != appreg.StatusRejected {
		t.Errorf("want status=rejected, got %s", rejected.Status)
	}
	if rejected.ReviewerNote != "policy violation" {
		t.Errorf("want reviewer_note set, got %q", rejected.ReviewerNote)
	}
}

func TestRequestChanges_Cycle(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	changed, err := svc.RequestChanges(context.Background(), req.ID, "admin-1", "please add privacy policy url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed.Status != appreg.StatusChangesRequested {
		t.Errorf("want status=changes_requested, got %s", changed.Status)
	}

	newInput := validInput()
	newInput.PrivacyPolicyURL = "https://example.com/privacy"
	resubmitted, err := svc.Resubmit(context.Background(), req.ID, "identity-1", newInput)
	if err != nil {
		t.Fatalf("resubmit after changes_requested failed: %v", err)
	}
	if resubmitted.Status != appreg.StatusPending {
		t.Errorf("want status=pending after resubmit, got %s", resubmitted.Status)
	}
}

func TestGetForOwner_Success(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	got, err := svc.GetForOwner(context.Background(), req.ID, "identity-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != req.ID {
		t.Errorf("want id=%s, got %s", req.ID, got.ID)
	}
}

func TestGetForOwner_WrongOwner(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	_, err := svc.GetForOwner(context.Background(), req.ID, "identity-2")
	if err != appreg.ErrNotOwner {
		t.Errorf("want ErrNotOwner, got %v", err)
	}
}

func TestGetForOwner_NotFound(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	_, err := svc.GetForOwner(context.Background(), uuid.New(), "identity-1")
	if err != appreg.ErrRequestNotFound {
		t.Errorf("want ErrRequestNotFound, got %v", err)
	}
}

func TestListMine(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	_, _ = svc.Submit(context.Background(), "identity-1", validInput())
	input2 := validInput()
	input2.Name = "Another App"
	_, _ = svc.Submit(context.Background(), "identity-2", input2)

	reqs, err := svc.ListMine(context.Background(), "identity-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 1 {
		t.Errorf("want 1 result for identity-1, got %d", len(reqs))
	}
}

func TestListForAdmin_NoFilter(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	_, _ = svc.Submit(context.Background(), "identity-1", validInput())
	input2 := validInput()
	input2.Name = "Another App"
	_, _ = svc.Submit(context.Background(), "identity-2", input2)

	reqs, err := svc.ListForAdmin(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 2 {
		t.Errorf("want 2 results, got %d", len(reqs))
	}
}

func TestListForAdmin_WithFilter(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	_, _ = svc.Reject(context.Background(), req.ID, "admin-1", "not suitable")

	input2 := validInput()
	input2.Name = "Another App"
	_, _ = svc.Submit(context.Background(), "identity-2", input2)

	reqs, err := svc.ListForAdmin(context.Background(), "pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 1 {
		t.Errorf("want 1 pending result, got %d", len(reqs))
	}
}

func TestGetByID_Found(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	req, _ := svc.Submit(context.Background(), "identity-1", validInput())
	got, err := svc.GetByID(context.Background(), req.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != req.ID {
		t.Errorf("want id=%s, got %s", req.ID, got.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	_, err := svc.GetByID(context.Background(), uuid.New())
	if err != appreg.ErrRequestNotFound {
		t.Errorf("want ErrRequestNotFound, got %v", err)
	}
}

func TestValidation_InvalidType(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	input := validInput()
	input.Type = "invalid"
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != appreg.ErrInvalidType {
		t.Errorf("want ErrInvalidType, got %v", err)
	}
}

func TestValidation_InvalidEmail(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	input := validInput()
	input.ContactEmail = "not-an-email"
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != appreg.ErrInvalidEmail {
		t.Errorf("want ErrInvalidEmail, got %v", err)
	}
}

func TestValidation_InvalidRedirectURI_HTTP_NonLoopback(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	input := validInput()
	input.RedirectURIs = []string{"http://evil.com/callback"}
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != appreg.ErrInvalidRedirectURI {
		t.Errorf("want ErrInvalidRedirectURI for http non-loopback, got %v", err)
	}
}

func TestValidation_RedirectURI_LocalhostAllowed(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	input := validInput()
	input.RedirectURIs = []string{"http://localhost:8080/callback"}
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != nil {
		t.Errorf("expected localhost http URI to be valid, got %v", err)
	}
}

func TestValidation_NativeCustomScheme(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	input := validInput()
	input.Type = "native"
	input.RedirectURIs = []string{"myapp://callback"}
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != nil {
		t.Errorf("expected custom scheme native URI to be valid, got %v", err)
	}
}

func TestValidation_InvalidHomepageURL(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)

	input := validInput()
	input.HomepageURL = "http://example.com"
	_, err := svc.Submit(context.Background(), "identity-1", input)
	if err != appreg.ErrInvalidURL {
		t.Errorf("want ErrInvalidURL for non-https homepage, got %v", err)
	}
}
