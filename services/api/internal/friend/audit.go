package friend

import (
	"context"

	"github.com/Ans1110/trip-app/internal/auth"
)

// AuditWriter persists friend-domain audit events. Implemented by auth.IRepository.
type AuditWriter interface {
	CreateAuditLog(ctx context.Context, log *auth.AuditLog) error
}

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
