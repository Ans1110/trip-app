package vote

import (
	"errors"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)
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
		logger: logger.With(zap.String("layer", "vote.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	trips := protected.Group("/trips/:id/polls")
	{
		trips.GET("", h.ListPolls)
		trips.POST("", h.CreatePoll)
	}
	polls := protected.Group("/polls/:poll_id")
	{
		polls.GET("", h.GetPoll)
		polls.DELETE("", h.DeletePoll)
		polls.POST("/options", h.AddOption)
		polls.POST("/vote", h.CastVote)
		polls.POST("/close", h.ClosePoll)
	}
	protected.DELETE("/polls/options/:option_id", h.DeleteOption)
}

// CreatePoll godoc
// @Summary  Create a poll inside a trip room
// @Tags     vote
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    trip_id  path      string             true  "trip id"
// @Param    body     body      CreatePollPayload  true  "poll"
// @Success  201      {object}  response.Response{data=PollDTO}
// @Router   /trips/{trip_id}/polls [post]
func (h *Handler) CreatePoll(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	tripID, ok := parseUUID(c, "id", "invalid trip id")
	if !ok {
		return
	}
	var p CreatePollPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreatePoll(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Created(c, out)
}

// ListPolls godoc
// @Summary  List polls for a trip room
// @Tags     vote
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path      string  true  "trip id"
// @Success  200      {object}  response.Response{data=[]PollDTO}
// @Router   /trips/{trip_id}/polls [get]
func (h *Handler) ListPolls(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	tripID, ok := parseUUID(c, "id", "invalid trip id")
	if !ok {
		return
	}
	out, err := h.svc.ListPolls(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// GetPoll godoc
// @Summary  Fetch a single poll (visibility-filtered per caller)
// @Tags     vote
// @Security BearerAuth
// @Produce  json
// @Param    poll_id  path      string  true  "poll id"
// @Success  200      {object}  response.Response{data=PollDTO}
// @Router   /polls/{poll_id} [get]
func (h *Handler) GetPoll(c *gin.Context) {
	uid, pollID, ok := h.bindUserAndPoll(c)
	if !ok {
		return
	}
	out, err := h.svc.GetPoll(c.Request.Context(), uid, pollID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// AddOption godoc
// @Summary  Append an option to an open poll
// @Description Allowed for the poll creator, and for any room member when allow_option_add=true.
// @Tags     vote
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    poll_id  path      string            true  "poll id"
// @Param    body     body      AddOptionPayload  true  "option"
// @Success  200      {object}  response.Response{data=PollDTO}
// @Router   /polls/{poll_id}/options [post]
func (h *Handler) AddOption(c *gin.Context) {
	uid, pollID, ok := h.bindUserAndPoll(c)
	if !ok {
		return
	}
	var p AddOptionPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.AddOption(c.Request.Context(), uid, pollID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteOption godoc
// @Summary  Remove an option (creator can prune any; members can only remove their own additions)
// @Tags     vote
// @Security BearerAuth
// @Produce  json
// @Param    option_id  path      string  true  "option id"
// @Success  200        {object}  response.Response{data=PollDTO}
// @Router   /polls/options/{option_id} [delete]
func (h *Handler) DeleteOption(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	optionID, ok := parseUUID(c, "option_id", "invalid option id")
	if !ok {
		return
	}
	out, err := h.svc.DeleteOption(c.Request.Context(), uid, optionID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// CastVote godoc
// @Summary  Cast (or replace) a vote. Supplying an empty option_ids array is rejected — use DELETE semantics instead.
// @Tags     vote
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    poll_id  path      string           true  "poll id"
// @Param    body     body      CastVotePayload  true  "ballot"
// @Success  200      {object}  response.Response{data=PollDTO}
// @Router   /polls/{poll_id}/vote [post]
func (h *Handler) CastVote(c *gin.Context) {
	uid, pollID, ok := h.bindUserAndPoll(c)
	if !ok {
		return
	}
	var p CastVotePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CastVote(c.Request.Context(), uid, pollID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ClosePoll godoc
// @Summary  Close a poll early (creator only)
// @Tags     vote
// @Security BearerAuth
// @Produce  json
// @Param    poll_id  path      string  true  "poll id"
// @Success  200      {object}  response.Response{data=PollDTO}
// @Router   /polls/{poll_id}/close [post]
func (h *Handler) ClosePoll(c *gin.Context) {
	uid, pollID, ok := h.bindUserAndPoll(c)
	if !ok {
		return
	}
	out, err := h.svc.ClosePoll(c.Request.Context(), uid, pollID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeletePoll godoc
// @Summary  Delete a poll and all its options/ballots (creator only)
// @Tags     vote
// @Security BearerAuth
// @Produce  json
// @Param    poll_id  path      string  true  "poll id"
// @Success  200      {object}  response.Response
// @Router   /polls/{poll_id} [delete]
func (h *Handler) DeletePoll(c *gin.Context) {
	uid, pollID, ok := h.bindUserAndPoll(c)
	if !ok {
		return
	}
	if err := h.svc.DeletePoll(c.Request.Context(), uid, pollID); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

// ---- helpers ----

func (h *Handler) bindUserAndPoll(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	pollID, ok := parseUUID(c, "poll_id", "invalid poll id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return uid, pollID, true
}

func (h *Handler) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPollNotFound), errors.Is(err, ErrOptionNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c)
	case errors.Is(err, ErrPollClosed), errors.Is(err, ErrDuplicateOption):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrInvalidPayload), errors.Is(err, ErrTooManyChoices):
		response.BadRequest(c, err.Error())
	default:
		h.logger.Error("vote handler: internal", zap.Error(err))
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

func parseUUID(c *gin.Context, param, msg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.BadRequest(c, msg)
		return uuid.Nil, false
	}
	return id, true
}
