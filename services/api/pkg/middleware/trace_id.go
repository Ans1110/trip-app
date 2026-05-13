package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const HeaderTraceID = "X-Trace-ID"

type ctxKey string

const traceIDKey ctxKey = "trace_id"

func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(HeaderTraceID)
		if traceID == "" {
			traceID = uuid.NewString()
		}

		ctx := WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Writer.Header().Set(HeaderTraceID, traceID)

		c.Next()
	}
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, HeaderTraceID, traceID)
}

func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(HeaderTraceID).(string); ok {
		return v
	}
	return ""
}

func FromContext(ctx context.Context) *zap.Logger {
	traceID := GetTraceID(ctx)
	return zap.L().With(zap.String("trace_id", traceID))
}
