package admin

import "github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"

type IdentityState string

const (
	IdentityStateActive   IdentityState = "active"
	IdentityStateInactive IdentityState = "inactive"
)

type Identity struct {
	ID                    string        `json:"id"`
	SchemaID              string        `json:"schema_id,omitempty"`
	State                 IdentityState `json:"state"`
	Email                 string        `json:"email,omitempty"`
	Phone                 string        `json:"phone,omitempty"`
	PrimaryIdentifierType string        `json:"primary_identifier_type,omitempty"`
	Roles                 []string      `json:"roles,omitempty"`
}

type SearchIdentitiesInput struct {
	CredentialsIdentifier string
	State                 IdentityState
}

type DisableIdentityInput struct {
	IdentityID string
	ActorID    string
}

type EnableIdentityInput struct {
	IdentityID string
	ActorID    string
}

type RevokeIdentitySessionsInput struct {
	IdentityID string
	ActorID    string
}

type DeleteIdentityInput struct {
	IdentityID string
	ActorID    string
}

type ListAuditLogsInput struct {
	ActorType  string
	ActorID    string
	TargetType string
	TargetID   string
	EventType  string
	Limit      int
	Offset     int
}

type AuditLog = audit.Log
