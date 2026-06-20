package trip

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

	// Trip
	CreateTrip(c *gin.Context)
	ListTrips(c *gin.Context)
	GetTrip(c *gin.Context)
	UpdateTrip(c *gin.Context)
	DeleteTrip(c *gin.Context)

	// Room
	GetRoom(c *gin.Context)
	LeaveRoom(c *gin.Context)
	RemoveMember(c *gin.Context)
	RegenerateCode(c *gin.Context)
	PreviewByCode(c *gin.Context)
	JoinRoom(c *gin.Context)

	// Itinerary
	CreateItinerary(c *gin.Context)
	ListItinerary(c *gin.Context)
	UpdateItinerary(c *gin.Context)
	DeleteItinerary(c *gin.Context)

	// Todos
	CreateTodo(c *gin.Context)
	ListTodos(c *gin.Context)
	UpdateTodo(c *gin.Context)
	DeleteTodo(c *gin.Context)
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
		logger: logger.With(zap.String("layer", "trip.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	trips := protected.Group("/trips")
	{
		trips.GET("", h.ListTrips)
		trips.POST("", h.CreateTrip)
		trips.GET("/:id", h.GetTrip)
		trips.PATCH("/:id", h.UpdateTrip)
		trips.DELETE("/:id", h.DeleteTrip)

		trips.GET("/:id/room", h.GetRoom)
		trips.POST("/:id/room/leave", h.LeaveRoom)
		trips.DELETE("/:id/room/members/:user_id", h.RemoveMember)
		trips.POST("/:id/room/code", h.RegenerateCode)

		trips.GET("/:id/itinerary", h.ListItinerary)
		trips.POST("/:id/itinerary", h.CreateItinerary)
		trips.PATCH("/:id/itinerary/:item_id", h.UpdateItinerary)
		trips.DELETE("/:id/itinerary/:item_id", h.DeleteItinerary)

		trips.GET("/:id/todos", h.ListTodos)
		trips.POST("/:id/todos", h.CreateTodo)
		trips.PATCH("/:id/todos/:todo_id", h.UpdateTodo)
		trips.DELETE("/:id/todos/:todo_id", h.DeleteTodo)
	}

	rooms := protected.Group("/rooms")
	{
		rooms.GET("/by-code/:code", h.PreviewByCode)
		rooms.POST("/join", h.JoinRoom)
	}
}

// ---- Trip ----

// CreateTrip godoc
// @Summary  Create a trip and its room
// @Tags     trips
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      CreateTripPayload  true  "trip"
// @Success  201   {object}  response.Response{data=TripResponse}
// @Router   /trips [post]
func (h *Handler) CreateTrip(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var p CreateTripPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateTrip(auditCtx(c), uid, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Created(c, out)
}

// ListTrips godoc
// @Summary  List trips the caller owns or is a room member of
// @Tags     trips
// @Security BearerAuth
// @Produce  json
// @Param    status  query  string  false  "planning|ongoing|completed"
// @Param    q       query  string  false  "title search"
// @Param    role    query  string  false  "owner|member|all"
// @Param    limit   query  int     false  "default 20, max 100"
// @Param    offset  query  int     false  "default 0"
// @Success  200     {object}  response.Response{data=[]TripResponse}
// @Router   /trips [get]
func (h *Handler) ListTrips(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	q := ListTripsQuery{
		Status: c.Query("status"),
		Q:      c.Query("q"),
		Role:   c.Query("role"),
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			q.Offset = n
		}
	}
	out, err := h.svc.ListTrips(c.Request.Context(), uid, q)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// GetTrip godoc
// @Summary  Fetch a trip by id (caller must be owner or room member)
// @Tags     trips
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "trip id"
// @Success  200  {object}  response.Response{data=TripResponse}
// @Router   /trips/{id} [get]
func (h *Handler) GetTrip(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.GetTrip(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// UpdateTrip godoc
// @Summary  Update a trip (owner only)
// @Tags     trips
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path      string  true  "trip id"
// @Param    body  body      UpdateTripPayload  true  "patch"
// @Success  200   {object}  response.Response{data=TripResponse}
// @Router   /trips/{id} [patch]
func (h *Handler) UpdateTrip(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p UpdateTripPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateTrip(auditCtx(c), uid, tripID, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteTrip godoc
// @Summary  Soft-delete a trip (owner only)
// @Tags     trips
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "trip id"
// @Success  200  {object}  response.Response
// @Router   /trips/{id} [delete]
func (h *Handler) DeleteTrip(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteTrip(auditCtx(c), uid, tripID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ---- Room ----

// GetRoom godoc
// @Summary  Get the room (with members) for a trip
// @Tags     trips,rooms
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "trip id"
// @Success  200  {object}  response.Response{data=RoomResponse}
// @Router   /trips/{id}/room [get]
func (h *Handler) GetRoom(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.GetRoom(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// LeaveRoom godoc
// @Summary  Leave a room (trip owner cannot)
// @Tags     trips,rooms
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "trip id"
// @Success  200  {object}  response.Response
// @Router   /trips/{id}/room/leave [post]
func (h *Handler) LeaveRoom(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	if err := h.svc.LeaveRoom(auditCtx(c), uid, tripID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// RemoveMember godoc
// @Summary  Kick a member from the room (admin only)
// @Tags     trips,rooms
// @Security BearerAuth
// @Produce  json
// @Param    id       path  string  true  "trip id"
// @Param    user_id  path  string  true  "member id to remove"
// @Success  200  {object}  response.Response
// @Router   /trips/{id}/room/members/{user_id} [delete]
func (h *Handler) RemoveMember(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.svc.RemoveMember(auditCtx(c), uid, tripID, targetID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// RegenerateCode godoc
// @Summary  Regenerate room code (admin only)
// @Tags     trips,rooms
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "trip id"
// @Success  200  {object}  response.Response{data=RoomResponse}
// @Router   /trips/{id}/room/code [post]
func (h *Handler) RegenerateCode(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.RegenerateCode(auditCtx(c), uid, tripID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// PreviewByCode godoc
// @Summary  Preview a room by code (used after QR scan / code entry, before join)
// @Tags     rooms
// @Security BearerAuth
// @Produce  json
// @Param    code  path  string  true  "8-char room code"
// @Success  200   {object}  response.Response{data=RoomPreview}
// @Router   /rooms/by-code/{code} [get]
func (h *Handler) PreviewByCode(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	out, err := h.svc.PreviewByCode(c.Request.Context(), uid, c.Param("code"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// JoinRoom godoc
// @Summary  Join a room via room code (shared by QR or raw code entry)
// @Tags     rooms
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body  JoinRoomPayload  true  "code"
// @Success  200   {object}  response.Response{data=JoinRoomResponse}
// @Router   /rooms/join [post]
func (h *Handler) JoinRoom(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var p JoinRoomPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.JoinRoom(auditCtx(c), uid, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Itinerary ----

func (h *Handler) CreateItinerary(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p CreateItineraryPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateItinerary(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Created(c, out)
}

func (h *Handler) ListItinerary(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListItinerary(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) UpdateItinerary(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		response.BadRequest(c, "invalid item id")
		return
	}
	var p UpdateItineraryPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateItinerary(c.Request.Context(), uid, tripID, itemID, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) DeleteItinerary(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		response.BadRequest(c, "invalid item id")
		return
	}
	if err := h.svc.DeleteItinerary(c.Request.Context(), uid, tripID, itemID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ---- Todos ----

func (h *Handler) CreateTodo(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p CreateTodoPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateTodo(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Created(c, out)
}

func (h *Handler) ListTodos(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListTodos(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) UpdateTodo(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	todoID, err := uuid.Parse(c.Param("todo_id"))
	if err != nil {
		response.BadRequest(c, "invalid todo id")
		return
	}
	var p UpdateTodoPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateTodo(c.Request.Context(), uid, tripID, todoID, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) DeleteTodo(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	todoID, err := uuid.Parse(c.Param("todo_id"))
	if err != nil {
		response.BadRequest(c, "invalid todo id")
		return
	}
	if err := h.svc.DeleteTodo(c.Request.Context(), uid, tripID, todoID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ---- Helpers ----

func (h *Handler) bindUserAndTrip(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid trip id")
		return uuid.Nil, uuid.Nil, false
	}
	return uid, tripID, true
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrTripNotFound),
		errors.Is(err, ErrRoomNotFound),
		errors.Is(err, ErrItineraryNotFound),
		errors.Is(err, ErrTodoNotFound),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrNotMember):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrOwnerCannotLeave),
		errors.Is(err, ErrCannotRemoveOwner):
		response.Forbidden(c)
	case errors.Is(err, ErrInvalidDateRange),
		errors.Is(err, ErrInvalidPayload),
		errors.Is(err, ErrCodeRequired):
		response.BadRequest(c, err.Error())
	default:
		h.logger.Error("trip handler: internal error", zap.Error(err))
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
