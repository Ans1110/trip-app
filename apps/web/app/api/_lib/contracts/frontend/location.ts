import "server-only";
import type {
  UpstreamPlaceDetails,
  UpstreamPlaceSummary,
  UpstreamRoute,
  UpstreamSavedPlace,
  UpstreamTravelMode,
  UpstreamTripMemberPosition,
  UpstreamWeatherForecast,
} from "../upstream";

export type SavedPlaceView = UpstreamSavedPlace;
export type PlaceSummaryView = UpstreamPlaceSummary;
export type PlaceDetailsView = UpstreamPlaceDetails;
export type RouteView = UpstreamRoute;
export type WeatherForecastView = UpstreamWeatherForecast;
export type TravelModeView = UpstreamTravelMode;
export type TripMemberPositionView = UpstreamTripMemberPosition;

export const toSavedPlaceView = (p: UpstreamSavedPlace): SavedPlaceView => p;
export const toPlaceSummaryView = (
  p: UpstreamPlaceSummary,
): PlaceSummaryView => p;
export const toPlaceDetailsView = (
  p: UpstreamPlaceDetails,
): PlaceDetailsView => p;
export const toRouteView = (r: UpstreamRoute): RouteView => r;
export const toWeatherForecastView = (
  w: UpstreamWeatherForecast,
): WeatherForecastView => w;
export const toTripMemberPositionView = (
  p: UpstreamTripMemberPosition,
): TripMemberPositionView => p;
