package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)

	// User-facing
	SubmitReport(c *gin.Context)

	// Elevation
	Elevate(c *gin.Context)
	AdminLogout(c *gin.Context)

	// Admin
	ListUsers(c *gin.Context)
	GetUser(c *gin.Context)
	BlockUser(c *gin.Context)
	UnblockUser(c *gin.Context)
	DeactivateUser(c *gin.Context)

	ListPosts(c *gin.Context)
	DeletePost(c *gin.Context)
	RestorePost(c *gin.Context)
	DeleteComment(c *gin.Context)

	ListReports(c *gin.Context)
	ResolveReport(c *gin.Context)
	DismissReport(c *gin.Context)
}

type Handler struct {
	svc    IService
	gate   *ElevationGate
	logger *zap.Logger
}

func NewHandler(svc IService, gate *ElevationGate, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, gate: gate, logger: logger.With(zap.String("layer", "admin.handler"))}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	protected.POST("/reports", h.SubmitReport)

	admin := protected.Group("/admin")
	{
		// Elevation endpoints sit inside /admin but outside the elevation gate
		// so an admin user can mint the flag with their password.
		adminAuth := admin.Group("/auth")
		{
			adminAuth.POST("/elevate", h.Elevate)
			adminAuth.POST("/logout", h.AdminLogout)
		}
	}

	admin.Use(RequireAdminElevated(h.gate))
	{
		users := admin.Group("/users")
		{
			users.GET("", h.ListUsers)
			users.GET("/:id", h.GetUser)
			users.POST("/:id/block", h.BlockUser)
			users.POST("/:id/unblock", h.UnblockUser)
			users.POST("/:id/deactivate", h.DeactivateUser)
		}

		posts := admin.Group("/posts")
		{
			posts.GET("", h.ListPosts)
			posts.DELETE("/:id", h.DeletePost)
			posts.POST("/:id/restore", h.RestorePost)
		}

		admin.DELETE("/comments/:id", h.DeleteComment)

		reports := admin.Group("/reports")
		{
			reports.GET("", h.ListReports)
			reports.POST("/:id/resolve", h.ResolveReport)
			reports.POST("/:id/dismiss", h.DismissReport)
		}
	}
}

// Elevate godoc
// @Summary      Elevate to admin session (yml-configured super-admin only)
// @Tags         admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      ElevatePayload  true  "admin password"
// @Success      200   {object}  response.Response{data=ElevationResponse}
// @Router       /admin/auth/elevate [post]
func (h *Handler) Elevate(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var payload ElevatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	email := middleware.GetUserEmail(c)
	expires, err := h.gate.Elevate(c.Request.Context(), uid, email, payload.Password)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, ElevationResponse{Elevated: true, ExpiresAt: expires})
}

// AdminLogout godoc
// @Summary      Revoke the current admin elevation grant
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /admin/auth/logout [post]
func (h *Handler) AdminLogout(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	if err := h.gate.Revoke(c.Request.Context(), uid); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// SubmitReport godoc
// @Summary      Submit a report for a post, comment, or user
// @Tags         reports
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CreateReportPayload  true  "report fields"
// @Success      200   {object}  response.Response{data=ReportResponse}
// @Router       /reports [post]
func (h *Handler) SubmitReport(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var payload CreateReportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.SubmitReport(c.Request.Context(), uid, payload)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, out)
}

// ListUsers godoc
// @Summary      List users (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        q          query  string  false  "search email/name"
// @Param        status     query  string  false  "active|deactivated|deleted"
// @Param        is_blocked query  string  false  "true|false"
// @Param        cursor     query  string  false  "opaque cursor"
// @Param        limit      query  int     false  "page size"
// @Success      200        {object}  response.Response{data=ListUsersResponse}
// @Router       /admin/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	filter := UserFilter{
		Keyword: c.Query("q"),
		Status:  c.Query("status"),
	}
	if raw := strings.TrimSpace(c.Query("is_blocked")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "invalid is_blocked")
			return
		}
		filter.IsBlocked = &v
	}
	cursor, err := DecodeUsersCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.ListUsers(c.Request.Context(), filter, cursor, parseLimit(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, out)
}

// GetUser godoc
// @Summary      Get user detail (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "user id"
// @Success      200  {object}  response.Response{data=AdminUserResponse}
// @Router       /admin/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	targetID, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	out, err := h.svc.GetUser(c.Request.Context(), targetID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, out)
}

// BlockUser godoc
// @Summary      Block a user (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "user id"
// @Success      200  {object}  response.Response
// @Router       /admin/users/{id}/block [post]
func (h *Handler) BlockUser(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	if err := h.svc.BlockUser(c.Request.Context(), actor, target); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// UnblockUser godoc
// @Summary      Unblock a user (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "user id"
// @Success      200  {object}  response.Response
// @Router       /admin/users/{id}/unblock [post]
func (h *Handler) UnblockUser(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	if err := h.svc.UnblockUser(c.Request.Context(), actor, target); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// DeactivateUser godoc
// @Summary      Deactivate a user (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "user id"
// @Success      200  {object}  response.Response
// @Router       /admin/users/{id}/deactivate [post]
func (h *Handler) DeactivateUser(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeactivateUser(c.Request.Context(), actor, target); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListPosts godoc
// @Summary      List posts (admin, includes soft-deleted when include_deleted=true)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        q                query  string  false  "search title/content"
// @Param        status           query  string  false  "draft|published|archived"
// @Param        author_id        query  string  false  "filter by author"
// @Param        include_deleted  query  string  false  "true|false"
// @Param        cursor           query  string  false  "opaque cursor"
// @Param        limit            query  int     false  "page size"
// @Success      200              {object}  response.Response{data=ListPostsResponse}
// @Router       /admin/posts [get]
func (h *Handler) ListPosts(c *gin.Context) {
	filter := PostFilter{
		Keyword: c.Query("q"),
		Status:  c.Query("status"),
	}
	if raw := c.Query("author_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "invalid author id")
			return
		}
		filter.AuthorID = id
	}
	if raw := strings.TrimSpace(c.Query("include_deleted")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "invalid include_deleted")
			return
		}
		filter.IncludeDel = v
	}
	cursor, err := DecodePostsCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.ListPosts(c.Request.Context(), filter, cursor, parseLimit(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, out)
}

// DeletePost godoc
// @Summary      Soft-delete a post (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response
// @Router       /admin/posts/{id} [delete]
func (h *Handler) DeletePost(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeletePost(c.Request.Context(), actor, target); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// RestorePost godoc
// @Summary      Restore a soft-deleted post (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response
// @Router       /admin/posts/{id}/restore [post]
func (h *Handler) RestorePost(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	if err := h.svc.RestorePost(c.Request.Context(), actor, target); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// DeleteComment godoc
// @Summary      Soft-delete a comment (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "comment id"
// @Success      200  {object}  response.Response
// @Router       /admin/comments/{id} [delete]
func (h *Handler) DeleteComment(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteComment(c.Request.Context(), actor, target); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListReports godoc
// @Summary      List reports (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        status        query  string  false  "pending|resolved|dismissed"
// @Param        subject_type  query  string  false  "post|comment|user"
// @Param        cursor        query  string  false  "opaque cursor"
// @Param        limit         query  int     false  "page size"
// @Success      200           {object}  response.Response{data=ListReportsResponse}
// @Router       /admin/reports [get]
func (h *Handler) ListReports(c *gin.Context) {
	filter := ReportFilter{
		Status:      c.Query("status"),
		SubjectType: c.Query("subject_type"),
	}
	if filter.Status != "" && !IsValidStatus(filter.Status) {
		response.BadRequest(c, "invalid status")
		return
	}
	if filter.SubjectType != "" && !IsValidSubject(filter.SubjectType) {
		response.BadRequest(c, "invalid subject_type")
		return
	}
	cursor, err := DecodeReportsCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.ListReports(c.Request.Context(), filter, cursor, parseLimit(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, out)
}

// ResolveReport godoc
// @Summary      Mark a report resolved (admin)
// @Tags         admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                 true  "report id"
// @Param        body  body      ResolveReportPayload   false "resolution note"
// @Success      200   {object}  response.Response{data=ReportResponse}
// @Router       /admin/reports/{id}/resolve [post]
func (h *Handler) ResolveReport(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	var payload ResolveReportPayload
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&payload); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}
	out, err := h.svc.ResolveReport(c.Request.Context(), actor, target, payload)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, out)
}

// DismissReport godoc
// @Summary      Dismiss a report (admin)
// @Tags         admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                 true  "report id"
// @Param        body  body      ResolveReportPayload   false "dismissal note"
// @Success      200   {object}  response.Response{data=ReportResponse}
// @Router       /admin/reports/{id}/dismiss [post]
func (h *Handler) DismissReport(c *gin.Context) {
	actor, target, ok := h.requireActorTarget(c, "id")
	if !ok {
		return
	}
	var payload ResolveReportPayload
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&payload); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}
	out, err := h.svc.DismissReport(c.Request.Context(), actor, target, payload)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) requireActorTarget(c *gin.Context, param string) (uuid.UUID, uuid.UUID, bool) {
	actor, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	target, ok := parseUUID(c, param)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return actor, target, true
}

func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrReportNotFound),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrPostNotFound),
		errors.Is(err, ErrCommentNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrDuplicateReport):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrInvalidReporter):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrElevationDenied),
		errors.Is(err, ErrElevationUnavailable),
		errors.Is(err, ErrNotElevated):
		response.Forbidden(c)
	default:
		h.logger.Error("admin handler: internal error", zap.Error(err))
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

func parseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.BadRequest(c, "invalid "+param)
		return uuid.Nil, false
	}
	return id, true
}

func parseLimit(c *gin.Context) int {
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return 0
}
