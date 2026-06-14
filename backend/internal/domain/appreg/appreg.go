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
	ID                     uuid.UUID  `json:"id"`
	IdentityID             string     `json:"identity_id"`
	Status                 Status     `json:"status"`
	Name                   string     `json:"name"`
	Slug                   string     `json:"slug"`
	Type                   string     `json:"type"`
	Description            string     `json:"description"`
	HomepageURL            string     `json:"homepage_url"`
	PrivacyPolicyURL       string     `json:"privacy_policy_url"`
	TermsURL               string     `json:"terms_url"`
	ContactEmail           string     `json:"contact_email"`
	Organization           string     `json:"organization"`
	Purpose                string     `json:"purpose"`
	RedirectURIs           []string   `json:"redirect_uris"`
	PostLogoutRedirectURIs []string   `json:"post_logout_redirect_uris"`
	Scopes                 []string   `json:"scopes"`
	ReviewerID             string     `json:"reviewer_id"`
	ReviewerNote           string     `json:"reviewer_note"`
	DecidedAt              *time.Time `json:"decided_at"`
	CreatedAppID           *uuid.UUID `json:"created_app_id"`
	CreatedClientID        *uuid.UUID `json:"created_client_id"`
	Version                int        `json:"version"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
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
