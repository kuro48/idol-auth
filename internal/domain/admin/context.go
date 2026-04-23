package admin

import (
	"context"

	"github.com/ryunosukekurokawa/idol-auth/internal/domain/audit"
)

type auditContextKey string

const requestMetadataContextKey auditContextKey = "admin_request_metadata"

type RequestMetadata struct {
	IPAddress string
	UserAgent string
	RequestID string
}

func WithRequestMetadata(ctx context.Context, metadata RequestMetadata) context.Context {
	return context.WithValue(ctx, requestMetadataContextKey, metadata)
}

func enrichAuditLog(ctx context.Context, entry audit.Log) audit.Log {
	metadata, ok := ctx.Value(requestMetadataContextKey).(RequestMetadata)
	if !ok {
		return entry
	}
	entry.IPAddress = metadata.IPAddress
	entry.UserAgent = metadata.UserAgent
	entry.RequestID = metadata.RequestID
	return entry
}
