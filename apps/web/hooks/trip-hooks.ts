"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import { ApiError } from "@/lib/friend-api";
import {
  tripApi,
  type CreateItineraryInput,
  type CreateTodoInput,
  type CreateTripInput,
  type Itinerary,
  type JoinRoomInput,
  type JoinRoomResult,
  type ListTripsQuery,
  type ReorderItineraryInput,
  type ReorderTodosInput,
  type Room,
  type RoomPreview,
  type Todo,
  type Trip,
  type UpdateItineraryInput,
  type UpdateTodoInput,
  type UpdateTripInput,
} from "@/lib/trip-api";

export const tripKeys = {
  list: (query: ListTripsQuery) => ["trips", "list", query] as const,
  detail: (id: string) => ["trips", "detail", id] as const,
  room: (id: string) => ["trips", "room", id] as const,
  itinerary: (id: string) => ["trips", "itinerary", id] as const,
  todos: (id: string) => ["trips", "todos", id] as const,
  previewByCode: (code: string) => ["rooms", "preview", code] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

// ---- Trip ----

export function useTrips(
  query: ListTripsQuery = {},
  options?: QueryOpts<Trip[]>,
) {
  return useQuery<Trip[], ApiError>({
    queryKey: tripKeys.list(query),
    queryFn: ({ signal }) => tripApi.list(query, signal),
    ...options,
  });
}

export function useTrip(id: string | undefined, options?: QueryOpts<Trip>) {
  return useQuery<Trip, ApiError>({
    queryKey: tripKeys.detail(id ?? ""),
    queryFn: ({ signal }) => tripApi.get(id!, signal),
    enabled: !!id,
    ...options,
  });
}

export function useCreateTrip(options?: MutOpts<Trip, CreateTripInput>) {
  const qc = useQueryClient();
  return useMutation<Trip, ApiError, CreateTripInput>({
    mutationFn: (input) => tripApi.create(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: ["trips", "list"] });
      return options?.onSuccess?.(...args);
    },
  });
}

type UpdateTripVars = { id: string; input: UpdateTripInput };

export function useUpdateTrip(options?: MutOpts<Trip, UpdateTripVars>) {
  const qc = useQueryClient();
  return useMutation<Trip, ApiError, UpdateTripVars>({
    mutationFn: ({ id, input }) => tripApi.update(id, input),
    ...options,
    onSuccess: (data, ...rest) => {
      qc.invalidateQueries({ queryKey: ["trips", "list"] });
      qc.invalidateQueries({ queryKey: tripKeys.detail(data.id) });
      return options?.onSuccess?.(data, ...rest);
    },
  });
}

export function useDeleteTrip(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => tripApi.remove(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: ["trips", "list"] });
      return options?.onSuccess?.(...args);
    },
  });
}

// ---- Room ----

export function useRoom(id: string | undefined, options?: QueryOpts<Room>) {
  return useQuery<Room, ApiError>({
    queryKey: tripKeys.room(id ?? ""),
    queryFn: ({ signal }) => tripApi.getRoom(id!, signal),
    enabled: !!id,
    ...options,
  });
}

export function useLeaveRoom(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => tripApi.leaveRoom(id),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: ["trips"] });
      return options?.onSuccess?.(...args);
    },
  });
}

type RemoveMemberVars = { id: string; userId: string };

export function useRemoveMember(options?: MutOpts<null, RemoveMemberVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, RemoveMemberVars>({
    mutationFn: ({ id, userId }) => tripApi.removeMember(id, userId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.room(vars.id) });
      qc.invalidateQueries({ queryKey: tripKeys.detail(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

export function useRegenerateCode(options?: MutOpts<Room, string>) {
  const qc = useQueryClient();
  return useMutation<Room, ApiError, string>({
    mutationFn: (id) => tripApi.regenerateCode(id),
    ...options,
    onSuccess: (data, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.room(data.trip_id) });
      qc.invalidateQueries({ queryKey: tripKeys.detail(data.trip_id) });
      return options?.onSuccess?.(data, ...rest);
    },
  });
}

export function useRoomPreviewByCode(
  code: string | undefined,
  options?: QueryOpts<RoomPreview>,
) {
  return useQuery<RoomPreview, ApiError>({
    queryKey: tripKeys.previewByCode(code ?? ""),
    queryFn: ({ signal }) => tripApi.previewByCode(code!, signal),
    enabled: !!code,
    retry: false,
    ...options,
  });
}

export function useJoinRoom(options?: MutOpts<JoinRoomResult, JoinRoomInput>) {
  const qc = useQueryClient();
  return useMutation<JoinRoomResult, ApiError, JoinRoomInput>({
    mutationFn: (input) => tripApi.join(input),
    ...options,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: ["trips", "list"] });
      return options?.onSuccess?.(...args);
    },
  });
}

// ---- Itinerary ----

export function useItinerary(
  id: string | undefined,
  options?: QueryOpts<Itinerary[]>,
) {
  return useQuery<Itinerary[], ApiError>({
    queryKey: tripKeys.itinerary(id ?? ""),
    queryFn: ({ signal }) => tripApi.listItinerary(id!, signal),
    enabled: !!id,
    ...options,
  });
}

type CreateItineraryVars = { id: string; input: CreateItineraryInput };

export function useCreateItinerary(
  options?: MutOpts<Itinerary, CreateItineraryVars>,
) {
  const qc = useQueryClient();
  return useMutation<Itinerary, ApiError, CreateItineraryVars>({
    mutationFn: ({ id, input }) => tripApi.createItinerary(id, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.itinerary(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type UpdateItineraryVars = {
  id: string;
  itemId: string;
  input: UpdateItineraryInput;
};

export function useUpdateItinerary(
  options?: MutOpts<Itinerary, UpdateItineraryVars>,
) {
  const qc = useQueryClient();
  return useMutation<Itinerary, ApiError, UpdateItineraryVars>({
    mutationFn: ({ id, itemId, input }) =>
      tripApi.updateItinerary(id, itemId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.itinerary(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeleteItineraryVars = { id: string; itemId: string };

export function useDeleteItinerary(
  options?: MutOpts<null, DeleteItineraryVars>,
) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeleteItineraryVars>({
    mutationFn: ({ id, itemId }) => tripApi.deleteItinerary(id, itemId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.itinerary(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type ReorderItineraryVars = { id: string; input: ReorderItineraryInput };

export function useReorderItinerary(
  options?: MutOpts<null, ReorderItineraryVars>,
) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, ReorderItineraryVars>({
    mutationFn: ({ id, input }) => tripApi.reorderItinerary(id, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.itinerary(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

// ---- Todos ----

export function useTodos(
  id: string | undefined,
  options?: QueryOpts<Todo[]>,
) {
  return useQuery<Todo[], ApiError>({
    queryKey: tripKeys.todos(id ?? ""),
    queryFn: ({ signal }) => tripApi.listTodos(id!, signal),
    enabled: !!id,
    ...options,
  });
}

type CreateTodoVars = { id: string; input: CreateTodoInput };

export function useCreateTodo(options?: MutOpts<Todo, CreateTodoVars>) {
  const qc = useQueryClient();
  return useMutation<Todo, ApiError, CreateTodoVars>({
    mutationFn: ({ id, input }) => tripApi.createTodo(id, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.todos(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type UpdateTodoVars = { id: string; todoId: string; input: UpdateTodoInput };

export function useUpdateTodo(options?: MutOpts<Todo, UpdateTodoVars>) {
  const qc = useQueryClient();
  return useMutation<Todo, ApiError, UpdateTodoVars>({
    mutationFn: ({ id, todoId, input }) =>
      tripApi.updateTodo(id, todoId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.todos(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeleteTodoVars = { id: string; todoId: string };

export function useDeleteTodo(options?: MutOpts<null, DeleteTodoVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeleteTodoVars>({
    mutationFn: ({ id, todoId }) => tripApi.deleteTodo(id, todoId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.todos(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type ReorderTodosVars = { id: string; input: ReorderTodosInput };

export function useReorderTodos(options?: MutOpts<null, ReorderTodosVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, ReorderTodosVars>({
    mutationFn: ({ id, input }) => tripApi.reorderTodos(id, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: tripKeys.todos(vars.id) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

export function errorMessage(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) return err.message;
  return "Network error";
}
