import { ApiError } from "./friend-api";
import type { UserSummary } from "./friend-api";

export type { UserSummary } from "./friend-api";

export type TripStatus = "planning" | "ongoing" | "completed";

// Statuses a client may set directly. "completed" is derived server-side
// from the trip's end date and cannot be assigned manually.
export type TripUpdatableStatus = "planning" | "ongoing";

export type RoomBrief = {
  id: string;
  room_code: string;
};

export type Trip = {
  id: string;
  owner: UserSummary;
  title: string;
  description: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
  status: TripStatus;
  member_count: number;
  my_role?: string;
  room?: RoomBrief;
  created_at: string;
  updated_at: string;
};

export type RoomMember = {
  user: UserSummary;
  role: "admin" | "member";
  joined_at: string;
};

export type Room = {
  id: string;
  trip_id: string;
  room_code: string;
  members: RoomMember[];
  created_at: string;
};

export type RoomPreview = {
  trip_id: string;
  room_id: string;
  title: string;
  description: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
  status: TripStatus;
  owner: UserSummary;
  member_count: number;
  already_joined: boolean;
};

export type JoinRoomResult = {
  trip_id: string;
  room_id: string;
  already_member: boolean;
};

export type Itinerary = {
  id: string;
  trip_id: string;
  day: number;
  title: string;
  description: string;
  start_time?: string;
  end_time?: string;
  location: string;
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type Todo = {
  id: string;
  trip_id: string;
  title: string;
  is_completed: boolean;
  assignee_id?: string;
  due_date?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type CreateTripInput = {
  title: string;
  description?: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
};

export type UpdateTripInput = {
  title?: string;
  description?: string;
  cover_image?: string;
  start_date?: string;
  end_date?: string;
  status?: TripUpdatableStatus;
};

export type ListTripsQuery = {
  status?: TripStatus;
  q?: string;
  role?: "owner" | "member" | "all";
  limit?: number;
  offset?: number;
};

export type JoinRoomInput = { code: string };

export type CreateItineraryInput = {
  day: number;
  title?: string;
  description?: string;
  start_time?: string;
  end_time?: string;
  location?: string;
  sort_order?: number;
};

export type UpdateItineraryInput = {
  day?: number;
  title?: string;
  description?: string;
  start_time?: string | null;
  end_time?: string | null;
  location?: string;
  sort_order?: number;
};

export type CreateTodoInput = {
  title: string;
  assignee_id?: string;
  due_date?: string;
};

export type UpdateTodoInput = {
  title?: string;
  assignee_id?: string | null;
  is_completed?: boolean;
  due_date?: string | null;
};

type Envelope<T> = { code: number; message: string; data: T | null };

async function request<T>(
  path: string,
  init: RequestInit & { json?: unknown } = {},
): Promise<T> {
  const { json, headers, ...rest } = init;
  const res = await fetch(path, {
    credentials: "include",
    ...rest,
    headers: {
      ...(json !== undefined ? { "content-type": "application/json" } : {}),
      ...headers,
    },
    body: json !== undefined ? JSON.stringify(json) : rest.body,
  });

  const body = (await res.json().catch(() => null)) as Envelope<T> | null;
  if (!res.ok) {
    throw new ApiError(
      body?.message ?? `Request failed (${res.status})`,
      res.status,
      body?.code,
    );
  }
  return (body?.data as T) ?? (null as T);
}

const buildListTripsQuery = (q: ListTripsQuery): string => {
  const qs = new URLSearchParams();
  if (q.status) qs.set("status", q.status);
  if (q.q) qs.set("q", q.q);
  if (q.role) qs.set("role", q.role);
  if (q.limit !== undefined) qs.set("limit", String(q.limit));
  if (q.offset !== undefined) qs.set("offset", String(q.offset));
  return qs.toString();
};

export const tripApi = {
  // Trip
  list: (query: ListTripsQuery = {}, signal?: AbortSignal) => {
    const qs = buildListTripsQuery(query);
    const path = qs ? `/api/trips?${qs}` : "/api/trips";
    return request<Trip[]>(path, { signal });
  },
  create: (input: CreateTripInput, signal?: AbortSignal) =>
    request<Trip>("/api/trips", { method: "POST", json: input, signal }),
  get: (id: string, signal?: AbortSignal) =>
    request<Trip>(`/api/trips/${encodeURIComponent(id)}`, { signal }),
  update: (id: string, input: UpdateTripInput, signal?: AbortSignal) =>
    request<Trip>(`/api/trips/${encodeURIComponent(id)}`, {
      method: "PATCH",
      json: input,
      signal,
    }),
  remove: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/trips/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  // Room
  getRoom: (id: string, signal?: AbortSignal) =>
    request<Room>(`/api/trips/${encodeURIComponent(id)}/room`, { signal }),
  leaveRoom: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/trips/${encodeURIComponent(id)}/room/leave`, {
      method: "POST",
      signal,
    }),
  removeMember: (id: string, userId: string, signal?: AbortSignal) =>
    request<null>(
      `/api/trips/${encodeURIComponent(id)}/room/members/${encodeURIComponent(userId)}`,
      { method: "DELETE", signal },
    ),
  regenerateCode: (id: string, signal?: AbortSignal) =>
    request<Room>(`/api/trips/${encodeURIComponent(id)}/room/code`, {
      method: "POST",
      signal,
    }),
  previewByCode: (code: string, signal?: AbortSignal) =>
    request<RoomPreview>(`/api/rooms/by-code/${encodeURIComponent(code)}`, {
      signal,
    }),
  join: (input: JoinRoomInput, signal?: AbortSignal) =>
    request<JoinRoomResult>("/api/rooms/join", {
      method: "POST",
      json: input,
      signal,
    }),

  // Itinerary
  listItinerary: (id: string, signal?: AbortSignal) =>
    request<Itinerary[]>(`/api/trips/${encodeURIComponent(id)}/itinerary`, {
      signal,
    }),
  createItinerary: (
    id: string,
    input: CreateItineraryInput,
    signal?: AbortSignal,
  ) =>
    request<Itinerary>(`/api/trips/${encodeURIComponent(id)}/itinerary`, {
      method: "POST",
      json: input,
      signal,
    }),
  updateItinerary: (
    id: string,
    itemId: string,
    input: UpdateItineraryInput,
    signal?: AbortSignal,
  ) =>
    request<Itinerary>(
      `/api/trips/${encodeURIComponent(id)}/itinerary/${encodeURIComponent(itemId)}`,
      { method: "PATCH", json: input, signal },
    ),
  deleteItinerary: (id: string, itemId: string, signal?: AbortSignal) =>
    request<null>(
      `/api/trips/${encodeURIComponent(id)}/itinerary/${encodeURIComponent(itemId)}`,
      { method: "DELETE", signal },
    ),

  // Todos
  listTodos: (id: string, signal?: AbortSignal) =>
    request<Todo[]>(`/api/trips/${encodeURIComponent(id)}/todos`, { signal }),
  createTodo: (id: string, input: CreateTodoInput, signal?: AbortSignal) =>
    request<Todo>(`/api/trips/${encodeURIComponent(id)}/todos`, {
      method: "POST",
      json: input,
      signal,
    }),
  updateTodo: (
    id: string,
    todoId: string,
    input: UpdateTodoInput,
    signal?: AbortSignal,
  ) =>
    request<Todo>(
      `/api/trips/${encodeURIComponent(id)}/todos/${encodeURIComponent(todoId)}`,
      { method: "PATCH", json: input, signal },
    ),
  deleteTodo: (id: string, todoId: string, signal?: AbortSignal) =>
    request<null>(
      `/api/trips/${encodeURIComponent(id)}/todos/${encodeURIComponent(todoId)}`,
      { method: "DELETE", signal },
    ),
};
