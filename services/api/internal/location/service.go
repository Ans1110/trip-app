package location

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotFound        = errors.New("saved place not found")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidPayload  = errors.New("invalid payload")
	ErrProviderMissing = errors.New("location provider not configured")
	ErrNoResults       = errors.New("no results")
)

const (
	cacheTTLPlacesSearch = 6 * time.Hour
	cacheTTLPlaceDetails = 24 * time.Hour
	cacheTTLRoute        = 12 * time.Hour
	cacheTTLWeather      = 3 * time.Hour
)

type TripAuthorizer interface {
	IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error)
}

type IService interface {
	// Saved places
	CreateSavedPlace(ctx context.Context, userID uuid.UUID, p CreateSavedPlacePayload) (*SavedPlaceResponse, error)
	UpdateSavedPlace(ctx context.Context, userID, id uuid.UUID, p UpdateSavedPlacePayload) (*SavedPlaceResponse, error)
	DeleteSavedPlace(ctx context.Context, userID, id uuid.UUID) error
	GetSavedPlace(ctx context.Context, userID, id uuid.UUID) (*SavedPlaceResponse, error)
	ListSavedPlaces(ctx context.Context, userID uuid.UUID, q ListSavedPlacesQuery) ([]SavedPlaceResponse, error)

	// Google Places / Directions passthroughs
	SearchPlaces(ctx context.Context, q SearchPlacesQuery) ([]PlaceSummary, error)
	NearbyPlaces(ctx context.Context, q NearbyPlacesQuery) ([]PlaceSummary, error)
	PlaceDetails(ctx context.Context, placeID, lang string) (*PlaceDetails, error)
	ComputeRoute(ctx context.Context, p ComputeRoutePayload) (*Route, error)

	// Weather
	Forecast(ctx context.Context, q WeatherQuery) (*WeatherForecast, error)

	// Trip member positions (opt-in real-time sharing)
	ShareTripPosition(ctx context.Context, userID, tripID uuid.UUID, p ShareTripPositionPayload) error
	RevokeTripPosition(ctx context.Context, userID, tripID uuid.UUID) error
	RevokeAllUserPositions(ctx context.Context, userID uuid.UUID) error
	ListTripPositions(ctx context.Context, userID, tripID uuid.UUID) ([]TripMemberPositionResponse, error)
}

type ServiceConfig struct {
	Repo      IRepository
	TripAuth  TripAuthorizer
	Places    PlacesProvider
	Routes    RouteProvider
	Weather   WeatherProvider
	Cache     *Cache
	Positions *PositionStore
	Audit     audit.Writer
	Logger    *zap.Logger
}

type service struct {
	repo      IRepository
	auth      TripAuthorizer
	places    PlacesProvider
	routes    RouteProvider
	weather   WeatherProvider
	cache     *Cache
	positions *PositionStore
	audit     audit.Writer
	logger    *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:      cfg.Repo,
		auth:      cfg.TripAuth,
		places:    cfg.Places,
		routes:    cfg.Routes,
		weather:   cfg.Weather,
		cache:     cfg.Cache,
		positions: cfg.Positions,
		audit:     cfg.Audit,
		logger:    logger.With(zap.String("layer", "location.service")),
	}
}

// ---- Saved places ----

func (s *service) CreateSavedPlace(ctx context.Context, userID uuid.UUID, p CreateSavedPlacePayload) (*SavedPlaceResponse, error) {
	tripID, err := s.resolveTripBinding(ctx, userID, p.TripID)
	if err != nil {
		return nil, err
	}

	sp := &SavedPlace{
		UserID:   userID,
		TripID:   tripID,
		Name:     strings.TrimSpace(p.Name),
		Address:  strings.TrimSpace(p.Address),
		Lat:      p.Lat,
		Lng:      p.Lng,
		PlaceID:  strings.TrimSpace(p.PlaceID),
		Category: strings.TrimSpace(p.Category),
		Notes:    p.Notes,
		Color:    p.Color,
	}
	if err := s.repo.CreateSavedPlace(ctx, sp); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) && sp.PlaceID != "" {
			existing, ferr := s.repo.FindSavedPlaceByUserTripPlaceID(ctx, userID, tripID, sp.PlaceID, sp.Category)
			if ferr != nil {
				return nil, ferr
			}
			if existing != nil {
				return toSavedPlaceResponse(existing), nil
			}
		}
		return nil, err
	}
	s.recordAudit(ctx, AuditSavedPlaceCreated, audit.Success, userID, sp.ID.String(), nil)
	return toSavedPlaceResponse(sp), nil
}

func (s *service) UpdateSavedPlace(ctx context.Context, userID, id uuid.UUID, p UpdateSavedPlacePayload) (*SavedPlaceResponse, error) {
	sp, err := s.repo.FindSavedPlaceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrNotFound
	}
	if sp.UserID != userID {
		return nil, ErrForbidden
	}

	patch := map[string]any{}
	if p.Name != nil {
		v := strings.TrimSpace(*p.Name)
		if v == "" {
			return nil, ErrInvalidPayload
		}
		patch["name"] = v
	}
	if p.Address != nil {
		patch["address"] = strings.TrimSpace(*p.Address)
	}
	if p.Lat != nil {
		patch["lat"] = *p.Lat
	}
	if p.Lng != nil {
		patch["lng"] = *p.Lng
	}
	if p.PlaceID != nil {
		patch["place_id"] = strings.TrimSpace(*p.PlaceID)
	}
	if p.Category != nil {
		patch["category"] = strings.TrimSpace(*p.Category)
	}
	if p.Notes != nil {
		patch["notes"] = *p.Notes
	}
	if p.Color != nil {
		patch["color"] = *p.Color
	}
	switch {
	case p.UnsetTripID:
		patch["trip_id"] = nil
	case p.TripID != nil:
		tid, err := s.resolveTripBinding(ctx, userID, p.TripID)
		if err != nil {
			return nil, err
		}
		if tid != nil {
			patch["trip_id"] = *tid
		}
	}

	if len(patch) == 0 {
		return toSavedPlaceResponse(sp), nil
	}
	if err := s.repo.UpdateSavedPlace(ctx, id, patch); err != nil {
		return nil, err
	}
	updated, err := s.repo.FindSavedPlaceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrNotFound
	}
	s.recordAudit(ctx, AuditSavedPlaceUpdated, audit.Success, userID, id.String(), nil)
	return toSavedPlaceResponse(updated), nil
}

func (s *service) DeleteSavedPlace(ctx context.Context, userID, id uuid.UUID) error {
	sp, err := s.repo.FindSavedPlaceByID(ctx, id)
	if err != nil {
		return err
	}
	if sp == nil {
		return ErrNotFound
	}
	if sp.UserID != userID {
		return ErrForbidden
	}
	if err := s.repo.SoftDeleteSavedPlace(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, AuditSavedPlaceDeleted, audit.Success, userID, id.String(), nil)
	return nil
}

func (s *service) GetSavedPlace(ctx context.Context, userID, id uuid.UUID) (*SavedPlaceResponse, error) {
	sp, err := s.repo.FindSavedPlaceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrNotFound
	}
	if sp.UserID != userID {
		return nil, ErrForbidden
	}
	return toSavedPlaceResponse(sp), nil
}

func (s *service) ListSavedPlaces(ctx context.Context, userID uuid.UUID, q ListSavedPlacesQuery) ([]SavedPlaceResponse, error) {
	var tripID *uuid.UUID
	if q.TripID != nil && *q.TripID != "" {
		tid, err := uuid.Parse(*q.TripID)
		if err != nil {
			return nil, ErrInvalidPayload
		}
		if err := s.ensureTripMember(ctx, userID, tid); err != nil {
			return nil, err
		}
		tripID = &tid
	}
	var category *string
	if q.Category != nil {
		v := strings.TrimSpace(*q.Category)
		category = &v
	}
	rows, err := s.repo.ListSavedPlacesForUser(ctx, userID, tripID, category)
	if err != nil {
		return nil, err
	}
	return toSavedPlaceResponses(rows), nil
}

func (s *service) resolveTripBinding(ctx context.Context, userID uuid.UUID, raw *string) (*uuid.UUID, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	tid, err := uuid.Parse(*raw)
	if err != nil {
		return nil, ErrInvalidPayload
	}
	if err := s.ensureTripMember(ctx, userID, tid); err != nil {
		return nil, err
	}
	return &tid, nil
}

func (s *service) ensureTripMember(ctx context.Context, userID, tripID uuid.UUID) error {
	if s.auth == nil {
		return ErrForbidden
	}
	ok, err := s.auth.IsRoomMember(ctx, tripID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// ---- Places (cached passthrough) ----

func (s *service) SearchPlaces(ctx context.Context, q SearchPlacesQuery) ([]PlaceSummary, error) {
	if s.places == nil {
		return nil, ErrProviderMissing
	}
	req := PlaceSearchRequest{
		Query:  strings.TrimSpace(q.Q),
		Radius: q.Radius,
		Limit:  q.Limit,
		Lang:   q.Lang,
	}
	if req.Query == "" {
		return nil, ErrInvalidPayload
	}
	if q.Lat != nil && q.Lng != nil {
		req.Near = &LatLng{Lat: *q.Lat, Lng: *q.Lng}
	}

	key := "search:" + hashKey(req.Query, formatOptionalLatLng(req.Near), fmt.Sprint(req.Radius), fmt.Sprint(req.Limit), req.Lang)
	var cached []PlaceSummary
	if hit, _ := s.cache.Get(ctx, key, &cached); hit {
		return cached, nil
	}

	out, err := s.places.SearchText(ctx, req)
	if err != nil {
		return nil, mapProviderErr(err)
	}
	_ = s.cache.Set(ctx, key, out, cacheTTLPlacesSearch)
	return out, nil
}

func (s *service) NearbyPlaces(ctx context.Context, q NearbyPlacesQuery) ([]PlaceSummary, error) {
	if s.places == nil {
		return nil, ErrProviderMissing
	}
	req := NearbySearchRequest{
		Center: LatLng{Lat: q.Lat, Lng: q.Lng},
		Radius: q.Radius,
		Type:   strings.TrimSpace(q.Type),
		Limit:  q.Limit,
		Lang:   q.Lang,
	}
	key := "nearby:" + hashKey(fmt.Sprintf("%f,%f", req.Center.Lat, req.Center.Lng), fmt.Sprint(req.Radius), req.Type, fmt.Sprint(req.Limit), req.Lang)
	var cached []PlaceSummary
	if hit, _ := s.cache.Get(ctx, key, &cached); hit {
		return cached, nil
	}

	out, err := s.places.Nearby(ctx, req)
	if err != nil {
		return nil, mapProviderErr(err)
	}
	_ = s.cache.Set(ctx, key, out, cacheTTLPlacesSearch)
	return out, nil
}

func (s *service) PlaceDetails(ctx context.Context, placeID, lang string) (*PlaceDetails, error) {
	if s.places == nil {
		return nil, ErrProviderMissing
	}
	placeID = strings.TrimSpace(placeID)
	if placeID == "" {
		return nil, ErrInvalidPayload
	}
	key := "details:" + hashKey(placeID, lang)
	var cached PlaceDetails
	if hit, _ := s.cache.Get(ctx, key, &cached); hit {
		return &cached, nil
	}
	out, err := s.places.Details(ctx, placeID, lang)
	if err != nil {
		return nil, mapProviderErr(err)
	}
	_ = s.cache.Set(ctx, key, out, cacheTTLPlaceDetails)
	return out, nil
}

// ---- Routes ----

func (s *service) ComputeRoute(ctx context.Context, p ComputeRoutePayload) (*Route, error) {
	if s.routes == nil {
		return nil, ErrProviderMissing
	}
	mode := TravelMode(strings.ToLower(p.Mode))
	if mode == "" {
		mode = ModeDriving
	}
	if !mode.Valid() {
		return nil, ErrInvalidPayload
	}
	req := RouteRequest{
		Origin:      LatLng{Lat: p.Origin.Lat, Lng: p.Origin.Lng},
		Destination: LatLng{Lat: p.Destination.Lat, Lng: p.Destination.Lng},
		Mode:        mode,
		Lang:        p.Lang,
	}
	for _, wp := range p.Waypoints {
		req.Waypoints = append(req.Waypoints, LatLng{Lat: wp.Lat, Lng: wp.Lng})
	}

	key := "route:" + hashKey(
		fmt.Sprintf("%f,%f", req.Origin.Lat, req.Origin.Lng),
		fmt.Sprintf("%f,%f", req.Destination.Lat, req.Destination.Lng),
		waypointsKey(req.Waypoints),
		string(req.Mode),
		req.Lang,
	)
	var cached Route
	if hit, _ := s.cache.Get(ctx, key, &cached); hit {
		return &cached, nil
	}

	out, err := s.routes.ComputeRoute(ctx, req)
	if err != nil {
		return nil, mapProviderErr(err)
	}
	_ = s.cache.Set(ctx, key, out, cacheTTLRoute)
	return out, nil
}

// ---- Weather ----

func (s *service) Forecast(ctx context.Context, q WeatherQuery) (*WeatherForecast, error) {
	if s.weather == nil {
		return nil, ErrProviderMissing
	}
	at := LatLng{Lat: q.Lat, Lng: q.Lng}
	units := q.Units
	if units == "" {
		units = "metric"
	}
	// v4: hourly source switched from 2.5/forecast (3h buckets) to
	// One Call 4.0 /timeline/1h (true 1-hour granularity).
	key := "weather:v4:" + hashKey(fmt.Sprintf("%f,%f", at.Lat, at.Lng), q.Lang, units)
	var cached WeatherForecast
	if hit, _ := s.cache.Get(ctx, key, &cached); hit {
		return &cached, nil
	}
	out, err := s.weather.Forecast(ctx, at, q.Lang, units)
	if err != nil {
		return nil, mapProviderErr(err)
	}
	_ = s.cache.Set(ctx, key, out, cacheTTLWeather)
	return out, nil
}

// ---- Trip member positions ----

func (s *service) ShareTripPosition(ctx context.Context, userID, tripID uuid.UUID, p ShareTripPositionPayload) error {
	if s.positions == nil {
		return ErrProviderMissing
	}
	if err := s.ensureTripMember(ctx, userID, tripID); err != nil {
		return err
	}
	pos := MemberPosition{
		UserID:    userID,
		Lat:       p.Lat,
		Lng:       p.Lng,
		AccuracyM: p.AccuracyM,
		SharedAt:  time.Now().UTC(),
	}
	return s.positions.Set(ctx, tripID, pos)
}

func (s *service) RevokeTripPosition(ctx context.Context, userID, tripID uuid.UUID) error {
	if s.positions == nil {
		return ErrProviderMissing
	}
	if err := s.ensureTripMember(ctx, userID, tripID); err != nil {
		return err
	}
	return s.positions.Delete(ctx, tripID, userID)
}

func (s *service) RevokeAllUserPositions(ctx context.Context, userID uuid.UUID) error {
	if s.positions == nil {
		return nil
	}
	return s.positions.DeleteAllForUser(ctx, userID)
}

func (s *service) ListTripPositions(ctx context.Context, userID, tripID uuid.UUID) ([]TripMemberPositionResponse, error) {
	if s.positions == nil {
		return nil, ErrProviderMissing
	}
	if err := s.ensureTripMember(ctx, userID, tripID); err != nil {
		return nil, err
	}
	rows, err := s.positions.ListForTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]TripMemberPositionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, TripMemberPositionResponse{
			UserID:    r.UserID.String(),
			Lat:       r.Lat,
			Lng:       r.Lng,
			AccuracyM: r.AccuracyM,
			SharedAt:  r.SharedAt,
		})
	}
	return out, nil
}

// ---- Helpers ----

func mapProviderErr(err error) error {
	switch {
	case errors.Is(err, ErrProviderUnavailable):
		return ErrProviderMissing
	case errors.Is(err, ErrProviderNoResult):
		return ErrNoResults
	default:
		return err
	}
}

func formatOptionalLatLng(p *LatLng) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%f,%f", p.Lat, p.Lng)
}

func waypointsKey(wps []LatLng) string {
	if len(wps) == 0 {
		return ""
	}
	parts := make([]string, 0, len(wps))
	for _, wp := range wps {
		parts = append(parts, fmt.Sprintf("%f,%f", wp.Lat, wp.Lng))
	}
	return strings.Join(parts, "|")
}

func (s *service) recordAudit(
	ctx context.Context,
	action audit.Action,
	status audit.Status,
	actorID uuid.UUID,
	resourceID string,
	detail map[string]any,
) {
	if s.audit == nil {
		return
	}
	log := &audit.Log{
		ID:           uuid.New(),
		Action:       action,
		Status:       status,
		ResourceType: AuditResourceSavedPlace,
		ResourceID:   resourceID,
		TraceID:      middleware.GetTraceID(ctx),
		Detail:       detail,
	}
	if actorID != uuid.Nil {
		a := actorID
		log.ActorUserID = &a
	}
	if rid := middleware.RequestIDFromContext(ctx); rid != uuid.Nil {
		log.RequestID = &rid
	}
	meta := auditMetaFromCtx(ctx)
	if meta.IPAddress != "" {
		ip := meta.IPAddress
		log.IPAddress = &ip
	}
	if meta.UserAgent != "" {
		log.UserAgent = meta.UserAgent
	}
	if err := s.audit.Create(ctx, log); err != nil {
		s.logger.Warn("audit log write failed",
			zap.Error(err),
			zap.String("action", string(action)),
		)
	}
}

// ---- Mappers ----

func toSavedPlaceResponse(sp *SavedPlace) *SavedPlaceResponse {
	resp := &SavedPlaceResponse{
		ID:        sp.ID.String(),
		UserID:    sp.UserID.String(),
		Name:      sp.Name,
		Address:   sp.Address,
		Lat:       sp.Lat,
		Lng:       sp.Lng,
		PlaceID:   sp.PlaceID,
		Category:  sp.Category,
		Notes:     sp.Notes,
		Color:     sp.Color,
		CreatedAt: sp.CreatedAt,
		UpdatedAt: sp.UpdatedAt,
	}
	if sp.TripID != nil {
		t := sp.TripID.String()
		resp.TripID = &t
	}
	return resp
}

func toSavedPlaceResponses(rows []SavedPlace) []SavedPlaceResponse {
	out := make([]SavedPlaceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, *toSavedPlaceResponse(&rows[i]))
	}
	return out
}
