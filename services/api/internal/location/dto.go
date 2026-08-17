package location

import "time"

// ---- Saved places ----

type CreateSavedPlacePayload struct {
	Name     string  `json:"name" binding:"required,min=1,max=200"`
	Address  string  `json:"address" binding:"max=500"`
	Lat      float64 `json:"lat" binding:"required,min=-90,max=90"`
	Lng      float64 `json:"lng" binding:"required,min=-180,max=180"`
	PlaceID  string  `json:"place_id" binding:"max=200"`
	Category string  `json:"category" binding:"max=32"`
	Notes    string  `json:"notes" binding:"max=2000"`
	Color    string  `json:"color" binding:"max=16"`
	TripID   *string `json:"trip_id" binding:"omitempty,uuid"`
}

type UpdateSavedPlacePayload struct {
	Name        *string  `json:"name" binding:"omitempty,min=1,max=200"`
	Address     *string  `json:"address" binding:"omitempty,max=500"`
	Lat         *float64 `json:"lat" binding:"omitempty,min=-90,max=90"`
	Lng         *float64 `json:"lng" binding:"omitempty,min=-180,max=180"`
	PlaceID     *string  `json:"place_id" binding:"omitempty,max=200"`
	Category    *string  `json:"category" binding:"omitempty,max=32"`
	Notes       *string  `json:"notes" binding:"omitempty,max=2000"`
	Color       *string  `json:"color" binding:"omitempty,max=16"`
	TripID      *string  `json:"trip_id" binding:"omitempty,uuid"`
	UnsetTripID bool     `json:"unset_trip_id"`
}

type ListSavedPlacesQuery struct {
	TripID   *string
	Category *string
}

type SavedPlaceResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TripID    *string   `json:"trip_id,omitempty"`
	Name      string    `json:"name"`
	Address   string    `json:"address,omitempty"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	PlaceID   string    `json:"place_id,omitempty"`
	Category  string    `json:"category,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Color     string    `json:"color,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---- Search / nearby / details ----

type SearchPlacesQuery struct {
	Q      string
	Lat    *float64
	Lng    *float64
	Radius int
	Limit  int
	Lang   string
}

type NearbyPlacesQuery struct {
	Lat    float64
	Lng    float64
	Radius int
	Type   string
	Limit  int
	Lang   string
}

// ---- Routes ----

type ComputeRoutePayload struct {
	Origin      LatLngPayload   `json:"origin" binding:"required"`
	Destination LatLngPayload   `json:"destination" binding:"required"`
	Waypoints   []LatLngPayload `json:"waypoints" binding:"max=10,dive"`
	Mode        string          `json:"mode" binding:"omitempty,oneof=driving walking bicycling transit"`
	Lang        string          `json:"lang" binding:"max=8"`
}

type LatLngPayload struct {
	Lat float64 `json:"lat" binding:"required,min=-90,max=90"`
	Lng float64 `json:"lng" binding:"required,min=-180,max=180"`
}

// ---- Weather ----

type WeatherQuery struct {
	Lat   float64
	Lng   float64
	Lang  string
	Units string
}

// ---- Trip member positions (opt-in live sharing) ----

type ShareTripPositionPayload struct {
	Lat       float64 `json:"lat" binding:"required,min=-90,max=90"`
	Lng       float64 `json:"lng" binding:"required,min=-180,max=180"`
	AccuracyM float64 `json:"accuracy_m" binding:"omitempty,min=0"`
}

type TripMemberPositionResponse struct {
	UserID    string    `json:"user_id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	AccuracyM float64   `json:"accuracy_m,omitempty"`
	SharedAt  time.Time `json:"shared_at"`
}
