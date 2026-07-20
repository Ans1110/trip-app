// NOTE: server broadcast payloads (ITINERARY_UPDATE, TODO_UPDATE,
// CALENDAR_*) marshal Go structs without json tags, so their `data` field
// currently ships PascalCase field names and does NOT match the REST view
// shapes. We type `data` as `unknown` for those events and let the hook layer
// react via invalidateQueries, not setQueryData.

export type ClientOpType =
  | "CURSOR_MOVE"
  | "ITINERARY_UPDATE"
  | "TODO_UPDATE"
  | "CALENDAR_UPDATE"
  | "VOTE_CAST";

export type ErrorCode =
  | "BAD_OP"
  | "STALE_VERSION"
  | "NOT_FOUND"
  | "FORBIDDEN"
  | "INTERNAL"
  | "UNSUPPORTED_OP";

// ---- Client -> Server ----

export type CursorMoveData = {
  x: number;
  y: number;
  selection?: string;
};

export type ItineraryUpdateData = {
  item_id: string;
  title?: string;
  description?: string;
  location?: string;
  day?: number;
  latitude?: number;
  longitude?: number;
};

export type TodoUpdateData = {
  todo_id: string;
  title?: string;
  is_completed?: boolean;
  priority?: "low" | "normal" | "high";
};

type BaseClientMsg = {
  msg_id?: string;
  trip_id: string;
};

export type CursorMoveMsg = BaseClientMsg & {
  type: "CURSOR_MOVE";
  data: CursorMoveData;
};

export type ItineraryUpdateMsg = BaseClientMsg & {
  type: "ITINERARY_UPDATE";
  version: number;
  data: ItineraryUpdateData;
};

export type TodoUpdateMsg = BaseClientMsg & {
  type: "TODO_UPDATE";
  version: number;
  data: TodoUpdateData;
};

export type ClientMsg = CursorMoveMsg | ItineraryUpdateMsg | TodoUpdateMsg;

// ---- Server -> Client ----

export type WelcomeEvent = {
  type: "WELCOME";
  seq?: number;
  trip_id: string;
  from: string;
};

export type AckEvent = {
  type: "ACK";
  ref?: string;
  version?: number;
};

export type ErrorEvent = {
  type: "ERROR";
  ref?: string;
  code: ErrorCode;
  message: string;
};

export type CursorMoveEvent = {
  type: "CURSOR_MOVE";
  seq: number;
  from: string;
  trip_id: string;
  data: CursorMoveData;
};

export type ItineraryUpdatedEvent = {
  type: "ITINERARY_UPDATE";
  seq: number;
  from: string;
  trip_id: string;
  version: number;
  data: unknown;
};

export type TodoUpdatedEvent = {
  type: "TODO_UPDATE";
  seq: number;
  from: string;
  trip_id: string;
  version: number;
  data: unknown;
};

export type CalendarCreatedEvent = {
  type: "CALENDAR_CREATED";
  seq?: number;
  from?: string;
  trip_id?: string;
  version?: number;
  data: unknown;
};

export type CalendarUpdatedEvent = {
  type: "CALENDAR_UPDATED";
  seq?: number;
  from?: string;
  trip_id?: string;
  version?: number;
  data: unknown;
};

export type CalendarDeletedEvent = {
  type: "CALENDAR_DELETED";
  seq?: number;
  from?: string;
  trip_id?: string;
  version?: number;
};

// Vote events piggyback on the trip realtime channel. `poll` carries the full
// PollDTO JSON envelope from the Go vote service; the hook layer refetches
// rather than trusting the frame directly (visibility is per-viewer).
export type VoteBroadcastType =
  | "VOTE_POLL_CREATED"
  | "VOTE_POLL_UPDATED"
  | "VOTE_POLL_CLOSED"
  | "VOTE_OPTION_ADDED"
  | "VOTE_CAST";

export type VoteBroadcastEvent = {
  type: VoteBroadcastType;
  trip_id: string;
  poll: { id: string } & Record<string, unknown>;
};

export type VotePollDeletedEvent = {
  type: "VOTE_POLL_DELETED";
  trip_id: string;
  poll_id: string;
};

export type ServerMsg =
  | WelcomeEvent
  | AckEvent
  | ErrorEvent
  | CursorMoveEvent
  | ItineraryUpdatedEvent
  | TodoUpdatedEvent
  | CalendarCreatedEvent
  | CalendarUpdatedEvent
  | CalendarDeletedEvent
  | VoteBroadcastEvent
  | VotePollDeletedEvent;

export function isServerMsg(v: unknown): v is ServerMsg {
  return (
    typeof v === "object" &&
    v !== null &&
    typeof (v as { type?: unknown }).type === "string"
  );
}
