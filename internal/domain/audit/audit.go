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
	ActorTypeSystem      ActorType = "system"

	TargetTypeApp    TargetType = "app"
	TargetTypeClient TargetType = "client"
	TargetTypeUser   TargetType = "user"

	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
)

// Log is a single immutable audit event.
type Log struct {
	ID         uuid.UUID
	EventType  string
	ActorType  ActorType
	ActorID    string
	TargetType TargetType
	TargetID   string
	Result     Result
	IPAddress  string
	UserAgent  string
	RequestID  string
	Metadata   json.RawMessage
	OccurredAt time.Time
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
