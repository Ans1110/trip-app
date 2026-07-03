package calendar

import (
	"context"

	"github.com/Ans1110/trip-app/internal/audit"
)

const (
	AuditEventCreated audit.Action = "calendar_event_created"
	AuditEventUpdated audit.Action = "calendar_event_updated"
	AuditEventDeleted audit.Action = "calendar_event_deleted"
)

const AuditResourceEvent = "calendar_event"

type auditMeta struct {
	IPAddress string
	UserAgent string
}

type auditMetaCtxKey struct{}

func WithAuditMeta(ctx context.Context, ipAddress, userAgent string) context.Context {
	if ipAddress == "" && userAgent == "" {
		return ctx
	}
	return context.WithValue(ctx, auditMetaCtxKey{}, auditMeta{
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

func auditMetaFromCtx(ctx context.Context) auditMeta {
	if v, ok := ctx.Value(auditMetaCtxKey{}).(auditMeta); ok {
		return v
	}
	return auditMeta{}
}
