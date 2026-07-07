"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  calendarApi,
  type CalendarEvent,
  type CreateEventInput,
  type ListEventsQuery,
  type UpdateEventInput,
} from "@/lib/calendar-api";
import { ApiError } from "@/lib/friend-api";

export const calendarKeys = {
  all: ["calendar"] as const,
  mine: (query: ListEventsQuery) => ["calendar", "mine", query] as const,
  detail: (id: string) => ["calendar", "detail", id] as const,
  trip: (tripId: string, query: ListEventsQuery) =>
    ["calendar", "trip", tripId, query] as const,
  friend: (friendId: string, query: ListEventsQuery) =>
    ["calendar", "friend", friendId, query] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useMyEvents(
  query: ListEventsQuery = {},
  options?: QueryOpts<CalendarEvent[]>,
) {
  return useQuery<CalendarEvent[], ApiError>({
    queryKey: calendarKeys.mine(query),
    queryFn: ({ signal }) => calendarApi.listMine(query, signal),
    ...options,
  });
}

export function useEvent(
  id: string | undefined,
  options?: QueryOpts<CalendarEvent>,
) {
  return useQuery<CalendarEvent, ApiError>({
    queryKey: calendarKeys.detail(id ?? ""),
    queryFn: ({ signal }) => calendarApi.get(id!, signal),
    enabled: !!id,
    ...options,
  });
}

export function useTripEvents(
  tripId: string | undefined,
  query: ListEventsQuery = {},
  options?: QueryOpts<CalendarEvent[]>,
) {
  return useQuery<CalendarEvent[], ApiError>({
    queryKey: calendarKeys.trip(tripId ?? "", query),
    queryFn: ({ signal }) => calendarApi.listTrip(tripId!, query, signal),
    enabled: !!tripId,
    ...options,
  });
}

export function useFriendEvents(
  friendId: string | undefined,
  query: ListEventsQuery = {},
  options?: QueryOpts<CalendarEvent[]>,
) {
  return useQuery<CalendarEvent[], ApiError>({
    queryKey: calendarKeys.friend(friendId ?? "", query),
    queryFn: ({ signal }) => calendarApi.listFriend(friendId!, query, signal),
    enabled: !!friendId,
    ...options,
  });
}

export function useCreateEvent(
  options?: MutOpts<CalendarEvent, CreateEventInput>,
) {
  const qc = useQueryClient();
  return useMutation<CalendarEvent, ApiError, CreateEventInput>({
    mutationFn: (input) => calendarApi.create(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: calendarKeys.all });
      return options?.onSuccess?.(...args);
    },
  });
}

type UpdateEventVars = { id: string; input: UpdateEventInput };

export function useUpdateEvent(
  options?: MutOpts<CalendarEvent, UpdateEventVars>,
) {
  const qc = useQueryClient();
  return useMutation<CalendarEvent, ApiError, UpdateEventVars>({
    mutationFn: ({ id, input }) => calendarApi.update(id, input),
    ...options,
    onSuccess: (data, ...rest) => {
      qc.invalidateQueries({ queryKey: calendarKeys.all });
      qc.invalidateQueries({ queryKey: calendarKeys.detail(data.id) });
      return options?.onSuccess?.(data, ...rest);
    },
  });
}

export function useDeleteEvent(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => calendarApi.remove(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: calendarKeys.all });
      return options?.onSuccess?.(...args);
    },
  });
}

export { errorMessage } from "@/hooks/trip-hooks";
