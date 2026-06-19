package friend

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)

	List(c *gin.Context)
	Remove(c *gin.Context)
	Search(c *gin.Context)

	SendRequest(c *gin.Context)
	AcceptRequest(c *gin.Context)
	DeclineRequest(c *gin.Context)
	CancelRequest(c *gin.Context)
	IncomingRequests(c *gin.Context)
	OutgoingRequests(c *gin.Context)

	Block(c *gin.Context)
	Unblock(c *gin.Context)
	ListBlocks(c *gin.Context)

	CreateInvite(c *gin.Context)
	ListInvites(c *gin.Context)
	RevokeInvite(c *gin.Context)
	AcceptInvite(c *gin.Context)
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
		logger: logger.With(zap.String("layer", "friend.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/friends")
	{
		g.GET("", h.List)
		g.GET("/search", h.Search)
		g.DELETE("/:user_id", h.Remove)

		g.GET("/requests", h.IncomingRequests)
		g.GET("/requests/outgoing", h.OutgoingRequests)
		g.POST("/requests", h.SendRequest)
		g.POST("/requests/:id/accept", h.AcceptRequest)
		g.POST("/requests/:id/decline", h.DeclineRequest)
		g.DELETE("/requests/:id", h.CancelRequest)

		g.GET("/blocks", h.ListBlocks)
		g.POST("/blocks", h.Block)
		g.DELETE("/blocks/:user_id", h.Unblock)

		g.GET("/invites", h.ListInvites)
		g.POST("/invites", h.CreateInvite)
		g.DELETE("/invites/:id", h.RevokeInvite)
		g.POST("/invites/accept", h.AcceptInvite)
	}
}

// List godoc
// @Summary      List the caller's friends
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=[]FriendResponse}
// @Failure      401  {object}  response.Response
// @Router       /friends [get]
func (h *Handler) List(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	friends, err := h.svc.ListFriends(c.Request.Context(), uid)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, friends)
}

// Search godoc
// @Summary      Search users by email or name (excludes self)
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Param        q      query     string  true   "search term (email or name)"
// @Param        limit  query     int     false  "max results (default 20, cap 50)"
// @Success      200    {object}  response.Response{data=[]UserSummary}
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Router       /friends/search [get]
func (h *Handler) Search(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "query parameter q is required")
		return
	}
	limit := 0
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}
	users, err := h.svc.SearchUsers(c.Request.Context(), uid, query, limit)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, users)
}

// Remove godoc
// @Summary      Remove a friend
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Param        user_id  path      string  true  "friend's user id"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Router       /friends/{user_id} [delete]
func (h *Handler) Remove(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	friendID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.svc.RemoveFriend(auditCtx(c), uid, friendID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// IncomingRequests godoc
// @Summary      List pending friend requests where the caller is the receiver
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=[]RequestResponse}
// @Failure      401  {object}  response.Response
// @Router       /friends/requests [get]
func (h *Handler) IncomingRequests(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	reqs, err := h.svc.ListIncomingRequests(c.Request.Context(), uid)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, reqs)
}

// OutgoingRequests godoc
// @Summary      List pending friend requests sent by the caller
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=[]RequestResponse}
// @Failure      401  {object}  response.Response
// @Router       /friends/requests/outgoing [get]
func (h *Handler) OutgoingRequests(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	reqs, err := h.svc.ListOutgoingRequests(c.Request.Context(), uid)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, reqs)
}

// SendRequest godoc
// @Summary      Send a friend request
// @Tags         friends
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      SendRequestPayload  true  "receiver id and optional message"
// @Success      201   {object}  response.Response{data=RequestResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Failure      404   {object}  response.Response
// @Failure      409   {object}  response.Response
// @Router       /friends/requests [post]
func (h *Handler) SendRequest(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var req SendRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	receiverID, err := uuid.Parse(req.ReceiverID)
	if err != nil {
		response.BadRequest(c, "invalid receiver id")
		return
	}
	out, err := h.svc.SendRequest(auditCtx(c), uid, receiverID, req.Message)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Created(c, out)
}

// AcceptRequest godoc
// @Summary      Accept an incoming friend request
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "request id"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /friends/requests/{id}/accept [post]
func (h *Handler) AcceptRequest(c *gin.Context) {
	h.actOnRequest(c, h.svc.AcceptRequest)
}

// DeclineRequest godoc
// @Summary      Decline an incoming friend request
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "request id"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /friends/requests/{id}/decline [post]
func (h *Handler) DeclineRequest(c *gin.Context) {
	h.actOnRequest(c, h.svc.DeclineRequest)
}

// CancelRequest godoc
// @Summary      Cancel an outgoing friend request
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "request id"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /friends/requests/{id} [delete]
func (h *Handler) CancelRequest(c *gin.Context) {
	h.actOnRequest(c, h.svc.CancelRequest)
}

// ListBlocks godoc
// @Summary      List users the caller has blocked
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=[]BlockResponse}
// @Failure      401  {object}  response.Response
// @Router       /friends/blocks [get]
func (h *Handler) ListBlocks(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	rows, err := h.svc.ListBlocks(c.Request.Context(), uid)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, rows)
}

// Block godoc
// @Summary      Block a user (also removes existing friendship and pending requests)
// @Tags         friends
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      BlockPayload  true  "target id"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      404   {object}  response.Response
// @Router       /friends/blocks [post]
func (h *Handler) Block(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var req BlockPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		response.BadRequest(c, "invalid target id")
		return
	}
	if err := h.svc.BlockUser(auditCtx(c), uid, targetID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// Unblock godoc
// @Summary      Unblock a user
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Param        user_id  path      string  true  "previously blocked user's id"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Router       /friends/blocks/{user_id} [delete]
func (h *Handler) Unblock(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.svc.UnblockUser(auditCtx(c), uid, targetID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListInvites godoc
// @Summary      List the caller's active add-friend invitations
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=[]InviteResponse}
// @Failure      401  {object}  response.Response
// @Router       /friends/invites [get]
func (h *Handler) ListInvites(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	rows, err := h.svc.ListInvites(c.Request.Context(), uid)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, rows)
}

// CreateInvite godoc
// @Summary      Create a single-use or multi-use add-friend invitation
// @Description  Returns a token and a share URL the inviter can render as a QR code
//               or send as a link. The token only appears in this response.
// @Tags         friends
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CreateInvitePayload  false  "max uses and TTL overrides"
// @Success      201   {object}  response.Response{data=InviteResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Router       /friends/invites [post]
func (h *Handler) CreateInvite(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var req CreateInvitePayload
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}
	ttl := time.Duration(req.TTLSeconds) * time.Second
	out, err := h.svc.CreateInvite(auditCtx(c), uid, ttl, req.MaxUses)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Created(c, out)
}

// RevokeInvite godoc
// @Summary      Revoke an active add-friend invitation
// @Tags         friends
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "invite id"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /friends/invites/{id} [delete]
func (h *Handler) RevokeInvite(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	inviteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid invite id")
		return
	}
	if err := h.svc.RevokeInvite(auditCtx(c), uid, inviteID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// AcceptInvite godoc
// @Summary      Accept an add-friend invitation by token (QR scan or shared link)
// @Tags         friends
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      AcceptInvitePayload  true  "invite token"
// @Success      200   {object}  response.Response{data=UserSummary}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Failure      404   {object}  response.Response
// @Failure      409   {object}  response.Response
// @Router       /friends/invites/accept [post]
func (h *Handler) AcceptInvite(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var req AcceptInvitePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.AcceptInvite(auditCtx(c), uid, req.Token)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) actOnRequest(c *gin.Context, fn func(ctx context.Context, userID, requestID uuid.UUID) error) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return
	}
	if err := fn(auditCtx(c), uid, requestID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrRequestNotFound),
		errors.Is(err, ErrInviteNotFound),
		errors.Is(err, ErrNotFriends):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrAlreadyFriends),
		errors.Is(err, ErrRequestExists):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrCannotTargetSelf),
		errors.Is(err, ErrInviteSelf),
		errors.Is(err, ErrInviteInvalid),
		errors.Is(err, ErrInviteExhausted):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrBlocked):
		response.Forbidden(c)
	default:
		h.logger.Error("friend handler: internal error", zap.Error(err))
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

// auditCtx wraps the request context with the caller's IP and User-Agent so
// the service can attach them to audit log rows.
func auditCtx(c *gin.Context) context.Context {
	return WithAuditMeta(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
}

