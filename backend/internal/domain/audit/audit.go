// Package audit defines the AuditLog entity and the write-only repository interface.
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ActorType string
type TargetType string
type Result string

const (
	ActorTypeAdminClient ActorType = "admin_client"
	ActorTypeAppClient   ActorType = "app_client"
	ActorTypeIdentity    ActorType = "identity"
	ActorTypeSystem      ActorType = "system"

	TargetTypeApp           TargetType = "app"
	TargetTypeClient        TargetType = "client"
	TargetTypeUser          TargetType = "user"
	TargetTypeAppMembership TargetType = "app_membership"

	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
)

// Log is a single immutable audit event.
type Log struct {
	ID         uuid.UUID       `json:"id"`
	EventType  string          `json:"event_type"`
	ActorType  ActorType       `json:"actor_type"`
	ActorID    string          `json:"actor_id"`
	TargetType TargetType      `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Result     Result          `json:"result"`
	IPAddress  string          `json:"ip_address"`
	UserAgent  string          `json:"user_agent"`
	RequestID  string          `json:"request_id"`
	Metadata   json.RawMessage `json:"metadata"`
	OccurredAt time.Time       `json:"occurred_at"`
}

// Repository is the write-only interface for persisting audit events.
type Repository interface {
	Write(ctx context.Context, entry Log) error
	List(ctx context.Context, params ListParams) ([]Log, error)
}

// ListParams holds query parameters for fetching audit logs.
type ListParams struct {
	ActorType  string
	ActorID    string
	TargetType string
	TargetID   string
	EventType  string
	Limit      int
	Offset     int
}
