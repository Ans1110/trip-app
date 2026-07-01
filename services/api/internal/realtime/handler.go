package realtime

import (
	"net/http"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(group *gin.RouterGroup)
}

type Handler struct {
	svc      *Service
	hub      *Hub
	logger   *zap.Logger
	upgrader websocket.Upgrader
}

type HandlerConfig struct {
	AllowedOrigins []string
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
		svc:    svc,
		hub:    hub,
		logger: logger.With(zap.String("layer", "realtime.handler")),
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

func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/ws/trips/:id", h.upgrade)
}

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

	ok, err := h.svc.Authorize(c.Request.Context(), userID, tripID)
	if err != nil {
		h.logger.Error("realtime authz error", zap.Error(err))
		response.InternalError(c, "auth check failed")
		return
	}
	if !ok {
		response.Forbidden(c)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade has already written the error response on failure.
		h.logger.Warn("ws upgrade failed", zap.Error(err))
		return
	}

	client := newClient(h.hub, h.svc, conn, userID, tripID, h.logger)
	h.svc.Welcome(c.Request.Context(), client)
	client.run(c.Request.Context())
}
