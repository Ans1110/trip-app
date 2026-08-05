package profile

import (
	"context"
	"errors"
	"strconv"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)

	GetMyProfile(c *gin.Context)
	GetProfileByUsername(c *gin.Context)
	UpdateMyProfile(c *gin.Context)

	Follow(c *gin.Context)
	Unfollow(c *gin.Context)

	ListFollowers(c *gin.Context)
	ListFollowing(c *gin.Context)

	Recommendations(c *gin.Context)
}

type Handler struct {
	svc    IService
	logger *zap.Logger
}

func NewHandler(svc IService, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		svc:    svc,
		logger: logger.With(zap.String("layer", "profile.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	profiles := protected.Group("/profiles")
	{
		profiles.GET("/me", h.GetMyProfile)
		profiles.PATCH("/me", h.UpdateMyProfile)
		profiles.GET("/recommendations", h.Recommendations)
		profiles.GET("/by-username/:username", h.GetProfileByUsername)

		profiles.POST("/:user_id/follow", h.Follow)
		profiles.DELETE("/:user_id/follow", h.Unfollow)
		profiles.GET("/:user_id/followers", h.ListFollowers)
		profiles.GET("/:user_id/following", h.ListFollowing)
	}
}

// GetMyProfile godoc
// @Summary      Get the caller's profile
// @Tags         profiles
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=ProfileResponse}
// @Failure      401  {object}  response.Response
// @Router       /profiles/me [get]
func (h *Handler) GetMyProfile(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	p, err := h.svc.GetMyProfile(c.Request.Context(), uid)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, p)
}

// GetProfileByUsername godoc
// @Summary      Get a profile by @username
// @Tags         profiles
// @Security     BearerAuth
// @Produce      json
// @Param        username  path      string  true  "username"
// @Success      200       {object}  response.Response{data=ProfileResponse}
// @Failure      404       {object}  response.Response
// @Router       /profiles/by-username/{username} [get]
func (h *Handler) GetProfileByUsername(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	username := c.Param("username")
	if username == "" {
		response.BadRequest(c, "username is required")
		return
	}
	p, err := h.svc.GetProfileByUsername(c.Request.Context(), viewerID, username)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, p)
}

// UpdateMyProfile godoc
// @Summary      Update the caller's profile
// @Tags         profiles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      UpdateProfilePayload  true  "fields to patch"
// @Success      200   {object}  response.Response{data=ProfileResponse}
// @Failure      400   {object}  response.Response
// @Failure      409   {object}  response.Response
// @Router       /profiles/me [patch]
func (h *Handler) UpdateMyProfile(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var payload UpdateProfilePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.UpdateMyProfile(c.Request.Context(), uid, payload)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, p)
}

// Follow godoc
// @Summary      Follow a user
// @Tags         profiles
// @Security     BearerAuth
// @Produce      json
// @Param        user_id  path      string  true  "target user id"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Router       /profiles/{user_id}/follow [post]
func (h *Handler) Follow(c *gin.Context) {
	uid, targetID, ok := h.requireCallerAndTarget(c)
	if !ok {
		return
	}
	if err := h.svc.Follow(c.Request.Context(), uid, targetID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// Unfollow godoc
// @Summary      Unfollow a user
// @Tags         profiles
// @Security     BearerAuth
// @Produce      json
// @Param        user_id  path      string  true  "target user id"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Router       /profiles/{user_id}/follow [delete]
func (h *Handler) Unfollow(c *gin.Context) {
	uid, targetID, ok := h.requireCallerAndTarget(c)
	if !ok {
		return
	}
	if err := h.svc.Unfollow(c.Request.Context(), uid, targetID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListFollowers godoc
// @Summary      List a user's followers (keyset paginated)
// @Tags         profiles
// @Security     BearerAuth
// @Produce      json
// @Param        user_id  path      string  true   "target user id"
// @Param        cursor   query     string  false  "opaque cursor from previous page"
// @Param        limit    query     int     false  "page size (default 20, max 100)"
// @Success      200      {object}  response.Response{data=ListUsersResponse}
// @Router       /profiles/{user_id}/followers [get]
func (h *Handler) ListFollowers(c *gin.Context) {
	h.listFollowSide(c, h.svc.ListFollowers)
}

// ListFollowing godoc
// @Summary      List users a user is following (keyset paginated)
// @Tags         profiles
// @Security     BearerAuth
// @Produce      json
// @Param        user_id  path      string  true   "target user id"
// @Param        cursor   query     string  false  "opaque cursor from previous page"
// @Param        limit    query     int     false  "page size (default 20, max 100)"
// @Success      200      {object}  response.Response{data=ListUsersResponse}
// @Router       /profiles/{user_id}/following [get]
func (h *Handler) ListFollowing(c *gin.Context) {
	h.listFollowSide(c, h.svc.ListFollowing)
}

type followSideFn func(ctx context.Context, viewerID, targetID uuid.UUID, cursor *FollowCursor, limit int) (ListUsersResponse, error)

func (h *Handler) listFollowSide(c *gin.Context, fn followSideFn) {
	viewerID := middleware.GetUserID(c)
	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	cursor, err := DecodeFollowCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := fn(c.Request.Context(), viewerID, targetID, cursor, parseLimit(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// Recommendations godoc
// @Summary      Get recommended users to follow
// @Tags         profiles
// @Security     BearerAuth
// @Produce      json
// @Param        limit  query     int  false  "max results (default 10, cap 50)"
// @Success      200    {object}  response.Response{data=RecommendationResponse}
// @Router       /profiles/recommendations [get]
func (h *Handler) Recommendations(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	recs, err := h.svc.Recommendations(c.Request.Context(), uid, parseLimit(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, recs)
}

// requireCallerAndTarget resolves the JWT caller plus the :user_id path param.
func (h *Handler) requireCallerAndTarget(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return uuid.Nil, uuid.Nil, false
	}
	return uid, targetID, true
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound), errors.Is(err, ErrUserNotFound), errors.Is(err, ErrNotFollowing):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrUsernameTaken), errors.Is(err, ErrAlreadyFollowing):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrCannotFollowSelf),
		errors.Is(err, ErrInvalidUsername),
		errors.Is(err, ErrReservedUsername):
		response.BadRequest(c, err.Error())
	default:
		h.logger.Error("profile handler: internal error", zap.Error(err))
		response.InternalError(c, "internal error")
	}
}

func requireUser(c *gin.Context) (uuid.UUID, bool) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return uuid.Nil, false
	}
	return uid, true
}

func parseLimit(c *gin.Context) int {
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return 0
}
