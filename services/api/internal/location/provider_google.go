package location

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	googlePlacesBase = "https://places.googleapis.com/v1"
	googleRoutesURL  = "https://routes.googleapis.com/directions/v2:computeRoutes"
	googleTimeout    = 8 * time.Second
	googleUserAgent  = "trip-app/1.0"

	placesSearchMask = "places.id,places.displayName,places.formattedAddress,places.location,places.rating,places.userRatingCount,places.types,places.priceLevel,places.iconMaskBaseUri"
	placesDetailMask = "id,displayName,formattedAddress,location,rating,userRatingCount,types,priceLevel,iconMaskBaseUri,internationalPhoneNumber,nationalPhoneNumber,websiteUri,regularOpeningHours,photos"
	routesMask       = "routes.duration,routes.distanceMeters,routes.polyline.encodedPolyline,routes.description,routes.legs"
)

type GoogleMapsProvider struct {
	apiKey string
	client *http.Client
	logger *zap.Logger
}

func NewGoogleMapsProvider(apiKey string, logger *zap.Logger) *GoogleMapsProvider {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GoogleMapsProvider{
		apiKey: strings.TrimSpace(apiKey),
		client: &http.Client{Timeout: googleTimeout},
		logger: logger.With(zap.String("layer", "location.google")),
	}
}

func (g *GoogleMapsProvider) available() bool { return g != nil && g.apiKey != "" }

// ---- Places (New) ----

type googleLatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googleLocalizedText struct {
	Text         string `json:"text"`
	LanguageCode string `json:"languageCode,omitempty"`
}

type googlePlaceNew struct {
	ID                       string              `json:"id"`
	DisplayName              googleLocalizedText `json:"displayName"`
	FormattedAddress         string              `json:"formattedAddress"`
	Location                 googleLatLng        `json:"location"`
	Rating                   float64             `json:"rating"`
	UserRatingCount          int                 `json:"userRatingCount"`
	Types                    []string            `json:"types"`
	PriceLevel               string              `json:"priceLevel"`
	IconMaskBaseURI          string              `json:"iconMaskBaseUri"`
	InternationalPhoneNumber string              `json:"internationalPhoneNumber,omitempty"`
	NationalPhoneNumber      string              `json:"nationalPhoneNumber,omitempty"`
	WebsiteURI               string              `json:"websiteUri,omitempty"`
	RegularOpeningHours      *struct {
		WeekdayDescriptions []string `json:"weekdayDescriptions"`
	} `json:"regularOpeningHours,omitempty"`
	Photos []struct {
		Name string `json:"name"`
	} `json:"photos,omitempty"`
}

type googleSearchResp struct {
	Places []googlePlaceNew `json:"places"`
}

type googleTextSearchReq struct {
	TextQuery      string             `json:"textQuery"`
	LanguageCode   string             `json:"languageCode,omitempty"`
	MaxResultCount int                `json:"maxResultCount,omitempty"`
	LocationBias   *googleLocationCir `json:"locationBias,omitempty"`
}

type googleNearbyReq struct {
	LocationRestriction googleLocationCir `json:"locationRestriction"`
	IncludedTypes       []string          `json:"includedTypes,omitempty"`
	MaxResultCount      int               `json:"maxResultCount,omitempty"`
	LanguageCode        string            `json:"languageCode,omitempty"`
	RankPreference      string            `json:"rankPreference,omitempty"`
}

type googleLocationCir struct {
	Circle googleCircle `json:"circle"`
}

type googleCircle struct {
	Center googleLatLng `json:"center"`
	Radius float64      `json:"radius"`
}

func (g *GoogleMapsProvider) SearchText(ctx context.Context, req PlaceSearchRequest) ([]PlaceSummary, error) {
	if !g.available() {
		return nil, ErrProviderUnavailable
	}
	body := googleTextSearchReq{
		TextQuery:      req.Query,
		LanguageCode:   req.Lang,
		MaxResultCount: clampResultCount(req.Limit),
	}
	if req.Near != nil {
		radius := float64(req.Radius)
		if radius <= 0 {
			radius = 5000
		}
		body.LocationBias = &googleLocationCir{
			Circle: googleCircle{
				Center: googleLatLng{Latitude: req.Near.Lat, Longitude: req.Near.Lng},
				Radius: radius,
			},
		}
	}

	var out googleSearchResp
	if err := g.doJSON(ctx, http.MethodPost, googlePlacesBase+"/places:searchText", placesSearchMask, body, &out); err != nil {
		return nil, err
	}
	if len(out.Places) == 0 {
		return nil, ErrProviderNoResult
	}
	return trimSummaries(mapNewPlaces(out.Places), req.Limit), nil
}

func (g *GoogleMapsProvider) Nearby(ctx context.Context, req NearbySearchRequest) ([]PlaceSummary, error) {
	if !g.available() {
		return nil, ErrProviderUnavailable
	}
	radius := float64(req.Radius)
	if radius <= 0 {
		radius = 1500
	}
	body := googleNearbyReq{
		LocationRestriction: googleLocationCir{
			Circle: googleCircle{
				Center: googleLatLng{Latitude: req.Center.Lat, Longitude: req.Center.Lng},
				Radius: radius,
			},
		},
		MaxResultCount: clampResultCount(req.Limit),
		LanguageCode:   req.Lang,
	}
	if req.Type != "" {
		body.IncludedTypes = []string{req.Type}
	}

	var out googleSearchResp
	if err := g.doJSON(ctx, http.MethodPost, googlePlacesBase+"/places:searchNearby", placesSearchMask, body, &out); err != nil {
		return nil, err
	}
	if len(out.Places) == 0 {
		return nil, ErrProviderNoResult
	}
	return trimSummaries(mapNewPlaces(out.Places), req.Limit), nil
}

func (g *GoogleMapsProvider) Details(ctx context.Context, placeID, lang string) (*PlaceDetails, error) {
	if !g.available() {
		return nil, ErrProviderUnavailable
	}
	url := googlePlacesBase + "/places/" + placeID
	if lang != "" {
		url += "?languageCode=" + lang
	}

	var p googlePlaceNew
	if err := g.doJSON(ctx, http.MethodGet, url, placesDetailMask, nil, &p); err != nil {
		return nil, err
	}
	if p.ID == "" {
		return nil, ErrProviderNoResult
	}

	pd := &PlaceDetails{
		PlaceID:          p.ID,
		Name:             p.DisplayName.Text,
		Address:          p.FormattedAddress,
		Location:         LatLng{Lat: p.Location.Latitude, Lng: p.Location.Longitude},
		Rating:           p.Rating,
		UserRatingsTotal: p.UserRatingCount,
		Types:            p.Types,
		PriceLevel:       mapPriceLevel(p.PriceLevel),
	}
	if p.InternationalPhoneNumber != "" {
		pd.PhoneNumber = p.InternationalPhoneNumber
	} else {
		pd.PhoneNumber = p.NationalPhoneNumber
	}
	pd.Website = p.WebsiteURI
	if p.RegularOpeningHours != nil {
		pd.OpeningHours = p.RegularOpeningHours.WeekdayDescriptions
	}
	for _, ph := range p.Photos {
		if ph.Name != "" {
			pd.Photos = append(pd.Photos, ph.Name)
		}
	}
	return pd, nil
}

// ---- Routes (New) ----

type googleComputeRoutesReq struct {
	Origin                   googleRouteWaypoint   `json:"origin"`
	Destination              googleRouteWaypoint   `json:"destination"`
	Intermediates            []googleRouteWaypoint `json:"intermediates,omitempty"`
	TravelMode               string                `json:"travelMode"`
	RoutingPreference        string                `json:"routingPreference,omitempty"`
	ComputeAlternativeRoutes bool                  `json:"computeAlternativeRoutes"`
	LanguageCode             string                `json:"languageCode,omitempty"`
	Units                    string                `json:"units,omitempty"`
}

type googleRouteWaypoint struct {
	Location googleRouteLocation `json:"location"`
}

type googleRouteLocation struct {
	LatLng googleLatLng `json:"latLng"`
}

type googleRoutesResp struct {
	Routes []googleRouteEntry `json:"routes"`
}

type googleRouteEntry struct {
	DistanceMeters int              `json:"distanceMeters"`
	Duration       string           `json:"duration"`
	Description    string           `json:"description"`
	Polyline       *googlePolyline  `json:"polyline"`
	Legs           []googleLegEntry `json:"legs"`
}

type googlePolyline struct {
	EncodedPolyline string `json:"encodedPolyline"`
}

type googleLegEntry struct {
	DistanceMeters int                    `json:"distanceMeters"`
	Duration       string                 `json:"duration"`
	StaticDuration string                 `json:"staticDuration"`
	StartLocation  googleRouteLocation    `json:"startLocation"`
	EndLocation    googleRouteLocation    `json:"endLocation"`
	Steps          []googleRouteStepEntry `json:"steps"`
}

type googleRouteStepEntry struct {
	DistanceMeters        int                 `json:"distanceMeters"`
	StaticDuration        string              `json:"staticDuration"`
	StartLocation         googleRouteLocation `json:"startLocation"`
	EndLocation           googleRouteLocation `json:"endLocation"`
	NavigationInstruction *struct {
		Instructions string `json:"instructions"`
	} `json:"navigationInstruction,omitempty"`
}

func (g *GoogleMapsProvider) ComputeRoute(ctx context.Context, req RouteRequest) (*Route, error) {
	if !g.available() {
		return nil, ErrProviderUnavailable
	}
	mode := req.Mode
	if !mode.Valid() {
		mode = ModeDriving
	}
	body := googleComputeRoutesReq{
		Origin:       waypointOf(req.Origin),
		Destination:  waypointOf(req.Destination),
		TravelMode:   travelModeToNew(mode),
		LanguageCode: req.Lang,
	}
	if mode == ModeDriving {
		body.RoutingPreference = "TRAFFIC_UNAWARE"
	}
	if len(req.Waypoints) > 0 {
		body.Intermediates = make([]googleRouteWaypoint, 0, len(req.Waypoints))
		for _, wp := range req.Waypoints {
			body.Intermediates = append(body.Intermediates, waypointOf(wp))
		}
	}

	var out googleRoutesResp
	if err := g.doJSON(ctx, http.MethodPost, googleRoutesURL, routesMask, body, &out); err != nil {
		return nil, err
	}
	if len(out.Routes) == 0 {
		return nil, ErrProviderNoResult
	}
	primary := out.Routes[0]

	route := &Route{
		Summary:         primary.Description,
		Mode:            mode,
		DistanceMeters:  primary.DistanceMeters,
		DurationSeconds: parseDurationSeconds(primary.Duration),
	}
	if primary.Polyline != nil {
		route.Polyline = primary.Polyline.EncodedPolyline
	}
	for _, leg := range primary.Legs {
		mapped := RouteLeg{
			DistanceMeters:  leg.DistanceMeters,
			DurationSeconds: parseDurationSeconds(leg.Duration),
			StartLocation:   LatLng{Lat: leg.StartLocation.LatLng.Latitude, Lng: leg.StartLocation.LatLng.Longitude},
			EndLocation:     LatLng{Lat: leg.EndLocation.LatLng.Latitude, Lng: leg.EndLocation.LatLng.Longitude},
		}
		for _, step := range leg.Steps {
			instr := ""
			if step.NavigationInstruction != nil {
				instr = step.NavigationInstruction.Instructions
			}
			mapped.Steps = append(mapped.Steps, RouteStep{
				DistanceMeters:  step.DistanceMeters,
				DurationSeconds: parseDurationSeconds(step.StaticDuration),
				Instruction:     instr,
				StartLocation:   LatLng{Lat: step.StartLocation.LatLng.Latitude, Lng: step.StartLocation.LatLng.Longitude},
				EndLocation:     LatLng{Lat: step.EndLocation.LatLng.Latitude, Lng: step.EndLocation.LatLng.Longitude},
			})
		}
		route.Legs = append(route.Legs, mapped)
	}
	return route, nil
}

// ---- HTTP + helpers ----

func (g *GoogleMapsProvider) doJSON(ctx context.Context, method, url, fieldMask string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", googleUserAgent)
	req.Header.Set("X-Goog-Api-Key", g.apiKey)
	req.Header.Set("X-Goog-FieldMask", fieldMask)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("google maps: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		msg := extractErrorMessage(res.StatusCode, raw)
		if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
			g.logger.Warn("google maps auth failed", zap.Int("status", res.StatusCode), zap.String("body", string(raw)))
			return fmt.Errorf("google maps %s: %w", msg, ErrProviderUnavailable)
		}
		return fmt.Errorf("google maps: %s", msg)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func extractErrorMessage(status int, raw []byte) string {
	var envelope struct {
		Error struct {
			Code    int    `json:"code"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Error.Message != "" {
		s := envelope.Error.Status
		if s == "" {
			s = fmt.Sprintf("http %d", status)
		}
		return fmt.Sprintf("%s: %s", s, envelope.Error.Message)
	}
	return fmt.Sprintf("http %d: %s", status, string(raw))
}

func mapNewPlaces(in []googlePlaceNew) []PlaceSummary {
	out := make([]PlaceSummary, 0, len(in))
	for _, p := range in {
		out = append(out, PlaceSummary{
			PlaceID:          p.ID,
			Name:             p.DisplayName.Text,
			Address:          p.FormattedAddress,
			Location:         LatLng{Lat: p.Location.Latitude, Lng: p.Location.Longitude},
			Rating:           p.Rating,
			UserRatingsTotal: p.UserRatingCount,
			Types:            p.Types,
			PriceLevel:       mapPriceLevel(p.PriceLevel),
			Icon:             p.IconMaskBaseURI,
		})
	}
	return out
}

func trimSummaries(rows []PlaceSummary, limit int) []PlaceSummary {
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

// Places API (New) hard-caps at 20 results per request.
func clampResultCount(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 20 {
		return 20
	}
	return limit
}

func mapPriceLevel(s string) int {
	switch s {
	case "PRICE_LEVEL_FREE":
		return 0
	case "PRICE_LEVEL_INEXPENSIVE":
		return 1
	case "PRICE_LEVEL_MODERATE":
		return 2
	case "PRICE_LEVEL_EXPENSIVE":
		return 3
	case "PRICE_LEVEL_VERY_EXPENSIVE":
		return 4
	}
	return 0
}

func travelModeToNew(m TravelMode) string {
	switch m {
	case ModeWalking:
		return "WALK"
	case ModeBicycling:
		return "BICYCLE"
	case ModeTransit:
		return "TRANSIT"
	}
	return "DRIVE"
}

func waypointOf(p LatLng) googleRouteWaypoint {
	return googleRouteWaypoint{
		Location: googleRouteLocation{
			LatLng: googleLatLng{Latitude: p.Lat, Longitude: p.Lng},
		},
	}
}

// parseDurationSeconds parses the "3600s" duration string the Routes API
// returns. Anything unparseable becomes 0 rather than erroring out — a route
// with a bad duration is still useful to render.
func parseDurationSeconds(s string) int {
	s = strings.TrimSuffix(strings.TrimSpace(s), "s")
	if s == "" {
		return 0
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return int(v)
	}
	return 0
}
