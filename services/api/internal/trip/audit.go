package trip

import (
	"context"

	"github.com/Ans1110/trip-app/internal/audit"
)

// Audit action vocabulary for the trip / room domain.
const (
	AuditTripCreated  audit.Action = "trip_created"
	AuditTripUpdated  audit.Action = "trip_updated"
	AuditTripDeleted  audit.Action = "trip_deleted"

	AuditRoomMemberJoined   audit.Action = "room_member_joined"
	AuditRoomMemberLeft     audit.Action = "room_member_left"
	AuditRoomMemberRemoved  audit.Action = "room_member_removed"
	AuditRoomCodeRegenerated audit.Action = "room_code_regenerated"
)

const (
	AuditResourceTrip = "trip"
	AuditResourceRoom = "room"
)

type auditMeta struct {
	IPAddress string
	UserAgent string
}

type auditMetaCtxKey struct{}

// WithAuditMeta stashes IP / User-Agent on ctx so the service layer can attach
// them to audit rows without depending on gin.
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
