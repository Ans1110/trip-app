import "server-only";
import { UpstreamUserSummary } from "./friend";

export type UpstreamTripStatus = "planning" | "ongoing" | "completed";

export type UpstreamRoomBrief = {
  id: string;
  room_code: string;
};

export type UpstreamTripResponse = {
  id: string;
  owner: UpstreamUserSummary;
  title: string;
  description: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
  status: UpstreamTripStatus;
  location?: string;
  latitude?: number;
  longitude?: number;
  member_count: number;
  my_role?: string;
  room?: UpstreamRoomBrief;
  created_at: string;
  updated_at: string;
};

export type UpstreamCreateTripPayload = {
  title: string;
  description?: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
  location?: string;
  latitude?: number;
  longitude?: number;
};

export type UpstreamUpdateTripPayload = {
  title?: string;
  description?: string;
  cover_image?: string;
  start_date?: string;
  end_date?: string;
  status?: UpstreamTripStatus;
  location?: string;
  latitude?: number | null;
  longitude?: number | null;
};

export type UpstreamListTripsQuery = {
  status?: UpstreamTripStatus;
  q?: string;
  role?: "owner" | "member" | "all";
  limit?: number;
  offset?: number;
};

export type UpstreamRoomMember = {
  user: UpstreamUserSummary;
  role: "admin" | "member";
  joined_at: string;
};

export type UpstreamRoomResponse = {
  id: string;
  trip_id: string;
  room_code: string;
  members: UpstreamRoomMember[];
  created_at: string;
};

export type UpstreamRoomPreview = {
  trip_id: string;
  room_id: string;
  title: string;
  description: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
  status: UpstreamTripStatus;
  owner: UpstreamUserSummary;
  member_count: number;
  already_joined: boolean;
};

export type UpstreamJoinRoomPayload = {
  code: string;
};

export type UpstreamJoinRoomResponse = {
  trip_id: string;
  room_id: string;
  already_member: boolean;
};

export type UpstreamItineraryResponse = {
  id: string;
  trip_id: string;
  day: number;
  title: string;
  description: string;
  start_time?: string;
  end_time?: string;
  location: string;
  latitude?: number;
  longitude?: number;
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type UpstreamCreateItineraryPayload = {
  day: number;
  title?: string;
  description?: string;
  start_time?: string;
  end_time?: string;
  location?: string;
  latitude?: number;
  longitude?: number;
  sort_order?: number;
};

export type UpstreamUpdateItineraryPayload = {
  day?: number;
  title?: string;
  description?: string;
  start_time?: string | null;
  end_time?: string | null;
  location?: string;
  latitude?: number | null;
  longitude?: number | null;
  sort_order?: number;
};

export type UpstreamReorderItineraryPayload = {
  item_ids: string[];
};

export type UpstreamTodoPriority = "low" | "normal" | "high";

export type UpstreamTodoResponse = {
  id: string;
  trip_id: string;
  title: string;
  is_completed: boolean;
  assignee_id?: string;
  due_date?: string;
  priority: UpstreamTodoPriority;
  tags: string[];
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type UpstreamCreateTodoPayload = {
  title: string;
  assignee_id?: string;
  due_date?: string;
  priority?: UpstreamTodoPriority;
  tags?: string[];
};

export type UpstreamUpdateTodoPayload = {
  title?: string;
  assignee_id?: string | null;
  is_completed?: boolean;
  due_date?: string | null;
  priority?: UpstreamTodoPriority;
  tags?: string[];
  sort_order?: number;
};

export type UpstreamReorderTodosPayload = {
  todo_ids: string[];
};
