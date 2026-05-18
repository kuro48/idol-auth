// Package appreg manages the self-service application registration request lifecycle.
package appreg

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending          Status = "pending"
	StatusUnderReview      Status = "under_review"
	StatusApproved         Status = "approved"
	StatusRejected         Status = "rejected"
	StatusWithdrawn        Status = "withdrawn"
	StatusChangesRequested Status = "changes_requested"
)

var (
	ErrRequestNotFound   = errors.New("app registration request not found")
	ErrNotOwner          = errors.New("not the owner of this request")
	ErrAlreadyDecided    = errors.New("request has already been decided")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrDuplicatePending  = errors.New("a pending request with the same name already exists")
	ErrCannotResubmit    = errors.New("request can only be resubmitted when changes are requested")
	ErrCannotWithdraw    = errors.New("request cannot be withdrawn in its current status")
)

// Request represents a developer's application registration request.
type Request struct {
	ID                     uuid.UUID
	IdentityID             string
	Status                 Status
	Name                   string
	Slug                   string
	Type                   string
	Description            string
	HomepageURL            string
	PrivacyPolicyURL       string
	TermsURL               string
	ContactEmail           string
	Organization           string
	Purpose                string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	Scopes                 []string
	ReviewerID             string
	ReviewerNote           string
	DecidedAt              *time.Time
	CreatedAppID           *uuid.UUID
	CreatedClientID        *uuid.UUID
	Version                int
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// Event records a state transition in a request's lifecycle.
type Event struct {
	ID         uuid.UUID
	RequestID  uuid.UUID
	ActorType  string
	ActorID    string
	EventType  string
	Note       string
	OccurredAt time.Time
}

// CanWithdraw reports whether the request may be withdrawn by the developer.
func (r *Request) CanWithdraw() bool {
	switch r.Status {
	case StatusPending, StatusChangesRequested:
		return true
	}
	return false
}

// IsTerminal reports whether the request is in a final state.
func (r *Request) IsTerminal() bool {
	switch r.Status {
	case StatusApproved, StatusRejected, StatusWithdrawn:
		return true
	}
	return false
}
