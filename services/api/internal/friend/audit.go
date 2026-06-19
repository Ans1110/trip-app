package friend

import (
	"context"

	"github.com/Ans1110/trip-app/internal/audit"
)

// Friend-domain audit action vocabulary. The Action type and the persistence
// layer live in internal/audit.
const (
	AuditFriendRequestSent      audit.Action = "friend_request_sent"
	AuditFriendRequestAccepted  audit.Action = "friend_request_accepted"
	AuditFriendRequestDeclined  audit.Action = "friend_request_declined"
	AuditFriendRequestCancelled audit.Action = "friend_request_cancelled"
	AuditFriendRemoved          audit.Action = "friend_removed"
	AuditFriendBlocked          audit.Action = "friend_blocked"
	AuditFriendUnblocked        audit.Action = "friend_unblocked"
	AuditFriendInviteCreated    audit.Action = "friend_invite_created"
	AuditFriendInviteRevoked    audit.Action = "friend_invite_revoked"
	AuditFriendInviteAccepted   audit.Action = "friend_invite_accepted"
)

// Resource types tagged on audit log rows so operators can filter by domain.
const (
	AuditResourceFriendship    = "friendship"
	AuditResourceFriendRequest = "friend_request"
	AuditResourceFriendBlock   = "friend_block"
	AuditResourceFriendInvite  = "friend_invite"
)

// auditMeta carries per-request fields that the handler captures off the gin
// context (IP, User-Agent) so the service layer can record them on audit logs
// without depending on gin.
type auditMeta struct {
	IPAddress string
	UserAgent string
}

type auditMetaCtxKey struct{}

// WithAuditMeta returns a context that carries IP and User-Agent for audit
// logging. Handlers wrap the request context before invoking the service.
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
