package post

import (
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

	CreatePost(c *gin.Context)
	UpdatePost(c *gin.Context)
	DeletePost(c *gin.Context)
	GetPost(c *gin.Context)
	ListPosts(c *gin.Context)

	LikePost(c *gin.Context)
	UnlikePost(c *gin.Context)

	BookmarkPost(c *gin.Context)
	UnbookmarkPost(c *gin.Context)
	ListBookmarks(c *gin.Context)

	CreateComment(c *gin.Context)
	UpdateComment(c *gin.Context)
	DeleteComment(c *gin.Context)
	ListComments(c *gin.Context)
	ListReplies(c *gin.Context)
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
		logger: logger.With(zap.String("layer", "post.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	posts := protected.Group("/posts")
	{
		posts.POST("", h.CreatePost)
		posts.GET("", h.ListPosts)
		posts.GET("/:id", h.GetPost)
		posts.PATCH("/:id", h.UpdatePost)
		posts.DELETE("/:id", h.DeletePost)

		posts.POST("/:id/likes", h.LikePost)
		posts.DELETE("/:id/likes", h.UnlikePost)

		posts.POST("/:id/bookmarks", h.BookmarkPost)
		posts.DELETE("/:id/bookmarks", h.UnbookmarkPost)

		posts.GET("/:id/comments", h.ListComments)
		posts.POST("/:id/comments", h.CreateComment)
		posts.PATCH("/:id/comments/:comment_id", h.UpdateComment)
		posts.DELETE("/:id/comments/:comment_id", h.DeleteComment)
		posts.GET("/:id/comments/:comment_id/replies", h.ListReplies)
	}
	protected.GET("/bookmarks", h.ListBookmarks)
}

// CreatePost godoc
// @Summary      Publish a new travel post
// @Tags         posts
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CreatePostPayload  true  "post fields"
// @Success      200   {object}  response.Response{data=PostResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Router       /posts [post]
func (h *Handler) CreatePost(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var payload CreatePostPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.CreatePost(c.Request.Context(), uid, payload)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, p)
}

// UpdatePost godoc
// @Summary      Edit an existing post (author only)
// @Tags         posts
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "post id"
// @Param        body  body      UpdatePostPayload  true  "fields to patch"
// @Success      200   {object}  response.Response{data=PostResponse}
// @Failure      400   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Failure      404   {object}  response.Response
// @Router       /posts/{id} [patch]
func (h *Handler) UpdatePost(c *gin.Context) {
	uid, postID, ok := h.requireCallerAndPost(c)
	if !ok {
		return
	}
	var payload UpdatePostPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.UpdatePost(c.Request.Context(), uid, postID, payload)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, p)
}

// DeletePost godoc
// @Summary      Delete a post (author only)
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /posts/{id} [delete]
func (h *Handler) DeletePost(c *gin.Context) {
	uid, postID, ok := h.requireCallerAndPost(c)
	if !ok {
		return
	}
	if err := h.svc.DeletePost(c.Request.Context(), uid, postID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// GetPost godoc
// @Summary      Get a post by id
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response{data=PostResponse}
// @Failure      404  {object}  response.Response
// @Router       /posts/{id} [get]
func (h *Handler) GetPost(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	p, err := h.svc.GetPost(c.Request.Context(), viewerID, postID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, p)
}

// ListPosts godoc
// @Summary      List posts (optionally filter by author, keyset paginated)
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        author_id  query     string  false  "filter by author user id"
// @Param        cursor     query     string  false  "opaque cursor from previous page"
// @Param        limit      query     int     false  "page size (default 20, max 50)"
// @Success      200        {object}  response.Response{data=ListPostsResponse}
// @Router       /posts [get]
func (h *Handler) ListPosts(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	var authorID uuid.UUID
	if s := c.Query("author_id"); s != "" {
		parsed, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid author id")
			return
		}
		authorID = parsed
	}
	cursor, err := DecodePostCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	statusFilter := c.Query("status")
	if statusFilter != "" && !IsValidStatus(statusFilter) {
		response.BadRequest(c, "invalid status")
		return
	}
	out, err := h.svc.ListPosts(c.Request.Context(), viewerID, authorID, statusFilter, cursor, parseLimit(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// LikePost godoc
// @Summary      Like a post
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Router       /posts/{id}/likes [post]
func (h *Handler) LikePost(c *gin.Context) {
	uid, postID, ok := h.requireCallerAndPost(c)
	if !ok {
		return
	}
	if err := h.svc.LikePost(c.Request.Context(), uid, postID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// UnlikePost godoc
// @Summary      Remove a like from a post
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Router       /posts/{id}/likes [delete]
func (h *Handler) UnlikePost(c *gin.Context) {
	uid, postID, ok := h.requireCallerAndPost(c)
	if !ok {
		return
	}
	if err := h.svc.UnlikePost(c.Request.Context(), uid, postID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// BookmarkPost godoc
// @Summary      Bookmark a post for later
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /posts/{id}/bookmarks [post]
func (h *Handler) BookmarkPost(c *gin.Context) {
	uid, postID, ok := h.requireCallerAndPost(c)
	if !ok {
		return
	}
	if err := h.svc.BookmarkPost(c.Request.Context(), uid, postID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// UnbookmarkPost godoc
// @Summary      Remove a bookmark
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "post id"
// @Success      200  {object}  response.Response
// @Router       /posts/{id}/bookmarks [delete]
func (h *Handler) UnbookmarkPost(c *gin.Context) {
	uid, postID, ok := h.requireCallerAndPost(c)
	if !ok {
		return
	}
	if err := h.svc.UnbookmarkPost(c.Request.Context(), uid, postID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListBookmarks godoc
// @Summary      List the caller's bookmarked posts (keyset paginated)
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        cursor  query     string  false  "opaque cursor from previous page"
// @Param        limit   query     int     false  "page size (default 20, max 50)"
// @Success      200     {object}  response.Response{data=ListPostsResponse}
// @Router       /bookmarks [get]
func (h *Handler) ListBookmarks(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	cursor, err := DecodeBookmarkCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.ListBookmarks(c.Request.Context(), uid, cursor, parseLimit(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// CreateComment godoc
// @Summary      Add a comment to a post
// @Tags         posts
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "post id"
// @Param        body  body      CreateCommentPayload  true  "comment fields"
// @Success      200   {object}  response.Response{data=CommentResponse}
// @Failure      400   {object}  response.Response
// @Failure      404   {object}  response.Response
// @Router       /posts/{id}/comments [post]
func (h *Handler) CreateComment(c *gin.Context) {
	uid, postID, ok := h.requireCallerAndPost(c)
	if !ok {
		return
	}
	var payload CreateCommentPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateComment(c.Request.Context(), uid, postID, payload)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// UpdateComment godoc
// @Summary      Edit a comment (author only)
// @Tags         posts
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id          path      string                true  "post id"
// @Param        comment_id  path      string                true  "comment id"
// @Param        body        body      UpdateCommentPayload  true  "fields to patch"
// @Success      200         {object}  response.Response{data=CommentResponse}
// @Failure      403         {object}  response.Response
// @Failure      404         {object}  response.Response
// @Router       /posts/{id}/comments/{comment_id} [patch]
func (h *Handler) UpdateComment(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		response.BadRequest(c, "invalid comment id")
		return
	}
	var payload UpdateCommentPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateComment(c.Request.Context(), uid, commentID, payload)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteComment godoc
// @Summary      Delete a comment (author only)
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id          path      string  true  "post id"
// @Param        comment_id  path      string  true  "comment id"
// @Success      200         {object}  response.Response
// @Failure      403         {object}  response.Response
// @Failure      404         {object}  response.Response
// @Router       /posts/{id}/comments/{comment_id} [delete]
func (h *Handler) DeleteComment(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		response.BadRequest(c, "invalid comment id")
		return
	}
	if err := h.svc.DeleteComment(c.Request.Context(), uid, commentID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListComments godoc
// @Summary      List comments on a post (keyset paginated)
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id      path      string  true   "post id"
// @Param        cursor  query     string  false  "opaque cursor from previous page"
// @Param        limit   query     int     false  "page size (default 20, max 100)"
// @Success      200     {object}  response.Response{data=ListCommentsResponse}
// @Router       /posts/{id}/comments [get]
func (h *Handler) ListComments(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	cursor, err := DecodeCommentCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.ListComments(c.Request.Context(), viewerID, postID, cursor, parseLimit(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ListReplies godoc
// @Summary      List replies to a top-level comment (keyset paginated, ascending)
// @Tags         posts
// @Security     BearerAuth
// @Produce      json
// @Param        id          path      string  true   "post id"
// @Param        comment_id  path      string  true   "parent comment id"
// @Param        cursor      query     string  false  "opaque cursor from previous page"
// @Param        limit       query     int     false  "page size (default 20, max 100)"
// @Success      200         {object}  response.Response{data=ListRepliesResponse}
// @Router       /posts/{id}/comments/{comment_id}/replies [get]
func (h *Handler) ListReplies(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		response.BadRequest(c, "invalid comment id")
		return
	}
	cursor, err := DecodeReplyCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "invalid cursor")
		return
	}
	out, err := h.svc.ListReplies(c.Request.Context(), viewerID, commentID, cursor, parseLimit(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) requireCallerAndPost(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return uuid.Nil, uuid.Nil, false
	}
	return uid, postID, true
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPostNotFound), errors.Is(err, ErrCommentNotFound), errors.Is(err, ErrParentNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrInvalidParent):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c)
	case errors.Is(err, ErrAlreadyLiked), errors.Is(err, ErrNotLiked):
		response.Conflict(c, err.Error())
	default:
		h.logger.Error("post handler: internal error", zap.Error(err))
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
