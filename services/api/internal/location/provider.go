package location

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrProviderUnavailable = errors.New("location provider unavailable")
	ErrProviderNoResult    = errors.New("location provider returned no result")
)

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ---- Places ----

type PlaceSearchRequest struct {
	Query  string
	Near   *LatLng
	Radius int // meters; 0 = no bias
	Limit  int
	Lang   string
}

type NearbySearchRequest struct {
	Center LatLng
	Radius int
	Type   string
	Limit  int
	Lang   string
}

type PlaceSummary struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	Address          string   `json:"address,omitempty"`
	Location         LatLng   `json:"location"`
	Rating           float64  `json:"rating,omitempty"`
	UserRatingsTotal int      `json:"user_ratings_total,omitempty"`
	Types            []string `json:"types,omitempty"`
	PriceLevel       int      `json:"price_level,omitempty"`
	Icon             string   `json:"icon,omitempty"`
}

type PlaceDetails struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	Address          string   `json:"address,omitempty"`
	Location         LatLng   `json:"location"`
	Rating           float64  `json:"rating,omitempty"`
	UserRatingsTotal int      `json:"user_ratings_total,omitempty"`
	Types            []string `json:"types,omitempty"`
	PriceLevel       int      `json:"price_level,omitempty"`
	PhoneNumber      string   `json:"phone_number,omitempty"`
	Website          string   `json:"website,omitempty"`
	OpeningHours     []string `json:"opening_hours,omitempty"`
	Photos           []string `json:"photos,omitempty"`
}

type PlacesProvider interface {
	SearchText(ctx context.Context, req PlaceSearchRequest) ([]PlaceSummary, error)
	Nearby(ctx context.Context, req NearbySearchRequest) ([]PlaceSummary, error)
	Details(ctx context.Context, placeID, lang string) (*PlaceDetails, error)
}

// ---- Routes ----

type TravelMode string

const (
	ModeDriving   TravelMode = "driving"
	ModeWalking   TravelMode = "walking"
	ModeBicycling TravelMode = "bicycling"
	ModeTransit   TravelMode = "transit"
)

func (m TravelMode) Valid() bool {
	switch m {
	case ModeDriving, ModeWalking, ModeBicycling, ModeTransit:
		return true
	}
	return false
}

type RouteRequest struct {
	Origin      LatLng
	Destination LatLng
	Waypoints   []LatLng
	Mode        TravelMode
	Lang        string
}

type RouteStep struct {
	DistanceMeters  int    `json:"distance_meters"`
	DurationSeconds int    `json:"duration_seconds"`
	Instruction     string `json:"instruction"`
	StartLocation   LatLng `json:"start_location"`
	EndLocation     LatLng `json:"end_location"`
}

type RouteLeg struct {
	DistanceMeters  int         `json:"distance_meters"`
	DurationSeconds int         `json:"duration_seconds"`
	StartAddress    string      `json:"start_address,omitempty"`
	EndAddress      string      `json:"end_address,omitempty"`
	StartLocation   LatLng      `json:"start_location"`
	EndLocation     LatLng      `json:"end_location"`
	Steps           []RouteStep `json:"steps,omitempty"`
}

type Route struct {
	Summary         string     `json:"summary,omitempty"`
	Mode            TravelMode `json:"mode"`
	DistanceMeters  int        `json:"distance_meters"`
	DurationSeconds int        `json:"duration_seconds"`
	Polyline        string     `json:"polyline,omitempty"`
	Legs            []RouteLeg `json:"legs,omitempty"`
}

type RouteProvider interface {
	ComputeRoute(ctx context.Context, req RouteRequest) (*Route, error)
}

// ---- Weather ----

type WeatherPoint struct {
	Time     time.Time `json:"time"`
	TempC    float64   `json:"temp_c"`
	FeelsLC  float64   `json:"feels_like_c,omitempty"`
	Humidity int       `json:"humidity,omitempty"`
	WindKph  float64   `json:"wind_kph,omitempty"`
	Icon     string    `json:"icon,omitempty"`
	Summary  string    `json:"summary,omitempty"`
	PopPct   float64   `json:"pop_pct,omitempty"`
}

type WeatherDay struct {
	Date     time.Time `json:"date"`
	TempMinC float64   `json:"temp_min_c"`
	TempMaxC float64   `json:"temp_max_c"`
	Icon     string    `json:"icon,omitempty"`
	Summary  string    `json:"summary,omitempty"`
	PopPct   float64   `json:"pop_pct,omitempty"`
}

type WeatherForecast struct {
	Location LatLng         `json:"location"`
	Timezone string         `json:"timezone,omitempty"`
	Current  *WeatherPoint  `json:"current,omitempty"`
	Hourly   []WeatherPoint `json:"hourly,omitempty"`
	Daily    []WeatherDay   `json:"daily,omitempty"`
}

type WeatherProvider interface {
	Forecast(ctx context.Context, at LatLng, lang, units string) (*WeatherForecast, error)
}

// ---- Cache helpers ----

type Cache struct {
	rdb    *redis.Client
	prefix string
}

func NewCache(rdb *redis.Client, prefix string) *Cache {
	return &Cache{rdb: rdb, prefix: prefix}
}

// Get returns (found=true, err=nil) on cache hit, (false, nil) on miss, or
// (false, err) on infra failure. Callers should treat infra failure as "miss"
// unless the failure mode matters — the location endpoints just refetch.
func (c *Cache) Get(ctx context.Context, key string, out any) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, nil
	}
	raw, err := c.rdb.Get(ctx, c.prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	buf, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.prefix+key, buf, ttl).Err()
}

// hashKey turns arbitrary inputs into a stable short cache-key suffix.
func hashKey(parts ...string) string {
	h := sha1.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0x1f})
	}
	return hex.EncodeToString(h.Sum(nil))
}
