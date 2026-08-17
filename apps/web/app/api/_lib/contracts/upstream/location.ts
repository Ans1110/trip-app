import "server-only";

export type UpstreamLatLng = { lat: number; lng: number };

export type UpstreamSavedPlace = {
  id: string;
  user_id: string;
  trip_id?: string;
  name: string;
  address?: string;
  lat: number;
  lng: number;
  place_id?: string;
  category?: string;
  notes?: string;
  color?: string;
  created_at: string;
  updated_at: string;
};

export type UpstreamCreateSavedPlacePayload = {
  name: string;
  address?: string;
  lat: number;
  lng: number;
  place_id?: string;
  category?: string;
  notes?: string;
  color?: string;
  trip_id?: string;
};

export type UpstreamUpdateSavedPlacePayload = {
  name?: string;
  address?: string;
  lat?: number;
  lng?: number;
  place_id?: string;
  category?: string;
  notes?: string;
  color?: string;
  trip_id?: string;
  unset_trip_id?: boolean;
};

export type UpstreamPlaceSummary = {
  place_id: string;
  name: string;
  address?: string;
  location: UpstreamLatLng;
  rating?: number;
  user_ratings_total?: number;
  types?: string[];
  price_level?: number;
  icon?: string;
};

export type UpstreamPlaceDetails = UpstreamPlaceSummary & {
  phone_number?: string;
  website?: string;
  opening_hours?: string[];
  photos?: string[];
};

export type UpstreamTravelMode =
  | "driving"
  | "walking"
  | "bicycling"
  | "transit";

export type UpstreamRouteStep = {
  distance_meters: number;
  duration_seconds: number;
  instruction: string;
  start_location: UpstreamLatLng;
  end_location: UpstreamLatLng;
};

export type UpstreamRouteLeg = {
  distance_meters: number;
  duration_seconds: number;
  start_address?: string;
  end_address?: string;
  start_location: UpstreamLatLng;
  end_location: UpstreamLatLng;
  steps?: UpstreamRouteStep[];
};

export type UpstreamRoute = {
  summary?: string;
  mode: UpstreamTravelMode;
  distance_meters: number;
  duration_seconds: number;
  polyline?: string;
  legs?: UpstreamRouteLeg[];
};

export type UpstreamComputeRoutePayload = {
  origin: UpstreamLatLng;
  destination: UpstreamLatLng;
  waypoints?: UpstreamLatLng[];
  mode?: UpstreamTravelMode;
  lang?: string;
};

export type UpstreamWeatherPoint = {
  time: string;
  temp_c: number;
  feels_like_c?: number;
  humidity?: number;
  wind_kph?: number;
  icon?: string;
  summary?: string;
  pop_pct?: number;
};

export type UpstreamWeatherDay = {
  date: string;
  temp_min_c: number;
  temp_max_c: number;
  icon?: string;
  summary?: string;
  pop_pct?: number;
};

export type UpstreamWeatherForecast = {
  location: UpstreamLatLng;
  timezone?: string;
  current?: UpstreamWeatherPoint;
  hourly?: UpstreamWeatherPoint[];
  daily?: UpstreamWeatherDay[];
};

export type UpstreamSearchPlacesQuery = {
  q: string;
  lat?: number;
  lng?: number;
  radius?: number;
  limit?: number;
  lang?: string;
};

export type UpstreamNearbyPlacesQuery = {
  lat: number;
  lng: number;
  radius?: number;
  type?: string;
  limit?: number;
  lang?: string;
};

export type UpstreamListSavedPlacesQuery = {
  trip_id?: string;
  category?: string;
};

export type UpstreamWeatherQuery = {
  lat: number;
  lng: number;
  lang?: string;
  units?: "metric" | "imperial";
};

export type UpstreamShareTripPositionPayload = {
  lat: number;
  lng: number;
  accuracy_m?: number;
};

export type UpstreamTripMemberPosition = {
  user_id: string;
  lat: number;
  lng: number;
  accuracy_m?: number;
  shared_at: string;
};
