package account

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

type MembershipStatus string
type DeletionRequestStatus string

const (
	MembershipStatusActive  MembershipStatus = "active"
	MembershipStatusRevoked MembershipStatus = "revoked"

	DeletionStatusScheduled DeletionRequestStatus = "scheduled"
	DeletionStatusCancelled DeletionRequestStatus = "cancelled"
	DeletionStatusCompleted DeletionRequestStatus = "completed"
)

var ErrDeletionRequestNotFound = errors.New("deletion request not found")

type AppMembership struct {
	ID         uuid.UUID        `json:"id"`
	AppID      uuid.UUID        `json:"app_id"`
	AppSlug    string           `json:"app_slug"`
	AppName    string           `json:"app_name"`
	PartyType  app.PartyType    `json:"party_type"`
	IdentityID string           `json:"identity_id"`
	Status     MembershipStatus `json:"status"`
	Profile    json.RawMessage  `json:"profile,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	CreatedBy  string           `json:"created_by,omitempty"`
	UpdatedBy  string           `json:"updated_by,omitempty"`
}

type DeletionRequest struct {
	ID           uuid.UUID             `json:"id"`
	IdentityID   string                `json:"identity_id"`
	Status       DeletionRequestStatus `json:"status"`
	Reason       string                `json:"reason,omitempty"`
	RequestedAt  time.Time             `json:"requested_at"`
	ScheduledFor time.Time             `json:"scheduled_for"`
	CancelledAt  *time.Time            `json:"cancelled_at,omitempty"`
	CompletedAt  *time.Time            `json:"completed_at,omitempty"`
	LastActorID  string                `json:"last_actor_id,omitempty"`
}

type MembershipRepository interface {
	Upsert(ctx context.Context, membership AppMembership) (AppMembership, error)
	ListByIdentity(ctx context.Context, identityID string) ([]AppMembership, error)
	ListByAppID(ctx context.Context, appID uuid.UUID) ([]AppMembership, error)
	UpdateStatus(ctx context.Context, appID uuid.UUID, identityID string, status MembershipStatus, actorID string, now time.Time) error
	UpdateStatusByIdentity(ctx context.Context, identityID string, status MembershipStatus, actorID string, now time.Time) error
}

type DeletionRequestRepository interface {
	UpsertScheduled(ctx context.Context, request DeletionRequest) (DeletionRequest, error)
	GetByIdentity(ctx context.Context, identityID string) (DeletionRequest, error)
	Cancel(ctx context.Context, identityID, actorID string, cancelledAt time.Time) error
	ListDue(ctx context.Context, dueBefore time.Time, limit int) ([]DeletionRequest, error)
	MarkCompleted(ctx context.Context, identityID, actorID string, completedAt time.Time) error
}

type AppDirectory interface {
	GetByHydraClientID(ctx context.Context, hydraClientID string) (app.App, error)
	GetByID(ctx context.Context, appID uuid.UUID) (app.App, error)
}

type IdentityLifecycle interface {
	RevokeIdentitySessions(ctx context.Context, identityID string) error
	DeleteIdentity(ctx context.Context, identityID string) error
}

type AppTokenResolver interface {
	ResolveAppByToken(ctx context.Context, rawToken string) (app.App, error)
}

type Service struct {
	memberships MembershipRepository
	deletions   DeletionRequestRepository
	apps        AppDirectory
	identities  IdentityLifecycle
	tokens      AppTokenResolver
	auditLogs   audit.Repository
	now         func() time.Time
	gracePeriod time.Duration
}

func NewService(memberships MembershipRepository, deletions DeletionRequestRepository, apps AppDirectory, identities IdentityLifecycle, tokens AppTokenResolver, auditLogs audit.Repository, now func() time.Time, gracePeriod time.Duration) *Service {
	if now == nil {
		now = time.Now
	}
	if gracePeriod <= 0 {
		gracePeriod = 7 * 24 * time.Hour
	}
	return &Service{
		memberships: memberships,
		deletions:   deletions,
		apps:        apps,
		identities:  identities,
		tokens:      tokens,
		auditLogs:   auditLogs,
		now:         now,
		gracePeriod: gracePeriod,
	}
}

func (s *Service) EnsureMembershipForHydraClient(ctx context.Context, hydraClientID, identityID, actorID string) error {
	if s.memberships == nil || s.apps == nil {
		return nil
	}
	hydraClientID = strings.TrimSpace(hydraClientID)
	identityID = strings.TrimSpace(identityID)
	if hydraClientID == "" || identityID == "" {
		return nil
	}
	appEntity, err := s.apps.GetByHydraClientID(ctx, hydraClientID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	_, err = s.memberships.Upsert(ctx, AppMembership{
		ID:         uuid.New(),
		AppID:      appEntity.ID,
		AppSlug:    appEntity.Slug,
		AppName:    appEntity.Name,
		PartyType:  appEntity.PartyType,
		IdentityID: identityID,
		Status:     MembershipStatusActive,
		Profile:    json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  actorID,
		UpdatedBy:  actorID,
	})
	return err
}

func (s *Service) ResolveAppByToken(ctx context.Context, rawToken string) (app.App, error) {
	if s.tokens == nil {
		return app.App{}, errors.New("app token resolver unavailable")
	}
	return s.tokens.ResolveAppByToken(ctx, rawToken)
}

func (s *Service) ListMembershipsForIdentity(ctx context.Context, identityID string) ([]AppMembership, error) {
	return s.memberships.ListByIdentity(ctx, strings.TrimSpace(identityID))
}

func (s *Service) ListMembershipsForApp(ctx context.Context, appID uuid.UUID) ([]AppMembership, error) {
	return s.memberships.ListByAppID(ctx, appID)
}

func (s *Service) DisconnectIdentityFromApp(ctx context.Context, identityID string, appID uuid.UUID, actorID string) error {
	now := s.now().UTC()
	if err := s.memberships.UpdateStatus(ctx, appID, strings.TrimSpace(identityID), MembershipStatusRevoked, actorID, now); err != nil {
		return err
	}
	s.writeAudit(ctx, audit.Log{
		ID:         uuid.New(),
		EventType:  "app.membership.revoked",
		ActorType:  audit.ActorTypeIdentity,
		ActorID:    actorID,
		TargetType: audit.TargetTypeAppMembership,
		TargetID:   appID.String() + ":" + strings.TrimSpace(identityID),
		Result:     audit.ResultSuccess,
		OccurredAt: now,
	})
	return nil
}

func (s *Service) RevokeAppUser(ctx context.Context, appID uuid.UUID, identityID, actorID string) error {
	now := s.now().UTC()
	if err := s.memberships.UpdateStatus(ctx, appID, strings.TrimSpace(identityID), MembershipStatusRevoked, actorID, now); err != nil {
		return err
	}
	s.writeAudit(ctx, audit.Log{
		ID:         uuid.New(),
		EventType:  "app.membership.revoked",
		ActorType:  audit.ActorTypeAppClient,
		ActorID:    actorID,
		TargetType: audit.TargetTypeAppMembership,
		TargetID:   appID.String() + ":" + strings.TrimSpace(identityID),
		Result:     audit.ResultSuccess,
		OccurredAt: now,
	})
	return nil
}

func (s *Service) ScheduleDeletion(ctx context.Context, identityID, actorID, reason string) (DeletionRequest, error) {
	now := s.now().UTC()
	request := DeletionRequest{
		ID:           uuid.New(),
		IdentityID:   strings.TrimSpace(identityID),
		Status:       DeletionStatusScheduled,
		Reason:       strings.TrimSpace(reason),
		RequestedAt:  now,
		ScheduledFor: now.Add(s.gracePeriod),
		LastActorID:  actorID,
	}
	created, err := s.deletions.UpsertScheduled(ctx, request)
	if err != nil {
		return DeletionRequest{}, err
	}
	s.writeAudit(ctx, audit.Log{
		ID:         uuid.New(),
		EventType:  "identity.deletion.scheduled",
		ActorType:  audit.ActorTypeIdentity,
		ActorID:    actorID,
		TargetType: audit.TargetTypeUser,
		TargetID:   request.IdentityID,
		Result:     audit.ResultSuccess,
		OccurredAt: now,
	})
	return created, nil
}

func (s *Service) CancelDeletion(ctx context.Context, identityID, actorID string) error {
	now := s.now().UTC()
	if err := s.deletions.Cancel(ctx, strings.TrimSpace(identityID), actorID, now); err != nil {
		return err
	}
	s.writeAudit(ctx, audit.Log{
		ID:         uuid.New(),
		EventType:  "identity.deletion.cancelled",
		ActorType:  audit.ActorTypeIdentity,
		ActorID:    actorID,
		TargetType: audit.TargetTypeUser,
		TargetID:   strings.TrimSpace(identityID),
		Result:     audit.ResultSuccess,
		OccurredAt: now,
	})
	return nil
}

func (s *Service) GetDeletionRequest(ctx context.Context, identityID string) (*DeletionRequest, error) {
	request, err := s.deletions.GetByIdentity(ctx, strings.TrimSpace(identityID))
	if err != nil {
		if errors.Is(err, ErrDeletionRequestNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &request, nil
}

func (s *Service) ProcessDueDeletionRequests(ctx context.Context, limit int) error {
	if s.deletions == nil || s.identities == nil || s.memberships == nil {
		return nil
	}
	if limit <= 0 {
		limit = 50
	}
	now := s.now().UTC()
	requests, err := s.deletions.ListDue(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, request := range requests {
		identityID := strings.TrimSpace(request.IdentityID)
		if identityID == "" {
			continue
		}
		_ = s.identities.RevokeIdentitySessions(ctx, identityID)
		if err := s.memberships.UpdateStatusByIdentity(ctx, identityID, MembershipStatusRevoked, "system", now); err != nil {
			return err
		}
		if err := s.identities.DeleteIdentity(ctx, identityID); err != nil {
			return err
		}
		if err := s.deletions.MarkCompleted(ctx, identityID, "system", now); err != nil {
			return err
		}
		s.writeAudit(ctx, audit.Log{
			ID:         uuid.New(),
			EventType:  "identity.deletion.completed",
			ActorType:  audit.ActorTypeSystem,
			ActorID:    "system",
			TargetType: audit.TargetTypeUser,
			TargetID:   identityID,
			Result:     audit.ResultSuccess,
			OccurredAt: now,
		})
	}
	return nil
}

func (s *Service) writeAudit(ctx context.Context, entry audit.Log) {
	if s.auditLogs == nil {
		return
	}
	_ = s.auditLogs.Write(ctx, entry)
}
