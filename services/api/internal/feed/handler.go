package feed

import (
	"strconv"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)
	ListFeed(c *gin.Context)
}

type Handler struct {
	svc    IService
	logger *zap.Logger
}

func NewHandler(svc IService, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger.With(zap.String("layer", "feed.handler"))}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	protected.GET("/feed", h.ListFeed)
}

// ListFeed godoc
// @Summary      Get the caller's personalized home feed (ranked, keyset paginated)
// @Description  Global discovery feed. All public posts are eligible; posts from followed authors get a ranking boost.
// @Tags         feed
// @Security     BearerAuth
// @Produce      json
// @Param        cursor  query     string  false  "opaque cursor from previous page"
// @Param        limit   query     int     false  "page size (default 20, max 50)"
// @Success      200     {object}  response.Response{data=ListFeedResponse}
// @Router       /feed [get]
func (h *Handler) ListFeed(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	cursor, err := DecodeCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.ListFeed(c.Request.Context(), uid, cursor, parseLimit(c))
	if err != nil {
		h.logger.Error("feed list: internal error", zap.Error(err))
		response.InternalError(c, "internal error")
		return
	}
	response.OK(c, out)
}

func parseLimit(c *gin.Context) int {
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return 0
}
