import { ApiError } from "./friend-api";

export type EventVisibility = "private" | "room" | "friends" | "public";
export type EventSource = "user" | "trip";
export type EventType = "general" | "trip" | "flight" | "hotel" | "meeting";

// Client-side creation is limited to visibilities the server actually accepts.
// The room/public/trip sources come from the trip module, not this API.
export type CreatableVisibility = "private" | "friends";

export type CalendarEvent = {
  id: string;
  created_by: string;
  source_type: EventSource;
  source_id?: string;
  event_type: EventType;
  visibility: EventVisibility;
  title: string;
  description?: string;
  location?: string;
  start_at: string;
  end_at: string;
  time_zone?: string;
  all_day: boolean;
  color?: string;
  version: number;
  created_at: string;
  updated_at: string;
};

export type CreateEventInput = {
  title: string;
  description?: string;
  location?: string;
  start_at: string;
  end_at: string;
  time_zone?: string;
  all_day?: boolean;
  color?: string;
  event_type?: EventType;
  visibility: CreatableVisibility;
};

export type UpdateEventInput = {
  title?: string;
  description?: string;
  location?: string;
  start_at?: string;
  end_at?: string;
  time_zone?: string;
  all_day?: boolean;
  color?: string;
  event_type?: EventType;
  visibility?: "private" | "friends" | "public";
  version: number;
};

export type ListEventsQuery = {
  from?: string;
  to?: string;
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

const buildRangeQuery = (q: ListEventsQuery): string => {
  const qs = new URLSearchParams();
  if (q.from) qs.set("from", q.from);
  if (q.to) qs.set("to", q.to);
  return qs.toString();
};

export const calendarApi = {
  listMine: (query: ListEventsQuery = {}, signal?: AbortSignal) => {
    const qs = buildRangeQuery(query);
    const path = qs ? `/api/calendar/events?${qs}` : "/api/calendar/events";
    return request<CalendarEvent[]>(path, { signal });
  },
  create: (input: CreateEventInput, signal?: AbortSignal) =>
    request<CalendarEvent>("/api/calendar/events", {
      method: "POST",
      json: input,
      signal,
    }),
  get: (id: string, signal?: AbortSignal) =>
    request<CalendarEvent>(`/api/calendar/events/${encodeURIComponent(id)}`, {
      signal,
    }),
  update: (id: string, input: UpdateEventInput, signal?: AbortSignal) =>
    request<CalendarEvent>(`/api/calendar/events/${encodeURIComponent(id)}`, {
      method: "PATCH",
      json: input,
      signal,
    }),
  remove: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/calendar/events/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  listTrip: (
    tripId: string,
    query: ListEventsQuery = {},
    signal?: AbortSignal,
  ) => {
    const qs = buildRangeQuery(query);
    const base = `/api/calendar/trips/${encodeURIComponent(tripId)}/events`;
    return request<CalendarEvent[]>(qs ? `${base}?${qs}` : base, { signal });
  },
  listFriend: (
    friendId: string,
    query: ListEventsQuery = {},
    signal?: AbortSignal,
  ) => {
    const qs = buildRangeQuery(query);
    const base = `/api/calendar/friends/${encodeURIComponent(friendId)}/events`;
    return request<CalendarEvent[]>(qs ? `${base}?${qs}` : base, { signal });
  },
};
