package calendar

import (
	"context"
	"errors"
	"time"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)

	ListMyEvents(c *gin.Context)
	CreateEvent(c *gin.Context)
	GetEvent(c *gin.Context)
	UpdateEvent(c *gin.Context)
	DeleteEvent(c *gin.Context)
	ListTripEvents(c *gin.Context)
	ListFriendEvents(c *gin.Context)
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
		logger: logger.With(zap.String("layer", "calendar.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/calendar")
	{
		g.GET("/events", h.ListMyEvents)
		g.POST("/events", h.CreateEvent)
		g.GET("/events/:id", h.GetEvent)
		g.PATCH("/events/:id", h.UpdateEvent)
		g.DELETE("/events/:id", h.DeleteEvent)
		g.GET("/trips/:trip_id/events", h.ListTripEvents)
		g.GET("/friends/:friend_id/events", h.ListFriendEvents)
	}
}

// ListMyEvents godoc
// @Summary  List the caller's calendar events (via event_members)
// @Tags     calendar
// @Security BearerAuth
// @Produce  json
// @Param    from  query  string  false  "RFC3339 lower bound (inclusive on end_at)"
// @Param    to    query  string  false  "RFC3339 upper bound (inclusive on start_at)"
// @Success  200   {object}  response.Response{data=[]EventResponse}
// @Router   /calendar/events [get]
func (h *Handler) ListMyEvents(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	q, err := parseRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.ListMyEvents(c.Request.Context(), uid, q)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// CreateEvent godoc
// @Summary  Create a private or friends-visible calendar event
// @Description Trip / room events are created automatically by the trip module and cannot be produced through this endpoint.
// @Tags     calendar
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      CreateEventPayload  true  "event"
// @Success  201   {object}  response.Response{data=EventResponse}
// @Router   /calendar/events [post]
func (h *Handler) CreateEvent(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var p CreateEventPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateEvent(auditCtx(c), uid, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Created(c, out)
}

// GetEvent godoc
// @Summary  Get a single event by id (subject to visibility rules)
// @Tags     calendar
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "event id"
// @Success  200  {object}  response.Response{data=EventResponse}
// @Router   /calendar/events/{id} [get]
func (h *Handler) GetEvent(c *gin.Context) {
	uid, id, ok := h.bindUserAndEvent(c)
	if !ok {
		return
	}
	out, err := h.svc.GetEvent(c.Request.Context(), uid, id)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// UpdateEvent godoc
// @Summary  Update an event (creator only). Trip-sourced events accept only presentation edits.
// @Tags     calendar
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path      string              true  "event id"
// @Param    body  body      UpdateEventPayload  true  "patch"
// @Success  200   {object}  response.Response{data=EventResponse}
// @Router   /calendar/events/{id} [patch]
func (h *Handler) UpdateEvent(c *gin.Context) {
	uid, id, ok := h.bindUserAndEvent(c)
	if !ok {
		return
	}
	var p UpdateEventPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateEvent(auditCtx(c), uid, id, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteEvent godoc
// @Summary  Delete an event (creator only; trip-sourced events reject).
// @Tags     calendar
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "event id"
// @Success  200  {object}  response.Response
// @Router   /calendar/events/{id} [delete]
func (h *Handler) DeleteEvent(c *gin.Context) {
	uid, id, ok := h.bindUserAndEvent(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteEvent(auditCtx(c), uid, id); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListTripEvents godoc
// @Summary  Room-visible events for a given trip (caller must be a room member)
// @Tags     calendar
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path   string  true   "trip id"
// @Param    from     query  string  false  "RFC3339 lower bound"
// @Param    to       query  string  false  "RFC3339 upper bound"
// @Success  200      {object}  response.Response{data=[]EventResponse}
// @Router   /calendar/trips/{trip_id}/events [get]
func (h *Handler) ListTripEvents(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	tripID, err := uuid.Parse(c.Param("trip_id"))
	if err != nil {
		response.BadRequest(c, "invalid trip id")
		return
	}
	q, err := parseRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.ListRoomEvents(c.Request.Context(), uid, tripID, q)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ListFriendEvents godoc
// @Summary  Friend overlay — a friend's visibility=friends events. Dynamic: unfriend removes access.
// @Tags     calendar
// @Security BearerAuth
// @Produce  json
// @Param    friend_id  path   string  true   "friend user id"
// @Param    from       query  string  false  "RFC3339 lower bound"
// @Param    to         query  string  false  "RFC3339 upper bound"
// @Success  200        {object}  response.Response{data=[]EventResponse}
// @Router   /calendar/friends/{friend_id}/events [get]
func (h *Handler) ListFriendEvents(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	friendID, err := uuid.Parse(c.Param("friend_id"))
	if err != nil {
		response.BadRequest(c, "invalid friend id")
		return
	}
	q, err := parseRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.ListFriendEvents(c.Request.Context(), uid, friendID, q)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Helpers ----

func (h *Handler) bindUserAndEvent(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid event id")
		return uuid.Nil, uuid.Nil, false
	}
	return uid, id, true
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEventNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c)
	case errors.Is(err, ErrTripSourceReadOnly):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrStaleVersion):
		response.Conflict(c, "stale version")
	case errors.Is(err, ErrInvalidDateRange), errors.Is(err, ErrInvalidPayload):
		response.BadRequest(c, err.Error())
	default:
		h.logger.Error("calendar handler: internal error", zap.Error(err))
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

func auditCtx(c *gin.Context) context.Context {
	return WithAuditMeta(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
}

func parseRange(c *gin.Context) (ListEventsQuery, error) {
	q := ListEventsQuery{}
	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return q, errors.New("invalid 'from': " + err.Error())
		}
		q.From = &t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return q, errors.New("invalid 'to': " + err.Error())
		}
		q.To = &t
	}
	if q.From != nil && q.To != nil && q.To.Before(*q.From) {
		return q, errors.New("'to' must be on or after 'from'")
	}
	return q, nil
}
