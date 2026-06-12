package appreg

import "context"

// Notifier sends notifications to developers about their registration request status.
type Notifier interface {
	NotifySubmitted(ctx context.Context, req Request) error
	NotifyApproved(ctx context.Context, req Request) error
	NotifyRejected(ctx context.Context, req Request, reason string) error
	NotifyChangesRequested(ctx context.Context, req Request, reason string) error
}

// NoopNotifier discards all notifications (used when mail is not configured).
type NoopNotifier struct{}

func (NoopNotifier) NotifySubmitted(_ context.Context, _ Request) error              { return nil }
func (NoopNotifier) NotifyApproved(_ context.Context, _ Request) error               { return nil }
func (NoopNotifier) NotifyRejected(_ context.Context, _ Request, _ string) error     { return nil }
func (NoopNotifier) NotifyChangesRequested(_ context.Context, _ Request, _ string) error {
	return nil
}
