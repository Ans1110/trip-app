package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const HeaderRequestID = "X-Request-ID"

const requestIDKey ctxKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.GetHeader(HeaderRequestID))
		if err != nil {
			id = uuid.New()
		}
		ctx := WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", id.String())
		c.Header(HeaderRequestID, id.String())
		c.Next()
	}
}

func WithRequestID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestIDFromContext(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(requestIDKey).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}
