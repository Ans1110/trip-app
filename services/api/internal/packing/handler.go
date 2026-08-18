package packing

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
		logger: logger.With(zap.String("layer", "packing.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	trips := protected.Group("/trips/:id/packing/items")
	{
		trips.GET("", h.ListItems)
		trips.POST("", h.CreateItem)
	}
	items := protected.Group("/packing/items/:item_id")
	{
		items.PATCH("", h.UpdateItem)
		items.DELETE("", h.DeleteItem)
		items.POST("/pack", h.PackItem)
		items.DELETE("/pack", h.UnpackItem)
	}
}

// ListItems godoc
// @Summary  List packing items for a trip (room members only)
// @Tags     packing
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "trip id"
// @Success  200  {object}  response.Response{data=[]ItemDTO}
// @Router   /trips/{id}/packing/items [get]
func (h *Handler) ListItems(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListItems(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// CreateItem godoc
// @Summary  Add a packing item to a trip
// @Tags     packing
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path      string             true  "trip id"
// @Param    body  body      CreateItemPayload  true  "item"
// @Success  201   {object}  response.Response{data=ItemDTO}
// @Router   /trips/{id}/packing/items [post]
func (h *Handler) CreateItem(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p CreateItemPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateItem(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Created(c, out)
}

// UpdateItem godoc
// @Summary  Patch an item's editable fields (any room member)
// @Tags     packing
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    item_id  path      string             true  "item id"
// @Param    body     body      UpdateItemPayload  true  "patch"
// @Success  200      {object}  response.Response{data=ItemDTO}
// @Router   /packing/items/{item_id} [patch]
func (h *Handler) UpdateItem(c *gin.Context) {
	uid, itemID, ok := h.bindUserAndItem(c)
	if !ok {
		return
	}
	var p UpdateItemPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateItem(c.Request.Context(), uid, itemID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteItem godoc
// @Summary  Delete a packing item (any room member)
// @Tags     packing
// @Security BearerAuth
// @Produce  json
// @Param    item_id  path  string  true  "item id"
// @Success  200      {object}  response.Response
// @Router   /packing/items/{item_id} [delete]
func (h *Handler) DeleteItem(c *gin.Context) {
	uid, itemID, ok := h.bindUserAndItem(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteItem(c.Request.Context(), uid, itemID); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

// PackItem godoc
// @Summary  Mark an item packed by the caller (any room member)
// @Tags     packing
// @Security BearerAuth
// @Produce  json
// @Param    item_id  path      string  true  "item id"
// @Success  200      {object}  response.Response{data=ItemDTO}
// @Router   /packing/items/{item_id}/pack [post]
func (h *Handler) PackItem(c *gin.Context) {
	uid, itemID, ok := h.bindUserAndItem(c)
	if !ok {
		return
	}
	out, err := h.svc.SetPacked(c.Request.Context(), uid, itemID, true)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// UnpackItem godoc
// @Summary  Clear the caller's own pack mark on an item (any room member)
// @Tags     packing
// @Security BearerAuth
// @Produce  json
// @Param    item_id  path      string  true  "item id"
// @Success  200      {object}  response.Response{data=ItemDTO}
// @Router   /packing/items/{item_id}/pack [delete]
func (h *Handler) UnpackItem(c *gin.Context) {
	uid, itemID, ok := h.bindUserAndItem(c)
	if !ok {
		return
	}
	out, err := h.svc.SetPacked(c.Request.Context(), uid, itemID, false)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ---- helpers ----

func (h *Handler) bindUserAndTrip(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	tripID, ok := parseUUID(c, "id", "invalid trip id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return uid, tripID, true
}

func (h *Handler) bindUserAndItem(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	itemID, ok := parseUUID(c, "item_id", "invalid item id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return uid, itemID, true
}

func (h *Handler) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrItemNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c)
	case errors.Is(err, ErrInvalidPayload):
		response.BadRequest(c, err.Error())
	default:
		h.logger.Error("packing handler: internal", zap.Error(err))
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
