package appreg

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Repository persists registration requests and their events.
type Repository interface {
	Create(ctx context.Context, req Request) (Request, error)
	Update(ctx context.Context, req Request) (Request, error)
	GetByID(ctx context.Context, id uuid.UUID) (Request, error)
	ListByIdentity(ctx context.Context, identityID string) ([]Request, error)
	ListForAdmin(ctx context.Context, statusFilter string) ([]Request, error)
	HasPendingByNameAndIdentity(ctx context.Context, name, identityID string, excludeID uuid.UUID) (bool, error)
	AppendEvent(ctx context.Context, event Event) error
}

// Service orchestrates the registration request lifecycle.
type Service struct {
	repo     Repository
	notifier Notifier
	now      func() time.Time
}

func NewService(repo Repository, notifier Notifier, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	if notifier == nil {
		notifier = NoopNotifier{}
	}
	return &Service{repo: repo, notifier: notifier, now: now}
}

// Submit creates a new registration request for the given developer identity.
func (s *Service) Submit(ctx context.Context, identityID string, input SubmitInput) (Request, error) {
	if err := input.Validate(); err != nil {
		return Request{}, err
	}

	dup, err := s.repo.HasPendingByNameAndIdentity(ctx, input.Name, identityID, uuid.Nil)
	if err != nil {
		return Request{}, fmt.Errorf("check duplicate: %w", err)
	}
	if dup {
		return Request{}, ErrDuplicatePending
	}

	now := s.now().UTC()
	req := Request{
		ID:                     uuid.New(),
		IdentityID:             identityID,
		Status:                 StatusPending,
		Name:                   input.Name,
		Slug:                   slugifyName(input.Name),
		Type:                   input.Type,
		Description:            input.Description,
		HomepageURL:            input.HomepageURL,
		PrivacyPolicyURL:       input.PrivacyPolicyURL,
		TermsURL:               input.TermsURL,
		ContactEmail:           input.ContactEmail,
		Organization:           input.Organization,
		Purpose:                input.Purpose,
		RedirectURIs:           input.RedirectURIs,
		PostLogoutRedirectURIs: input.PostLogoutRedirectURIs,
		Scopes:                 input.Scopes,
		Version:                0,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	created, err := s.repo.Create(ctx, req)
	if err != nil {
		return Request{}, fmt.Errorf("create request: %w", err)
	}

	_ = s.repo.AppendEvent(ctx, Event{
		ID:         uuid.New(),
		RequestID:  created.ID,
		ActorType:  "developer",
		ActorID:    identityID,
		EventType:  "submitted",
		OccurredAt: now,
	})
	_ = s.notifier.NotifySubmitted(ctx, created)

	return created, nil
}

// GetForOwner returns a request only if it belongs to the given identity.
func (s *Service) GetForOwner(ctx context.Context, id uuid.UUID, identityID string) (Request, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Request{}, err
	}
	if req.IdentityID != identityID {
		return Request{}, ErrNotOwner
	}
	return req, nil
}

// ListMine returns all requests submitted by the given identity.
func (s *Service) ListMine(ctx context.Context, identityID string) ([]Request, error) {
	return s.repo.ListByIdentity(ctx, identityID)
}

// Withdraw cancels a request that is still in progress.
func (s *Service) Withdraw(ctx context.Context, id uuid.UUID, identityID string) (Request, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Request{}, err
	}
	if req.IdentityID != identityID {
		return Request{}, ErrNotOwner
	}
	if !req.CanWithdraw() {
		return Request{}, ErrCannotWithdraw
	}

	now := s.now().UTC()
	updated := Request{
		ID:                     req.ID,
		IdentityID:             req.IdentityID,
		Status:                 StatusWithdrawn,
		Name:                   req.Name,
		Slug:                   req.Slug,
		Type:                   req.Type,
		Description:            req.Description,
		HomepageURL:            req.HomepageURL,
		PrivacyPolicyURL:       req.PrivacyPolicyURL,
		TermsURL:               req.TermsURL,
		ContactEmail:           req.ContactEmail,
		Organization:           req.Organization,
		Purpose:                req.Purpose,
		RedirectURIs:           req.RedirectURIs,
		PostLogoutRedirectURIs: req.PostLogoutRedirectURIs,
		Scopes:                 req.Scopes,
		ReviewerID:             req.ReviewerID,
		ReviewerNote:           req.ReviewerNote,
		DecidedAt:              req.DecidedAt,
		CreatedAppID:           req.CreatedAppID,
		CreatedClientID:        req.CreatedClientID,
		Version:                req.Version + 1,
		CreatedAt:              req.CreatedAt,
		UpdatedAt:              now,
	}

	saved, err := s.repo.Update(ctx, updated)
	if err != nil {
		return Request{}, fmt.Errorf("withdraw request: %w", err)
	}

	_ = s.repo.AppendEvent(ctx, Event{
		ID:         uuid.New(),
		RequestID:  id,
		ActorType:  "developer",
		ActorID:    identityID,
		EventType:  "withdrawn",
		OccurredAt: now,
	})

	return saved, nil
}

// Resubmit updates and re-queues a request that had changes requested.
func (s *Service) Resubmit(ctx context.Context, id uuid.UUID, identityID string, input SubmitInput) (Request, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Request{}, err
	}
	if req.IdentityID != identityID {
		return Request{}, ErrNotOwner
	}
	if req.Status != StatusChangesRequested {
		return Request{}, ErrCannotResubmit
	}

	if err := input.Validate(); err != nil {
		return Request{}, err
	}

	dup, err := s.repo.HasPendingByNameAndIdentity(ctx, input.Name, identityID, id)
	if err != nil {
		return Request{}, fmt.Errorf("check duplicate: %w", err)
	}
	if dup {
		return Request{}, ErrDuplicatePending
	}

	now := s.now().UTC()
	updated := Request{
		ID:                     req.ID,
		IdentityID:             req.IdentityID,
		Status:                 StatusPending,
		Name:                   input.Name,
		Slug:                   slugifyName(input.Name),
		Type:                   input.Type,
		Description:            input.Description,
		HomepageURL:            input.HomepageURL,
		PrivacyPolicyURL:       input.PrivacyPolicyURL,
		TermsURL:               input.TermsURL,
		ContactEmail:           input.ContactEmail,
		Organization:           input.Organization,
		Purpose:                input.Purpose,
		RedirectURIs:           input.RedirectURIs,
		PostLogoutRedirectURIs: input.PostLogoutRedirectURIs,
		Scopes:                 input.Scopes,
		ReviewerID:             req.ReviewerID,
		ReviewerNote:           "",
		DecidedAt:              req.DecidedAt,
		CreatedAppID:           req.CreatedAppID,
		CreatedClientID:        req.CreatedClientID,
		Version:                req.Version + 1,
		CreatedAt:              req.CreatedAt,
		UpdatedAt:              now,
	}

	saved, err := s.repo.Update(ctx, updated)
	if err != nil {
		return Request{}, fmt.Errorf("resubmit request: %w", err)
	}

	_ = s.repo.AppendEvent(ctx, Event{
		ID:         uuid.New(),
		RequestID:  id,
		ActorType:  "developer",
		ActorID:    identityID,
		EventType:  "resubmitted",
		OccurredAt: now,
	})
	_ = s.notifier.NotifySubmitted(ctx, saved)

	return saved, nil
}

// ListForAdmin returns all requests, optionally filtered by status.
func (s *Service) ListForAdmin(ctx context.Context, statusFilter string) ([]Request, error) {
	return s.repo.ListForAdmin(ctx, statusFilter)
}

// GetByID returns a request by ID (admin use — no ownership check).
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Request, error) {
	return s.repo.GetByID(ctx, id)
}

// BeginReview claims a pending request before approval provisioning begins.
func (s *Service) BeginReview(ctx context.Context, id uuid.UUID, reviewerID string) (Request, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Request{}, err
	}
	if req.IsTerminal() {
		return Request{}, ErrAlreadyDecided
	}
	if req.Status == StatusUnderReview {
		return Request{}, ErrInvalidTransition
	}
	if req.Status != StatusPending && req.Status != StatusChangesRequested {
		return Request{}, ErrInvalidTransition
	}

	now := s.now().UTC()
	updated := Request{
		ID:                     req.ID,
		IdentityID:             req.IdentityID,
		Status:                 StatusUnderReview,
		Name:                   req.Name,
		Slug:                   req.Slug,
		Type:                   req.Type,
		Description:            req.Description,
		HomepageURL:            req.HomepageURL,
		PrivacyPolicyURL:       req.PrivacyPolicyURL,
		TermsURL:               req.TermsURL,
		ContactEmail:           req.ContactEmail,
		Organization:           req.Organization,
		Purpose:                req.Purpose,
		RedirectURIs:           req.RedirectURIs,
		PostLogoutRedirectURIs: req.PostLogoutRedirectURIs,
		Scopes:                 req.Scopes,
		ReviewerID:             reviewerID,
		ReviewerNote:           req.ReviewerNote,
		DecidedAt:              req.DecidedAt,
		CreatedAppID:           req.CreatedAppID,
		CreatedClientID:        req.CreatedClientID,
		Version:                req.Version + 1,
		CreatedAt:              req.CreatedAt,
		UpdatedAt:              now,
	}

	saved, err := s.repo.Update(ctx, updated)
	if err != nil {
		return Request{}, fmt.Errorf("begin review: %w", err)
	}

	_ = s.repo.AppendEvent(ctx, Event{
		ID:         uuid.New(),
		RequestID:  id,
		ActorType:  "admin",
		ActorID:    reviewerID,
		EventType:  "review_started",
		OccurredAt: now,
	})

	return saved, nil
}

// Approve marks a request as approved. The caller must provision the OIDC client
// first and pass the resulting appID and clientID.
func (s *Service) Approve(ctx context.Context, id uuid.UUID, reviewerID string, appID, clientID uuid.UUID) (Request, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Request{}, err
	}
	if req.IsTerminal() {
		return Request{}, ErrAlreadyDecided
	}
	if req.Status != StatusUnderReview || req.ReviewerID != reviewerID {
		return Request{}, ErrInvalidTransition
	}

	now := s.now().UTC()
	updated := Request{
		ID:                     req.ID,
		IdentityID:             req.IdentityID,
		Status:                 StatusApproved,
		Name:                   req.Name,
		Slug:                   req.Slug,
		Type:                   req.Type,
		Description:            req.Description,
		HomepageURL:            req.HomepageURL,
		PrivacyPolicyURL:       req.PrivacyPolicyURL,
		TermsURL:               req.TermsURL,
		ContactEmail:           req.ContactEmail,
		Organization:           req.Organization,
		Purpose:                req.Purpose,
		RedirectURIs:           req.RedirectURIs,
		PostLogoutRedirectURIs: req.PostLogoutRedirectURIs,
		Scopes:                 req.Scopes,
		ReviewerID:             reviewerID,
		ReviewerNote:           req.ReviewerNote,
		DecidedAt:              &now,
		CreatedAppID:           &appID,
		CreatedClientID:        &clientID,
		Version:                req.Version + 1,
		CreatedAt:              req.CreatedAt,
		UpdatedAt:              now,
	}

	saved, err := s.repo.Update(ctx, updated)
	if err != nil {
		return Request{}, fmt.Errorf("approve request: %w", err)
	}

	_ = s.repo.AppendEvent(ctx, Event{
		ID:         uuid.New(),
		RequestID:  id,
		ActorType:  "admin",
		ActorID:    reviewerID,
		EventType:  "approved",
		OccurredAt: now,
	})
	_ = s.notifier.NotifyApproved(ctx, saved)

	return saved, nil
}

// Reject transitions a request to rejected with an optional reviewer note.
func (s *Service) Reject(ctx context.Context, id uuid.UUID, reviewerID, note string) (Request, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Request{}, err
	}
	if req.IsTerminal() {
		return Request{}, ErrAlreadyDecided
	}
	if req.Status == StatusUnderReview {
		return Request{}, ErrInvalidTransition
	}

	now := s.now().UTC()
	updated := Request{
		ID:                     req.ID,
		IdentityID:             req.IdentityID,
		Status:                 StatusRejected,
		Name:                   req.Name,
		Slug:                   req.Slug,
		Type:                   req.Type,
		Description:            req.Description,
		HomepageURL:            req.HomepageURL,
		PrivacyPolicyURL:       req.PrivacyPolicyURL,
		TermsURL:               req.TermsURL,
		ContactEmail:           req.ContactEmail,
		Organization:           req.Organization,
		Purpose:                req.Purpose,
		RedirectURIs:           req.RedirectURIs,
		PostLogoutRedirectURIs: req.PostLogoutRedirectURIs,
		Scopes:                 req.Scopes,
		ReviewerID:             reviewerID,
		ReviewerNote:           note,
		DecidedAt:              &now,
		CreatedAppID:           req.CreatedAppID,
		CreatedClientID:        req.CreatedClientID,
		Version:                req.Version + 1,
		CreatedAt:              req.CreatedAt,
		UpdatedAt:              now,
	}

	saved, err := s.repo.Update(ctx, updated)
	if err != nil {
		return Request{}, fmt.Errorf("reject request: %w", err)
	}

	_ = s.repo.AppendEvent(ctx, Event{
		ID:         uuid.New(),
		RequestID:  id,
		ActorType:  "admin",
		ActorID:    reviewerID,
		EventType:  "rejected",
		Note:       note,
		OccurredAt: now,
	})
	_ = s.notifier.NotifyRejected(ctx, saved, note)

	return saved, nil
}

// RequestChanges asks the developer to update their submission.
func (s *Service) RequestChanges(ctx context.Context, id uuid.UUID, reviewerID, note string) (Request, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Request{}, err
	}
	if req.IsTerminal() {
		return Request{}, ErrAlreadyDecided
	}
	if req.Status == StatusUnderReview {
		return Request{}, ErrInvalidTransition
	}

	now := s.now().UTC()
	updated := Request{
		ID:                     req.ID,
		IdentityID:             req.IdentityID,
		Status:                 StatusChangesRequested,
		Name:                   req.Name,
		Slug:                   req.Slug,
		Type:                   req.Type,
		Description:            req.Description,
		HomepageURL:            req.HomepageURL,
		PrivacyPolicyURL:       req.PrivacyPolicyURL,
		TermsURL:               req.TermsURL,
		ContactEmail:           req.ContactEmail,
		Organization:           req.Organization,
		Purpose:                req.Purpose,
		RedirectURIs:           req.RedirectURIs,
		PostLogoutRedirectURIs: req.PostLogoutRedirectURIs,
		Scopes:                 req.Scopes,
		ReviewerID:             reviewerID,
		ReviewerNote:           note,
		DecidedAt:              req.DecidedAt,
		CreatedAppID:           req.CreatedAppID,
		CreatedClientID:        req.CreatedClientID,
		Version:                req.Version + 1,
		CreatedAt:              req.CreatedAt,
		UpdatedAt:              now,
	}

	saved, err := s.repo.Update(ctx, updated)
	if err != nil {
		return Request{}, fmt.Errorf("request changes: %w", err)
	}

	_ = s.repo.AppendEvent(ctx, Event{
		ID:         uuid.New(),
		RequestID:  id,
		ActorType:  "admin",
		ActorID:    reviewerID,
		EventType:  "changes_requested",
		Note:       note,
		OccurredAt: now,
	})
	_ = s.notifier.NotifyChangesRequested(ctx, saved, note)

	return saved, nil
}

func slugifyName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = strings.TrimRight(s[:60], "-")
	}
	if s == "" || !slugPattern.MatchString(s) {
		return fmt.Sprintf("app-%s", uuid.New().String()[:8])
	}
	return s
}
