import "server-only";
import {
  UpstreamItineraryResponse,
  UpstreamJoinRoomResponse,
  UpstreamRoomBrief,
  UpstreamRoomMember,
  UpstreamRoomPreview,
  UpstreamRoomResponse,
  UpstreamTodoResponse,
  UpstreamTripResponse,
  UpstreamTripStatus,
} from "../upstream";
import { toUserSummaryView, UserSummaryView } from "./friend";

export type TripStatusView = UpstreamTripStatus;

export type RoomBriefView = {
  id: string;
  room_code: string;
};

export type TripView = {
  id: string;
  owner: UserSummaryView;
  title: string;
  description: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
  status: TripStatusView;
  member_count: number;
  my_role?: string;
  room?: RoomBriefView;
  created_at: string;
  updated_at: string;
};

export type RoomMemberView = {
  user: UserSummaryView;
  role: "admin" | "member";
  joined_at: string;
};

export type RoomView = {
  id: string;
  trip_id: string;
  room_code: string;
  members: RoomMemberView[];
  created_at: string;
};

export type RoomPreviewView = {
  trip_id: string;
  room_id: string;
  title: string;
  description: string;
  cover_image?: string;
  start_date: string;
  end_date: string;
  status: TripStatusView;
  owner: UserSummaryView;
  member_count: number;
  already_joined: boolean;
};

export type JoinRoomView = {
  trip_id: string;
  room_id: string;
  already_member: boolean;
};

export type ItineraryView = {
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

export type TodoPriorityView = "low" | "normal" | "high";

export type TodoView = {
  id: string;
  trip_id: string;
  title: string;
  is_completed: boolean;
  assignee_id?: string;
  due_date?: string;
  priority: TodoPriorityView;
  tags: string[];
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export const toRoomBriefView = (b: UpstreamRoomBrief): RoomBriefView => ({
  id: b.id,
  room_code: b.room_code,
});

export const toTripView = (t: UpstreamTripResponse): TripView => ({
  id: t.id,
  owner: toUserSummaryView(t.owner),
  title: t.title,
  description: t.description,
  cover_image: t.cover_image,
  start_date: t.start_date,
  end_date: t.end_date,
  status: t.status,
  member_count: t.member_count,
  my_role: t.my_role,
  room: t.room ? toRoomBriefView(t.room) : undefined,
  created_at: t.created_at,
  updated_at: t.updated_at,
});

export const toRoomMemberView = (m: UpstreamRoomMember): RoomMemberView => ({
  user: toUserSummaryView(m.user),
  role: m.role,
  joined_at: m.joined_at,
});

export const toRoomView = (r: UpstreamRoomResponse): RoomView => ({
  id: r.id,
  trip_id: r.trip_id,
  room_code: r.room_code,
  members: r.members.map(toRoomMemberView),
  created_at: r.created_at,
});

export const toRoomPreviewView = (p: UpstreamRoomPreview): RoomPreviewView => ({
  trip_id: p.trip_id,
  room_id: p.room_id,
  title: p.title,
  description: p.description,
  cover_image: p.cover_image,
  start_date: p.start_date,
  end_date: p.end_date,
  status: p.status,
  owner: toUserSummaryView(p.owner),
  member_count: p.member_count,
  already_joined: p.already_joined,
});

export const toJoinRoomView = (j: UpstreamJoinRoomResponse): JoinRoomView => ({
  trip_id: j.trip_id,
  room_id: j.room_id,
  already_member: j.already_member,
});

export const toItineraryView = (
  i: UpstreamItineraryResponse,
): ItineraryView => ({
  id: i.id,
  trip_id: i.trip_id,
  day: i.day,
  title: i.title,
  description: i.description,
  start_time: i.start_time,
  end_time: i.end_time,
  location: i.location,
  latitude: i.latitude,
  longitude: i.longitude,
  sort_order: i.sort_order,
  created_by: i.created_by,
  created_at: i.created_at,
  updated_at: i.updated_at,
});

export const toTodoView = (t: UpstreamTodoResponse): TodoView => ({
  id: t.id,
  trip_id: t.trip_id,
  title: t.title,
  is_completed: t.is_completed,
  assignee_id: t.assignee_id,
  due_date: t.due_date,
  priority: t.priority ?? "normal",
  tags: t.tags ?? [],
  sort_order: t.sort_order ?? 0,
  created_by: t.created_by,
  created_at: t.created_at,
  updated_at: t.updated_at,
});
