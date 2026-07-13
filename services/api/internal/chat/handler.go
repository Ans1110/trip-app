package chat

import (
	"net/http"
	"strconv"
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
		logger:  logger.With(zap.String("layer", "chat.handler")),
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
	g := protected.Group("/chat")
	{
		g.GET("/rooms", h.listRooms)
		g.POST("/rooms/dm", h.ensureDM)
		g.POST("/trips/:trip_id/room", h.ensureTripRoom)
		g.GET("/rooms/:id/messages", h.listMessages)
		g.GET("/rooms/:id/receipts", h.listReceipts)
		g.POST("/rooms/:id/read", h.markRead)
		g.POST("/rooms/:id/ticket", h.issueTicket)
		g.DELETE("/rooms/:id", h.hideRoom)
	}
	ws.GET("/ws/chat/rooms/:id", h.upgrade)
}

// ensureTripRoom godoc
// @Summary  Ensure the group chat room for a trip exists (idempotent)
// @Tags     chat
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path      string  true  "trip id"
// @Success  200      {object}  response.Response{data=RoomDTO}
// @Router   /chat/trips/{trip_id}/room [post]
func (h *Handler) ensureTripRoom(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	tripID, err := uuid.Parse(c.Param("trip_id"))
	if err != nil {
		response.BadRequest(c, "invalid trip id")
		return
	}
	room, err := h.svc.EnsureTripRoom(c.Request.Context(), tripID, uid)
	if err != nil {
		if err == ErrForbidden {
			response.Forbidden(c)
			return
		}
		h.logger.Error("chat: ensure trip room", zap.Error(err))
		response.InternalError(c, "ensure trip room failed")
		return
	}
	response.OK(c, RoomDTO{
		ID:        room.ID,
		TripID:    room.TripID,
		Name:      room.Name,
		Type:      room.Type,
		CreatedAt: room.CreatedAt,
	})
}

// listRooms godoc
// @Summary  List the caller's chat rooms
// @Tags     chat
// @Security BearerAuth
// @Produce  json
// @Success  200  {object}  response.Response{data=[]RoomDTO}
// @Router   /chat/rooms [get]
func (h *Handler) listRooms(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	rooms, err := h.svc.ListRooms(c.Request.Context(), uid)
	if err != nil {
		h.logger.Error("chat: list rooms", zap.Error(err))
		response.InternalError(c, "list rooms failed")
		return
	}
	response.OK(c, rooms)
}

// ensureDM godoc
// @Summary  Find-or-create a 1-1 DM room with peer
// @Tags     chat
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      EnsureDMRequest  true  "peer id"
// @Success  200   {object}  response.Response{data=RoomDTO}
// @Router   /chat/rooms/dm [post]
func (h *Handler) ensureDM(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	var req EnsureDMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	peer, err := uuid.Parse(req.PeerID)
	if err != nil {
		response.BadRequest(c, "invalid peer id")
		return
	}
	dto, err := h.svc.EnsureDM(c.Request.Context(), uid, peer)
	if err != nil {
		h.logger.Error("chat: ensure dm", zap.Error(err))
		response.InternalError(c, "ensure dm failed")
		return
	}
	response.OK(c, dto)
}

// listMessages godoc
// @Summary  Fetch message history for a room (newest-first, keyset paginated)
// @Tags     chat
// @Security BearerAuth
// @Produce  json
// @Param    id      path   string  true  "room id"
// @Param    before  query  string  false "RFC3339 timestamp — fetch messages strictly older than this"
// @Param    limit   query  int     false "page size (default 50, max 200)"
// @Success  200     {object}  response.Response{data=[]MessageDTO}
// @Router   /chat/rooms/{id}/messages [get]
func (h *Handler) listMessages(c *gin.Context) {
	uid, roomID, ok := h.requireMember(c)
	if !ok {
		return
	}
	_ = uid
	limit := 0
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}
	var before *time.Time
	if s := c.Query("before"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			response.BadRequest(c, "invalid before timestamp")
			return
		}
		before = &t
	}
	msgs, err := h.svc.ListMessages(c.Request.Context(), roomID, before, limit)
	if err != nil {
		h.logger.Error("chat: list messages", zap.Error(err))
		response.InternalError(c, "list messages failed")
		return
	}
	response.OK(c, msgs)
}

// listReceipts godoc
// @Summary  List read receipts for a room
// @Tags     chat
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "room id"
// @Success  200  {object}  response.Response{data=[]ReadReceiptDTO}
// @Router   /chat/rooms/{id}/receipts [get]
func (h *Handler) listReceipts(c *gin.Context) {
	_, roomID, ok := h.requireMember(c)
	if !ok {
		return
	}
	rows, err := h.svc.ListReadReceipts(c.Request.Context(), roomID)
	if err != nil {
		h.logger.Error("chat: list receipts", zap.Error(err))
		response.InternalError(c, "list receipts failed")
		return
	}
	response.OK(c, rows)
}

// markRead godoc
// @Summary  Advance the caller's read pointer to message id
// @Tags     chat
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path   string        true  "room id"
// @Param    body  body   MarkReadData  true  "last_read_id"
// @Success  200   {object}  response.Response{data=ReadReceiptDTO}
// @Router   /chat/rooms/{id}/read [post]
func (h *Handler) markRead(c *gin.Context) {
	uid, roomID, ok := h.requireMember(c)
	if !ok {
		return
	}
	var req MarkReadData
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.LastReadID == uuid.Nil {
		response.BadRequest(c, "invalid last_read_id")
		return
	}
	dto, err := h.svc.MarkRead(c.Request.Context(), roomID, uid, req.LastReadID)
	if err != nil {
		h.logger.Error("chat: mark read", zap.Error(err))
		response.InternalError(c, "mark read failed")
		return
	}
	response.OK(c, dto)
}

// issueTicket godoc
// @Summary  Issue a one-time WebSocket ticket for a chat room
// @Tags     chat
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "room id"
// @Success  200  {object}  response.Response{data=TicketResponse}
// @Router   /chat/rooms/{id}/ticket [post]
func (h *Handler) issueTicket(c *gin.Context) {
	uid, roomID, ok := h.requireMember(c)
	if !ok {
		return
	}
	token, err := h.tickets.Issue(c.Request.Context(), ticket.Payload{
		UserID:   uid,
		RoomID:   roomID,
		IssuedAt: time.Now(),
	})
	if err != nil {
		h.logger.Error("chat: ticket issue", zap.Error(err))
		response.InternalError(c, "ticket issue failed")
		return
	}
	response.OK(c, TicketResponse{
		Ticket:    token,
		ExpiresIn: int(h.tickets.TTL().Seconds()),
	})
}

// hideRoom godoc
// @Summary  Soft-hide a chat from the caller's list (DM only)
// @Description Messages and the room itself are preserved. Sending a new
// @Description message or re-opening the DM (EnsureDM) restores it.
// @Tags     chat
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "room id"
// @Success  200  {object}  response.Response
// @Router   /chat/rooms/{id} [delete]
func (h *Handler) hideRoom(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid room id")
		return
	}
	if err := h.svc.HideRoom(c.Request.Context(), roomID, uid); err != nil {
		switch err {
		case ErrForbidden:
			response.Forbidden(c)
		case ErrHideNotAllowed:
			response.BadRequest(c, "room type does not support hide")
		case ErrRoomNotFound:
			response.NotFound(c, "room not found")
		default:
			h.logger.Error("chat: hide room", zap.Error(err))
			response.InternalError(c, "hide room failed")
		}
		return
	}
	response.OK(c, nil)
}

// upgrade godoc
// @Summary  Upgrade to WebSocket for a chat room
// @Tags     chat
// @Produce  json
// @Param    id      path   string  true  "room id"
// @Param    ticket  query  string  true  "single-use ticket from /chat/rooms/{id}/ticket"
// @Success  101     "switching protocols (websocket)"
// @Router   /ws/chat/rooms/{id} [get]
func (h *Handler) upgrade(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid room id")
		return
	}
	payload, ok := middleware.GetTicketPayload(c)
	if !ok || payload.RoomID != roomID {
		response.Forbidden(c)
		return
	}

	ok, err = h.svc.Authorize(c.Request.Context(), userID, roomID)
	if err != nil {
		h.logger.Error("chat: upgrade authz", zap.Error(err))
		response.InternalError(c, "auth check failed")
		return
	}
	if !ok {
		response.Forbidden(c)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Warn("chat ws upgrade failed", zap.Error(err))
		return
	}

	client := newClient(h.hub, h.svc, conn, userID, roomID, h.logger)
	h.svc.Welcome(c.Request.Context(), client)
	client.run(c.Request.Context())
}

// requireMember pulls user + room id from the request, verifies membership,
// and short-circuits with 401/400/403 on failure.
func (h *Handler) requireMember(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return uuid.Nil, uuid.Nil, false
	}
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid room id")
		return uuid.Nil, uuid.Nil, false
	}
	member, err := h.svc.Authorize(c.Request.Context(), uid, roomID)
	if err != nil {
		h.logger.Error("chat: authz check", zap.Error(err))
		response.InternalError(c, "auth check failed")
		return uuid.Nil, uuid.Nil, false
	}
	if !member {
		response.Forbidden(c)
		return uuid.Nil, uuid.Nil, false
	}
	return uid, roomID, true
}
