package realtime

import (
	"net/http"
	"time"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/Ans1110/trip-app/pkg/ticket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected, ws *gin.RouterGroup)
}

type Handler struct {
	svc      *Service
	hub      *Hub
	tickets  *ticket.Store
	logger   *zap.Logger
	upgrader websocket.Upgrader
}

type HandlerConfig struct {
	AllowedOrigins []string
	Tickets        *ticket.Store
}

// TicketResponse is the body returned by POST /realtime/trips/{id}/ticket.
// `ExpiresIn` is seconds until the ticket is unusable — clients should open
// the WS immediately and treat the ticket as consumed after one attempt.
type TicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
}

func NewHandler(svc *Service, hub *Hub, logger *zap.Logger, cfg HandlerConfig) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[o] = struct{}{}
	}
	return &Handler{
		svc:     svc,
		hub:     hub,
		tickets: cfg.Tickets,
		logger:  logger.With(zap.String("layer", "realtime.handler")),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				if len(allowed) == 0 {
					return true
				}
				origin := r.Header.Get("Origin")
				if origin == "" {
					return false
				}
				_, ok := allowed[origin]
				return ok
			},
		},
	}
}

func (h *Handler) RegisterRoutes(protected, ws *gin.RouterGroup) {
	protected.POST("/realtime/trips/:id/ticket", h.issueTicket)
	ws.GET("/ws/trips/:id", h.upgrade)
}

// issueTicket godoc
// @Summary  Issue a one-time WebSocket ticket for a trip room
// @Description  Mints a short-lived opaque token that the browser passes as `?ticket=<token>` on the WS Upgrade. Caller must be a member of the trip's room. The ticket is single-use (Redis GETDEL) and expires quickly — clients should open the WS immediately after receiving it.
// @Tags     realtime
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "trip id"
// @Success  200  {object}  response.Response{data=TicketResponse}
// @Failure  400  {object}  response.Response  "invalid trip id"
// @Failure  401  {object}  response.Response
// @Failure  403  {object}  response.Response  "not a room member"
// @Failure  500  {object}  response.Response
// @Router   /realtime/trips/{id}/ticket [post]
func (h *Handler) issueTicket(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid trip id")
		return
	}

	ok, err := h.svc.Authorize(c.Request.Context(), userID, tripID)
	if err != nil {
		h.logger.Error("realtime ticket authz error", zap.Error(err))
		response.InternalError(c, "auth check failed")
		return
	}
	if !ok {
		response.Forbidden(c)
		return
	}

	token, err := h.tickets.Issue(c.Request.Context(), ticket.Payload{
		UserID:   userID,
		TripID:   tripID,
		IssuedAt: time.Now(),
	})
	if err != nil {
		h.logger.Error("realtime ticket issue", zap.Error(err))
		response.InternalError(c, "ticket issue failed")
		return
	}

	response.OK(c, TicketResponse{
		Ticket:    token,
		ExpiresIn: int(h.tickets.TTL().Seconds()),
	})
}

// upgrade godoc
// @Summary  Upgrade to WebSocket for a trip's realtime room
// @Description  HTTP/1.1 → WebSocket upgrade for the trip room. Auth is a one-time ticket (obtained from `POST /realtime/trips/{id}/ticket`) passed as `?ticket=<token>` — no JWT/Authorization header is accepted here. Membership is re-checked on Upgrade so revocations inside the ticket TTL still take effect, and the ticket's embedded TripID must match `{id}` to block cross-trip replay.
// @Tags     realtime
// @Produce  json
// @Param    id      path   string  true  "trip id"
// @Param    ticket  query  string  true  "single-use ticket from /realtime/trips/{id}/ticket"
// @Success  101     "switching protocols (websocket)"
// @Failure  400     {object}  response.Response  "invalid trip id"
// @Failure  401     {object}  response.Response  "missing, invalid, expired, or already-consumed ticket"
// @Failure  403     {object}  response.Response  "ticket trip mismatch or membership revoked"
// @Failure  500     {object}  response.Response
// @Router   /ws/trips/{id} [get]
func (h *Handler) upgrade(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid trip id")
		return
	}

	payload, ok := middleware.GetTicketPayload(c)
	if !ok || payload.TripID != tripID {
		response.Forbidden(c)
		return
	}

	ok, err = h.svc.Authorize(c.Request.Context(), userID, tripID)
	if err != nil {
		h.logger.Error("realtime upgrade authz error", zap.Error(err))
		response.InternalError(c, "auth check failed")
		return
	}
	if !ok {
		response.Forbidden(c)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Warn("ws upgrade failed", zap.Error(err))
		return
	}

	client := newClient(h.hub, h.svc, conn, userID, tripID, h.logger)
	h.svc.Welcome(c.Request.Context(), client)
	client.run(c.Request.Context())
}
