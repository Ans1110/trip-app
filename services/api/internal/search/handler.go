package search

import (
	"errors"
	"strconv"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)
	SearchPosts(c *gin.Context)
}

type Handler struct {
	svc    IService
	logger *zap.Logger
}

func NewHandler(svc IService, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger.With(zap.String("layer", "search.handler"))}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	s := protected.Group("/search")
	{
		s.GET("/posts", h.SearchPosts)
	}
}

// SearchPosts godoc
// @Summary      Search travel posts (Postgres FTS, keyset paginated)
// @Tags         search
// @Security     BearerAuth
// @Produce      json
// @Param        q       query     string  true   "search query"
// @Param        cursor  query     string  false  "opaque cursor from previous page"
// @Param        limit   query     int     false  "page size (default 20, max 50)"
// @Success      200     {object}  response.Response{data=SearchPostsResponse}
// @Failure      400     {object}  response.Response
// @Router       /search/posts [get]
func (h *Handler) SearchPosts(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	q := c.Query("q")
	cursor, err := DecodeCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.SearchPosts(c.Request.Context(), viewerID, q, cursor, parseLimit(c))
	if err != nil {
		if errors.Is(err, ErrQueryRequired) {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error("search posts: internal error", zap.Error(err))
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
