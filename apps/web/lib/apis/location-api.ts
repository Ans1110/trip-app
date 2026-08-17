import { request } from "../utils";

export type LatLng = { lat: number; lng: number };

export type SavedPlace = {
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

export type CreateSavedPlaceInput = {
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

export type UpdateSavedPlaceInput = {
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

export type PlaceSummary = {
  place_id: string;
  name: string;
  address?: string;
  location: LatLng;
  rating?: number;
  user_ratings_total?: number;
  types?: string[];
  price_level?: number;
  icon?: string;
};

export type PlaceDetails = PlaceSummary & {
  phone_number?: string;
  website?: string;
  opening_hours?: string[];
  photos?: string[];
};

export type TravelMode = "driving" | "walking" | "bicycling" | "transit";

export type RouteStep = {
  distance_meters: number;
  duration_seconds: number;
  instruction: string;
  start_location: LatLng;
  end_location: LatLng;
};

export type RouteLeg = {
  distance_meters: number;
  duration_seconds: number;
  start_address?: string;
  end_address?: string;
  start_location: LatLng;
  end_location: LatLng;
  steps?: RouteStep[];
};

export type Route = {
  summary?: string;
  mode: TravelMode;
  distance_meters: number;
  duration_seconds: number;
  polyline?: string;
  legs?: RouteLeg[];
};

export type ComputeRouteInput = {
  origin: LatLng;
  destination: LatLng;
  waypoints?: LatLng[];
  mode?: TravelMode;
  lang?: string;
};

export type WeatherPoint = {
  time: string;
  temp_c: number;
  feels_like_c?: number;
  humidity?: number;
  wind_kph?: number;
  icon?: string;
  summary?: string;
  pop_pct?: number;
};

export type WeatherDay = {
  date: string;
  temp_min_c: number;
  temp_max_c: number;
  icon?: string;
  summary?: string;
  pop_pct?: number;
};

export type WeatherForecast = {
  location: LatLng;
  timezone?: string;
  current?: WeatherPoint;
  hourly?: WeatherPoint[];
  daily?: WeatherDay[];
};

export type SearchPlacesQuery = {
  q: string;
  lat?: number;
  lng?: number;
  radius?: number;
  limit?: number;
  lang?: string;
};

export type NearbyPlacesQuery = {
  lat: number;
  lng: number;
  radius?: number;
  type?: string;
  limit?: number;
  lang?: string;
};

export type ListSavedPlacesQuery = {
  trip_id?: string;
  category?: string;
};

export type WeatherQuery = {
  lat: number;
  lng: number;
  lang?: string;
  units?: "metric" | "imperial";
};

export type ShareTripPositionInput = {
  lat: number;
  lng: number;
  accuracy_m?: number;
};

export type TripMemberPosition = {
  user_id: string;
  lat: number;
  lng: number;
  accuracy_m?: number;
  shared_at: string;
};

const buildQS = (params: Record<string, string | number | undefined>) => {
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === "") continue;
    qs.set(k, String(v));
  }
  return qs.toString();
};

export const locationApi = {
  listSaved: (query: ListSavedPlacesQuery = {}, signal?: AbortSignal) => {
    const qs = buildQS({
      trip_id: query.trip_id,
      category: query.category,
    });
    const path = qs ? `/api/location/saved?${qs}` : "/api/location/saved";
    return request<SavedPlace[]>(path, { signal });
  },
  getSaved: (id: string, signal?: AbortSignal) =>
    request<SavedPlace>(`/api/location/saved/${encodeURIComponent(id)}`, {
      signal,
    }),
  createSaved: (input: CreateSavedPlaceInput, signal?: AbortSignal) =>
    request<SavedPlace>("/api/location/saved", {
      method: "POST",
      json: input,
      signal,
    }),
  updateSaved: (
    id: string,
    input: UpdateSavedPlaceInput,
    signal?: AbortSignal,
  ) =>
    request<SavedPlace>(`/api/location/saved/${encodeURIComponent(id)}`, {
      method: "PATCH",
      json: input,
      signal,
    }),
  deleteSaved: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/location/saved/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  search: (query: SearchPlacesQuery, signal?: AbortSignal) => {
    const qs = buildQS({ ...query });
    return request<PlaceSummary[]>(`/api/location/places/search?${qs}`, {
      signal,
    });
  },
  nearby: (query: NearbyPlacesQuery, signal?: AbortSignal) => {
    const qs = buildQS({ ...query });
    return request<PlaceSummary[]>(`/api/location/places/nearby?${qs}`, {
      signal,
    });
  },
  details: (placeId: string, lang?: string, signal?: AbortSignal) => {
    const qs = lang ? `?lang=${encodeURIComponent(lang)}` : "";
    return request<PlaceDetails>(
      `/api/location/places/${encodeURIComponent(placeId)}${qs}`,
      { signal },
    );
  },

  route: (input: ComputeRouteInput, signal?: AbortSignal) =>
    request<Route>("/api/location/routes", {
      method: "POST",
      json: input,
      signal,
    }),

  weather: (query: WeatherQuery, signal?: AbortSignal) => {
    const qs = buildQS({ ...query });
    return request<WeatherForecast>(`/api/location/weather?${qs}`, { signal });
  },

  shareTripPosition: (
    tripId: string,
    input: ShareTripPositionInput,
    signal?: AbortSignal,
  ) =>
    request<null>(
      `/api/location/trips/${encodeURIComponent(tripId)}/share`,
      { method: "POST", json: input, signal },
    ),
  revokeTripPosition: (tripId: string, signal?: AbortSignal) =>
    request<null>(
      `/api/location/trips/${encodeURIComponent(tripId)}/share`,
      { method: "DELETE", signal },
    ),
  listTripPositions: (tripId: string, signal?: AbortSignal) =>
    request<TripMemberPosition[]>(
      `/api/location/trips/${encodeURIComponent(tripId)}/members`,
      { signal },
    ),
};
