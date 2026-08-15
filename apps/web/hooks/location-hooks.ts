"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  locationApi,
  type ComputeRouteInput,
  type CreateSavedPlaceInput,
  type ListSavedPlacesQuery,
  type NearbyPlacesQuery,
  type PlaceDetails,
  type PlaceSummary,
  type Route,
  type SavedPlace,
  type SearchPlacesQuery,
  type ShareTripPositionInput,
  type TripMemberPosition,
  type UpdateSavedPlaceInput,
  type WeatherForecast,
  type WeatherQuery,
} from "@/lib/apis/location-api";
import { ApiError } from "@/lib/utils";

export const locationKeys = {
  all: ["location"] as const,
  savedList: (q: ListSavedPlacesQuery) => ["location", "saved", q] as const,
  savedDetail: (id: string) => ["location", "saved", "detail", id] as const,
  search: (q: SearchPlacesQuery) => ["location", "search", q] as const,
  nearby: (q: NearbyPlacesQuery) => ["location", "nearby", q] as const,
  details: (id: string, lang?: string) =>
    ["location", "places", id, lang ?? ""] as const,
  weather: (q: WeatherQuery) => ["location", "weather", q] as const,
  tripMembers: (tripId: string) =>
    ["location", "trip-members", tripId] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useSavedPlaces(
  query: ListSavedPlacesQuery = {},
  options?: QueryOpts<SavedPlace[]>,
) {
  return useQuery<SavedPlace[], ApiError>({
    queryKey: locationKeys.savedList(query),
    queryFn: ({ signal }) => locationApi.listSaved(query, signal),
    ...options,
  });
}

export function useSavedPlace(
  id: string | undefined,
  options?: QueryOpts<SavedPlace>,
) {
  return useQuery<SavedPlace, ApiError>({
    queryKey: locationKeys.savedDetail(id ?? ""),
    queryFn: ({ signal }) => locationApi.getSaved(id!, signal),
    enabled: !!id,
    ...options,
  });
}

export function useCreateSavedPlace(
  options?: MutOpts<SavedPlace, CreateSavedPlaceInput>,
) {
  const qc = useQueryClient();
  return useMutation<SavedPlace, ApiError, CreateSavedPlaceInput>({
    mutationFn: (input) => locationApi.createSaved(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: locationKeys.all });
      return options?.onSuccess?.(...args);
    },
  });
}

type UpdateSavedVars = { id: string; input: UpdateSavedPlaceInput };

export function useUpdateSavedPlace(
  options?: MutOpts<SavedPlace, UpdateSavedVars>,
) {
  const qc = useQueryClient();
  return useMutation<SavedPlace, ApiError, UpdateSavedVars>({
    mutationFn: ({ id, input }) => locationApi.updateSaved(id, input),
    ...options,
    onSuccess: (data, ...rest) => {
      qc.invalidateQueries({ queryKey: locationKeys.all });
      qc.invalidateQueries({ queryKey: locationKeys.savedDetail(data.id) });
      return options?.onSuccess?.(data, ...rest);
    },
  });
}

export function useDeleteSavedPlace(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => locationApi.deleteSaved(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: locationKeys.all });
      return options?.onSuccess?.(...args);
    },
  });
}

export function useSearchPlaces(
  query: SearchPlacesQuery,
  options?: QueryOpts<PlaceSummary[]>,
) {
  const enabled = !!query.q?.trim();
  return useQuery<PlaceSummary[], ApiError>({
    queryKey: locationKeys.search(query),
    queryFn: ({ signal }) => locationApi.search(query, signal),
    enabled,
    ...options,
  });
}

export function useNearbyPlaces(
  query: NearbyPlacesQuery | null,
  options?: QueryOpts<PlaceSummary[]>,
) {
  return useQuery<PlaceSummary[], ApiError>({
    queryKey: locationKeys.nearby(query ?? { lat: 0, lng: 0 }),
    queryFn: ({ signal }) => locationApi.nearby(query!, signal),
    enabled: !!query,
    ...options,
  });
}

export function usePlaceDetails(
  placeId: string | undefined,
  lang?: string,
  options?: QueryOpts<PlaceDetails>,
) {
  return useQuery<PlaceDetails, ApiError>({
    queryKey: locationKeys.details(placeId ?? "", lang),
    queryFn: ({ signal }) => locationApi.details(placeId!, lang, signal),
    enabled: !!placeId,
    ...options,
  });
}

export function useWeather(
  query: WeatherQuery | null,
  options?: QueryOpts<WeatherForecast>,
) {
  return useQuery<WeatherForecast, ApiError>({
    queryKey: locationKeys.weather(query ?? { lat: 0, lng: 0 }),
    queryFn: ({ signal }) => locationApi.weather(query!, signal),
    enabled: !!query,
    ...options,
  });
}

export function useComputeRoute(options?: MutOpts<Route, ComputeRouteInput>) {
  return useMutation<Route, ApiError, ComputeRouteInput>({
    mutationFn: (input) => locationApi.route(input),
    ...options,
  });
}

// ---- Trip member positions (opt-in live sharing) ----

type ShareTripPositionVars = { tripId: string; input: ShareTripPositionInput };

export function useShareTripPosition(
  options?: MutOpts<null, ShareTripPositionVars>,
) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, ShareTripPositionVars>({
    mutationFn: ({ tripId, input }) =>
      locationApi.shareTripPosition(tripId, input),
    ...options,
    onSuccess: (data, variables, ...rest) => {
      qc.invalidateQueries({
        queryKey: locationKeys.tripMembers(variables.tripId),
      });
      return options?.onSuccess?.(data, variables, ...rest);
    },
  });
}

export function useRevokeTripPosition(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (tripId) => locationApi.revokeTripPosition(tripId),
    ...options,
    onSuccess: (data, variables, ...rest) => {
      qc.invalidateQueries({
        queryKey: locationKeys.tripMembers(variables),
      });
      return options?.onSuccess?.(data, variables, ...rest);
    },
  });
}

export function useTripMemberPositions(
  tripId: string | undefined,
  options?: QueryOpts<TripMemberPosition[]>,
) {
  return useQuery<TripMemberPosition[], ApiError>({
    queryKey: locationKeys.tripMembers(tripId ?? ""),
    queryFn: ({ signal }) => locationApi.listTripPositions(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

export { errorMessage } from "@/hooks/trip-hooks";
