package middleware

import (
	"errors"

	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/Ans1110/trip-app/pkg/ticket"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// ContextTicketPayload holds the ticket.Payload set by TicketAuth so
	// handlers can read TripID (and any future fields) without re-consuming.
	ContextTicketPayload = "ticket_payload"

	ticketQueryParam = "ticket"
)

// TicketAuth consumes a one-time ticket from the `?ticket=<token>` query param
// and, on success, sets ContextUserID + ContextTicketPayload for downstream
// handlers. It is the browser-friendly cousin of JWTAuth for endpoints that
// can't receive an Authorization header (WebSocket Upgrade, SSE, etc.).
//
// The BFF is the only caller that mints tickets — the browser never sees the
// JWT — so this middleware never falls back to Bearer. All failure modes are
// answered with a bare 401 externally, but logged at distinct levels
// internally so operators can tell "just expired" apart from "tampered" or
// "Redis is on fire".
func TicketAuth(store *ticket.Store, logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	log := logger.With(zap.String("middleware", "ticket_auth"))
	return func(c *gin.Context) {
		token := c.Query(ticketQueryParam)
		if token == "" {
			log.Debug("missing ticket query param",
				zap.String("path", c.Request.URL.Path))
			response.Unauthorized(c)
			c.Abort()
			return
		}

		payload, err := store.Consume(c.Request.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, ticket.ErrInvalid):
				log.Warn("ticket invalid",
					zap.Error(err),
					zap.String("path", c.Request.URL.Path))
			case errors.Is(err, ticket.ErrNotFound):
				log.Info("ticket not found or already used",
					zap.String("path", c.Request.URL.Path))
			default:
				log.Error("ticket consume failed",
					zap.Error(err),
					zap.String("path", c.Request.URL.Path))
			}
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set(ContextUserID, payload.UserID)
		c.Set(ContextTicketPayload, payload)
		c.Next()
	}
}

func GetTicketPayload(c *gin.Context) (ticket.Payload, bool) {
	v, ok := c.Get(ContextTicketPayload)
	if !ok {
		return ticket.Payload{}, false
	}
	p, ok := v.(ticket.Payload)
	if !ok {
		return ticket.Payload{}, false
	}
	return p, true
}
