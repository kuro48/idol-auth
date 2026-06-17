// Package loginhistory tracks observed authenticated sessions per identity
// so the user can review where and when their account has been signed in.
package loginhistory

import (
	"context"
	"strings"
	"time"
)

// Event captures a single observed Kratos session for an identity.
type Event struct {
	ID              string    `json:"id"`
	IdentityID      string    `json:"identity_id"`
	SessionID       string    `json:"session_id"`
	AuthenticatedAt time.Time `json:"authenticated_at"`
	IssuedAt        time.Time `json:"issued_at"`
	AAL             string    `json:"aal"`
	Methods         []string  `json:"methods"`
	IPAddress       string    `json:"ip_address,omitempty"`
	UserAgent       string    `json:"user_agent,omitempty"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// Repository persists login events.
type Repository interface {
	// Record inserts an event, deduplicating on session_id.
	// Returns nil if a row with the same session_id already exists.
	Record(ctx context.Context, evt Event) error
	// ListByIdentity returns events for identityID ordered by AuthenticatedAt DESC.
	ListByIdentity(ctx context.Context, identityID string, limit int) ([]Event, error)
}

// Service is a thin wrapper that trims input and calls the repository.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Record persists evt, ignoring empty or duplicate sessions silently.
func (s *Service) Record(ctx context.Context, evt Event) error {
	evt.IdentityID = strings.TrimSpace(evt.IdentityID)
	evt.SessionID = strings.TrimSpace(evt.SessionID)
	if evt.IdentityID == "" || evt.SessionID == "" {
		return nil
	}
	if evt.RecordedAt.IsZero() {
		evt.RecordedAt = time.Now().UTC()
	}
	return s.repo.Record(ctx, evt)
}

// List returns up to limit recent events for identityID.
func (s *Service) List(ctx context.Context, identityID string, limit int) ([]Event, error) {
	id := strings.TrimSpace(identityID)
	if id == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.ListByIdentity(ctx, id, limit)
}
