package appreg_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/kuro48/idol-auth/internal/domain/appreg"
)

func validSelfServiceInput() appreg.SubmitInput {
	in := validInput()
	in.SelfService = true
	in.Purpose = ""
	return in
}

// --- validation ---

func TestValidation_SelfService_PurposeOptional(t *testing.T) {
	in := validSelfServiceInput()

	if err := in.Validate(); err != nil {
		t.Fatalf("expected purpose to be optional for self-service, got %v", err)
	}
}

func TestValidation_SelfService_PurposeStillBounded(t *testing.T) {
	in := validSelfServiceInput()
	in.Purpose = string(make([]byte, 2001))

	if err := in.Validate(); !errors.Is(err, appreg.ErrInvalidPurpose) {
		t.Fatalf("expected ErrInvalidPurpose for oversized purpose, got %v", err)
	}
}

func TestValidation_SelfService_ScopeAllowlist(t *testing.T) {
	tests := []struct {
		name    string
		scopes  []string
		wantErr error
	}{
		{"allowed scopes pass", []string{"openid", "email", "profile", "offline_access"}, nil},
		{"disallowed scope rejected", []string{"openid", "admin"}, appreg.ErrScopeNotAllowed},
		{"empty scopes pass", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validSelfServiceInput()
			in.Scopes = tt.scopes

			err := in.Validate()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidation_Standard_ScopesUnrestricted(t *testing.T) {
	in := validInput()
	in.Scopes = []string{"openid", "custom_scope"}

	if err := in.Validate(); err != nil {
		t.Fatalf("expected custom scopes allowed in review flow, got %v", err)
	}
}

func TestValidation_Standard_PurposeStillRequired(t *testing.T) {
	in := validInput()
	in.Purpose = ""

	if err := in.Validate(); !errors.Is(err, appreg.ErrInvalidPurpose) {
		t.Fatalf("expected ErrInvalidPurpose, got %v", err)
	}
}

// --- SubmitAutoApproved ---

func TestSubmitAutoApproved_Success(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)
	appID := uuid.New()
	clientID := uuid.New()

	req, err := svc.SubmitAutoApproved(context.Background(), "dev-1", validSelfServiceInput(),
		func(appreg.Request) (uuid.UUID, uuid.UUID, error) {
			return appID, clientID, nil
		})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Status != appreg.StatusApproved {
		t.Errorf("expected status approved, got %s", req.Status)
	}
	if req.CreatedAppID == nil || *req.CreatedAppID != appID {
		t.Errorf("expected CreatedAppID %s, got %v", appID, req.CreatedAppID)
	}
	if req.CreatedClientID == nil || *req.CreatedClientID != clientID {
		t.Errorf("expected CreatedClientID %s, got %v", clientID, req.CreatedClientID)
	}
	if req.ReviewerID != appreg.SelfServiceReviewerID {
		t.Errorf("expected reviewer %q, got %q", appreg.SelfServiceReviewerID, req.ReviewerID)
	}
}

func TestSubmitAutoApproved_ProvisionFailureRejects(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)
	provisionErr := errors.New("hydra down")

	_, err := svc.SubmitAutoApproved(context.Background(), "dev-1", validSelfServiceInput(),
		func(appreg.Request) (uuid.UUID, uuid.UUID, error) {
			return uuid.Nil, uuid.Nil, provisionErr
		})

	if !errors.Is(err, provisionErr) {
		t.Fatalf("expected provision error, got %v", err)
	}
	var stored appreg.Request
	for _, r := range repo.requests {
		stored = r
	}
	if stored.Status != appreg.StatusRejected {
		t.Errorf("expected request rejected after provision failure, got %s", stored.Status)
	}
}

func TestSubmitAutoApproved_ValidationError(t *testing.T) {
	repo := newStubRepo()
	svc := newSvc(repo)
	in := validSelfServiceInput()
	in.Name = ""

	_, err := svc.SubmitAutoApproved(context.Background(), "dev-1", in,
		func(appreg.Request) (uuid.UUID, uuid.UUID, error) {
			return uuid.New(), uuid.New(), nil
		})

	if !errors.Is(err, appreg.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
	if len(repo.requests) != 0 {
		t.Errorf("expected no request stored on validation error, got %d", len(repo.requests))
	}
}
