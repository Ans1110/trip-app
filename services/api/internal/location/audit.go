package location

import (
	"context"

	"github.com/Ans1110/trip-app/internal/audit"
)

const (
	AuditSavedPlaceCreated audit.Action = "location_saved_place_created"
	AuditSavedPlaceUpdated audit.Action = "location_saved_place_updated"
	AuditSavedPlaceDeleted audit.Action = "location_saved_place_deleted"
)

const AuditResourceSavedPlace = "location_saved_place"

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
