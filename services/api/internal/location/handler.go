package location

import (
	"context"
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
}

type Handler struct {
	svc    IService
	logger *zap.Logger
}

func NewHandler(svc IService, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger.With(zap.String("layer", "location.handler"))}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/location")
	{
		// Saved places (user-scoped, trip binding is optional)
		g.GET("/saved", h.ListSavedPlaces)
		g.POST("/saved", h.CreateSavedPlace)
		g.GET("/saved/:id", h.GetSavedPlace)
		g.PATCH("/saved/:id", h.UpdateSavedPlace)
		g.DELETE("/saved/:id", h.DeleteSavedPlace)

		// Places (proxy)
		g.GET("/places/search", h.SearchPlaces)
		g.GET("/places/nearby", h.NearbyPlaces)
		g.GET("/places/:place_id", h.PlaceDetails)

		// Routes
		g.POST("/routes", h.ComputeRoute)

		// Weather
		g.GET("/weather", h.Weather)

		// Opt-in live position sharing within a trip (never passive; the FE
		// only calls Share after the user clicks the button).
		g.POST("/trips/:trip_id/share", h.ShareTripPosition)
		g.DELETE("/trips/:trip_id/share", h.RevokeTripPosition)
		g.GET("/trips/:trip_id/members", h.ListTripPositions)
	}
}

// ---- Saved places ----

// CreateSavedPlace godoc
// @Summary  Save a place for the caller (optionally scoped to a trip they belong to)
// @Tags     location
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      CreateSavedPlacePayload  true  "place"
// @Success  201   {object}  response.Response{data=SavedPlaceResponse}
// @Router   /location/saved [post]
func (h *Handler) CreateSavedPlace(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var p CreateSavedPlacePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateSavedPlace(auditCtx(c), uid, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Created(c, out)
}

// UpdateSavedPlace godoc
// @Summary  Update fields on a saved place (owner only)
// @Tags     location
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path      string                    true  "saved place id"
// @Param    body  body      UpdateSavedPlacePayload   true  "patch"
// @Success  200   {object}  response.Response{data=SavedPlaceResponse}
// @Router   /location/saved/{id} [patch]
func (h *Handler) UpdateSavedPlace(c *gin.Context) {
	uid, id, ok := h.bindUserAndSaved(c)
	if !ok {
		return
	}
	var p UpdateSavedPlacePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateSavedPlace(auditCtx(c), uid, id, p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteSavedPlace godoc
// @Summary  Delete a saved place (owner only)
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "saved place id"
// @Success  200  {object}  response.Response
// @Router   /location/saved/{id} [delete]
func (h *Handler) DeleteSavedPlace(c *gin.Context) {
	uid, id, ok := h.bindUserAndSaved(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteSavedPlace(auditCtx(c), uid, id); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// GetSavedPlace godoc
// @Summary  Fetch a saved place (owner only)
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "saved place id"
// @Success  200  {object}  response.Response{data=SavedPlaceResponse}
// @Router   /location/saved/{id} [get]
func (h *Handler) GetSavedPlace(c *gin.Context) {
	uid, id, ok := h.bindUserAndSaved(c)
	if !ok {
		return
	}
	out, err := h.svc.GetSavedPlace(c.Request.Context(), uid, id)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ListSavedPlaces godoc
// @Summary  List the caller's saved places; optionally filter by trip binding
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  query  string  false  "restrict to a specific trip the caller belongs to"
// @Success  200  {object}  response.Response{data=[]SavedPlaceResponse}
// @Router   /location/saved [get]
func (h *Handler) ListSavedPlaces(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	q := ListSavedPlacesQuery{}
	if raw := c.Query("trip_id"); raw != "" {
		q.TripID = &raw
	}
	out, err := h.svc.ListSavedPlaces(c.Request.Context(), uid, q)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Places ----

// SearchPlaces godoc
// @Summary  Text search over Google Places (cached)
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    q       query  string   true   "search text"
// @Param    lat     query  number   false  "bias latitude"
// @Param    lng     query  number   false  "bias longitude"
// @Param    radius  query  int      false  "bias radius (meters)"
// @Param    limit   query  int      false  "max results"
// @Param    lang    query  string   false  "IETF language code"
// @Success  200  {object}  response.Response{data=[]PlaceSummary}
// @Router   /location/places/search [get]
func (h *Handler) SearchPlaces(c *gin.Context) {
	if _, ok := requireUser(c); !ok {
		return
	}
	q := SearchPlacesQuery{
		Q:      c.Query("q"),
		Lang:   c.Query("lang"),
		Radius: parseInt(c.Query("radius")),
		Limit:  parseInt(c.Query("limit")),
	}
	if lat, ok := parseFloat(c.Query("lat")); ok {
		q.Lat = &lat
	}
	if lng, ok := parseFloat(c.Query("lng")); ok {
		q.Lng = &lng
	}
	out, err := h.svc.SearchPlaces(c.Request.Context(), q)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// NearbyPlaces godoc
// @Summary  Nearby POI search around a lat/lng
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    lat     query  number   true   "latitude"
// @Param    lng     query  number   true   "longitude"
// @Param    radius  query  int      false  "meters (default 1500)"
// @Param    type    query  string   false  "google place type (restaurant, cafe, ...)"
// @Param    limit   query  int      false  "max results"
// @Param    lang    query  string   false  "IETF language code"
// @Success  200  {object}  response.Response{data=[]PlaceSummary}
// @Router   /location/places/nearby [get]
func (h *Handler) NearbyPlaces(c *gin.Context) {
	if _, ok := requireUser(c); !ok {
		return
	}
	lat, ok := parseFloat(c.Query("lat"))
	if !ok {
		response.BadRequest(c, "invalid lat")
		return
	}
	lng, ok := parseFloat(c.Query("lng"))
	if !ok {
		response.BadRequest(c, "invalid lng")
		return
	}
	out, err := h.svc.NearbyPlaces(c.Request.Context(), NearbyPlacesQuery{
		Lat:    lat,
		Lng:    lng,
		Radius: parseInt(c.Query("radius")),
		Type:   c.Query("type"),
		Limit:  parseInt(c.Query("limit")),
		Lang:   c.Query("lang"),
	})
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// PlaceDetails godoc
// @Summary  Google Place Details by place_id
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    place_id  path   string  true   "google place id"
// @Param    lang      query  string  false  "IETF language code"
// @Success  200  {object}  response.Response{data=PlaceDetails}
// @Router   /location/places/{place_id} [get]
func (h *Handler) PlaceDetails(c *gin.Context) {
	if _, ok := requireUser(c); !ok {
		return
	}
	out, err := h.svc.PlaceDetails(c.Request.Context(), c.Param("place_id"), c.Query("lang"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Routes ----

// ComputeRoute godoc
// @Summary  Distance + duration estimate + polyline via Google Directions
// @Tags     location
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      ComputeRoutePayload  true  "origin/destination + optional waypoints"
// @Success  200   {object}  response.Response{data=Route}
// @Router   /location/routes [post]
func (h *Handler) ComputeRoute(c *gin.Context) {
	if _, ok := requireUser(c); !ok {
		return
	}
	var p ComputeRoutePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.ComputeRoute(c.Request.Context(), p)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Weather ----

// Weather godoc
// @Summary  Current + up-to-30-day forecast for a lat/lng via OpenWeather One Call 4.0 (cached 3h)
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    lat    query  number  true   "latitude"
// @Param    lng    query  number  true   "longitude"
// @Param    lang   query  string  false  "language (e.g. zh_tw)"
// @Param    units  query  string  false  "metric | imperial (default metric)"
// @Success  200  {object}  response.Response{data=WeatherForecast}
// @Router   /location/weather [get]
func (h *Handler) Weather(c *gin.Context) {
	if _, ok := requireUser(c); !ok {
		return
	}
	lat, ok := parseFloat(c.Query("lat"))
	if !ok {
		response.BadRequest(c, "invalid lat")
		return
	}
	lng, ok := parseFloat(c.Query("lng"))
	if !ok {
		response.BadRequest(c, "invalid lng")
		return
	}
	out, err := h.svc.Forecast(c.Request.Context(), WeatherQuery{
		Lat:   lat,
		Lng:   lng,
		Lang:  c.Query("lang"),
		Units: strings.ToLower(c.Query("units")),
	})
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Trip member positions ----

// ShareTripPosition godoc
// @Summary  Share the caller's current position within a trip (opt-in, 30-min TTL)
// @Tags     location
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    trip_id  path      string                    true  "trip id"
// @Param    body     body      ShareTripPositionPayload  true  "coords"
// @Success  200  {object}  response.Response
// @Router   /location/trips/{trip_id}/share [post]
func (h *Handler) ShareTripPosition(c *gin.Context) {
	uid, tid, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p ShareTripPositionPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.ShareTripPosition(c.Request.Context(), uid, tid, p); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// RevokeTripPosition godoc
// @Summary  Stop sharing the caller's position for a trip
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path  string  true  "trip id"
// @Success  200  {object}  response.Response
// @Router   /location/trips/{trip_id}/share [delete]
func (h *Handler) RevokeTripPosition(c *gin.Context) {
	uid, tid, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	if err := h.svc.RevokeTripPosition(c.Request.Context(), uid, tid); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListTripPositions godoc
// @Summary  List all trip members currently sharing their position
// @Tags     location
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path  string  true  "trip id"
// @Success  200  {object}  response.Response{data=[]TripMemberPositionResponse}
// @Router   /location/trips/{trip_id}/members [get]
func (h *Handler) ListTripPositions(c *gin.Context) {
	uid, tid, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListTripPositions(c.Request.Context(), uid, tid)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Helpers ----

func (h *Handler) bindUserAndTrip(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	tid, err := uuid.Parse(c.Param("trip_id"))
	if err != nil {
		response.BadRequest(c, "invalid trip id")
		return uuid.Nil, uuid.Nil, false
	}
	return uid, tid, true
}

func (h *Handler) bindUserAndSaved(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid saved place id")
		return uuid.Nil, uuid.Nil, false
	}
	return uid, id, true
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		// Client hung up (e.g. debounced search superseded by the next keystroke).
		// Not a server bug — respond 499 and skip the ERROR log.
		response.Error(c, 499, "client_closed_request", "request canceled")
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrNoResults):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c)
	case errors.Is(err, ErrInvalidPayload):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrProviderMissing):
		// The provider isn't wired (missing / invalid API key). It's not the caller's
		// fault, but nor is it a code bug — surface as 503-equivalent.
		response.Error(c, 503, "service_unavailable", "location provider not configured")
	default:
		h.logger.Error("location handler: internal error", zap.Error(err))
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

func parseFloat(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}
