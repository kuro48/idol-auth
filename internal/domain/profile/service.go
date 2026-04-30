package profile

import (
	"context"
	"fmt"
)

// IdentityProfileClient is the subset of the Kratos admin client used by Service.
type IdentityProfileClient interface {
	GetIdentityProfile(ctx context.Context, identityID string) (Profile, error)
	UpdateIdentityProfile(ctx context.Context, identityID string, input UpdateInput) error
}

// Service implements the profile domain operations backed by a Kratos identity store.
type Service struct {
	client IdentityProfileClient
}

// NewService returns a Service backed by the given client.
func NewService(client IdentityProfileClient) *Service {
	return &Service{client: client}
}

// GetProfile retrieves the shared profile for the given identity.
func (s *Service) GetProfile(ctx context.Context, identityID string) (Profile, error) {
	p, err := s.client.GetIdentityProfile(ctx, identityID)
	if err != nil {
		return Profile{}, fmt.Errorf("get identity profile: %w", err)
	}
	return p, nil
}

// UpdateProfile applies the non-nil fields in input and returns the updated profile.
func (s *Service) UpdateProfile(ctx context.Context, identityID string, input UpdateInput) (Profile, error) {
	if err := s.client.UpdateIdentityProfile(ctx, identityID, input); err != nil {
		return Profile{}, fmt.Errorf("update identity profile: %w", err)
	}
	p, err := s.client.GetIdentityProfile(ctx, identityID)
	if err != nil {
		return Profile{}, fmt.Errorf("reload profile after update: %w", err)
	}
	return p, nil
}
