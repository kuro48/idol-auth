package admin

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/app"
	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

type AppManager interface {
	CreateApp(ctx context.Context, input app.CreateAppInput) (app.App, error)
	ListApps(ctx context.Context) ([]app.App, error)
	CreateOIDCClient(ctx context.Context, appID uuid.UUID, input app.CreateOIDCClientInput) (app.ClientRegistration, error)
	ListOIDCClients(ctx context.Context, appID uuid.UUID) ([]app.OIDCClient, error)
}

type IdentityManager interface {
	SetIdentityRoles(ctx context.Context, identityID string, roles []string) error
	SearchIdentities(ctx context.Context, input SearchIdentitiesInput) ([]Identity, error)
	DisableIdentity(ctx context.Context, input DisableIdentityInput) (Identity, error)
	EnableIdentity(ctx context.Context, input EnableIdentityInput) (Identity, error)
	RevokeIdentitySessions(ctx context.Context, identityID string) error
	DeleteIdentity(ctx context.Context, identityID string) error
}

type Service struct {
	apps      AppManager
	identites IdentityManager
	auditLogs audit.Repository
	now       func() time.Time
}

type SetIdentityRolesInput struct {
	IdentityID string
	Roles      []string
	ActorID    string
}

func NewService(apps AppManager, identities IdentityManager, auditLogs audit.Repository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		apps:      apps,
		identites: identities,
		auditLogs: auditLogs,
		now:       now,
	}
}

func (s *Service) CreateApp(ctx context.Context, input app.CreateAppInput) (app.App, error) {
	return s.apps.CreateApp(ctx, input)
}

func (s *Service) ListApps(ctx context.Context) ([]app.App, error) {
	return s.apps.ListApps(ctx)
}

func (s *Service) CreateOIDCClient(ctx context.Context, appID uuid.UUID, input app.CreateOIDCClientInput) (app.ClientRegistration, error) {
	return s.apps.CreateOIDCClient(ctx, appID, input)
}

func (s *Service) ListOIDCClients(ctx context.Context, appID uuid.UUID) ([]app.OIDCClient, error) {
	return s.apps.ListOIDCClients(ctx, appID)
}

func (s *Service) SetIdentityRoles(ctx context.Context, input SetIdentityRolesInput) ([]string, error) {
	roles := normalizeRoles(input.Roles)
	if err := s.identites.SetIdentityRoles(ctx, strings.TrimSpace(input.IdentityID), roles); err != nil {
		return nil, err
	}
	if s.auditLogs != nil {
		payload, _ := json.Marshal(map[string]any{"roles": roles})
		_ = s.auditLogs.Write(ctx, enrichAuditLog(ctx, audit.Log{
			ID:         uuid.New(),
			EventType:  "identity.roles.updated",
			ActorType:  audit.ActorTypeAdminClient,
			ActorID:    input.ActorID,
			TargetType: audit.TargetTypeUser,
			TargetID:   strings.TrimSpace(input.IdentityID),
			Result:     audit.ResultSuccess,
			Metadata:   payload,
			OccurredAt: s.now().UTC(),
		}))
	}
	return roles, nil
}

func (s *Service) SearchIdentities(ctx context.Context, input SearchIdentitiesInput) ([]Identity, error) {
	return s.identites.SearchIdentities(ctx, input)
}

func (s *Service) DisableIdentity(ctx context.Context, input DisableIdentityInput) (Identity, error) {
	identity, err := s.identites.DisableIdentity(ctx, DisableIdentityInput{
		IdentityID: strings.TrimSpace(input.IdentityID),
		ActorID:    input.ActorID,
	})
	if err != nil {
		return Identity{}, err
	}
	if s.auditLogs != nil {
		_ = s.auditLogs.Write(ctx, enrichAuditLog(ctx, audit.Log{
			ID:         uuid.New(),
			EventType:  "identity.disabled",
			ActorType:  audit.ActorTypeAdminClient,
			ActorID:    input.ActorID,
			TargetType: audit.TargetTypeUser,
			TargetID:   strings.TrimSpace(input.IdentityID),
			Result:     audit.ResultSuccess,
			OccurredAt: s.now().UTC(),
		}))
	}
	return identity, nil
}

func (s *Service) EnableIdentity(ctx context.Context, input EnableIdentityInput) (Identity, error) {
	identity, err := s.identites.EnableIdentity(ctx, EnableIdentityInput{
		IdentityID: strings.TrimSpace(input.IdentityID),
		ActorID:    input.ActorID,
	})
	if err != nil {
		return Identity{}, err
	}
	if s.auditLogs != nil {
		_ = s.auditLogs.Write(ctx, enrichAuditLog(ctx, audit.Log{
			ID:         uuid.New(),
			EventType:  "identity.enabled",
			ActorType:  audit.ActorTypeAdminClient,
			ActorID:    input.ActorID,
			TargetType: audit.TargetTypeUser,
			TargetID:   strings.TrimSpace(input.IdentityID),
			Result:     audit.ResultSuccess,
			OccurredAt: s.now().UTC(),
		}))
	}
	return identity, nil
}

func (s *Service) RevokeIdentitySessions(ctx context.Context, input RevokeIdentitySessionsInput) error {
	if err := s.identites.RevokeIdentitySessions(ctx, strings.TrimSpace(input.IdentityID)); err != nil {
		return err
	}
	if s.auditLogs != nil {
		_ = s.auditLogs.Write(ctx, enrichAuditLog(ctx, audit.Log{
			ID:         uuid.New(),
			EventType:  "identity.sessions.revoked",
			ActorType:  audit.ActorTypeAdminClient,
			ActorID:    input.ActorID,
			TargetType: audit.TargetTypeUser,
			TargetID:   strings.TrimSpace(input.IdentityID),
			Result:     audit.ResultSuccess,
			OccurredAt: s.now().UTC(),
		}))
	}
	return nil
}

func (s *Service) DeleteIdentity(ctx context.Context, input DeleteIdentityInput) error {
	if err := s.identites.DeleteIdentity(ctx, strings.TrimSpace(input.IdentityID)); err != nil {
		return err
	}
	if s.auditLogs != nil {
		_ = s.auditLogs.Write(ctx, enrichAuditLog(ctx, audit.Log{
			ID:         uuid.New(),
			EventType:  "identity.deleted",
			ActorType:  audit.ActorTypeAdminClient,
			ActorID:    input.ActorID,
			TargetType: audit.TargetTypeUser,
			TargetID:   strings.TrimSpace(input.IdentityID),
			Result:     audit.ResultSuccess,
			OccurredAt: s.now().UTC(),
		}))
	}
	return nil
}

func (s *Service) ListAuditLogs(ctx context.Context, input ListAuditLogsInput) ([]AuditLog, error) {
	if s.auditLogs == nil {
		return []AuditLog{}, nil
	}
	return s.auditLogs.List(ctx, audit.ListParams{
		ActorType:  strings.TrimSpace(input.ActorType),
		ActorID:    strings.TrimSpace(input.ActorID),
		TargetType: strings.TrimSpace(input.TargetType),
		TargetID:   strings.TrimSpace(input.TargetID),
		EventType:  strings.TrimSpace(input.EventType),
		Limit:      input.Limit,
		Offset:     input.Offset,
	})
}

func normalizeRoles(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		role := strings.TrimSpace(strings.ToLower(value))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	slices.Sort(out)
	return out
}
