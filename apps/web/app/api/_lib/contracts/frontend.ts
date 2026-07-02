import "server-only";
import {
  UpstreamBlockResponse,
  UpstreamFriendResponse,
  UpstreamItineraryResponse,
  UpstreamJoinRoomResponse,
  UpstreamJWKResponse,
  UpstreamRealtimeTicketResponse,
  UpstreamRequestResponse,
  UpstreamRoomBrief,
  UpstreamRoomMember,
  UpstreamRoomPreview,
  UpstreamRoomResponse,
  UpstreamSessionResponse,
  UpstreamTodoResponse,
  UpstreamTOTPSetupResponse,
  UpstreamTripResponse,
  UpstreamTripStatus,
  UpstreamUserResponse,
  UpstreamUserSession,
  UpstreamUserSummary,
} from "./upstream";

export type User = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_verified: boolean;
  created_at: string;
};

export type SessionView = {
  user: User;
  expires_in: number;
  token_type?: string;
  requires_totp?: boolean;
  requires_verification?: boolean;
};

export type TOTPSetupView = {
  secret: string;
  provisioning_url: string;
};

export type UserSessionView = {
  id: string;
  device_name: string;
  device_type: "web" | "ios" | "android" | "";
  ip_address: string;
  user_agent: string;
  last_active_at: string;
  expires_at: string;
  created_at: string;
};

export type JWKView = UpstreamJWKResponse;

export const toUser = (u: UpstreamUserResponse): User => ({
  id: u.id,
  email: u.email,
  name: u.name,
  avatar_url: u.avatar_url,
  is_verified: u.is_verified,
  created_at: u.created_at,
});

export const toSessionView = (s: UpstreamSessionResponse): SessionView => ({
  user: toUser(s.user),
  expires_in: s.expires_in,
  token_type: s.token_type,
  requires_totp: s.requires_totp,
  requires_verification: s.requires_verification,
});

export const toTOTPSetupView = (
  t: UpstreamTOTPSetupResponse,
): TOTPSetupView => ({
  secret: t.secret,
  provisioning_url: t.provisioning_url,
});

export const toUserSessionView = (s: UpstreamUserSession): UserSessionView => ({
  id: s.id,
  device_name: s.device_name,
  device_type: s.device_type,
  ip_address: s.ip_address,
  user_agent: s.user_agent,
  last_active_at: s.last_active_at,
  expires_at: s.expires_at,
  created_at: s.created_at,
});

// ---- friend ----

export type UserSummaryView = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
};

export type FriendView = {
  user: UserSummaryView;
  created_at: string;
};

export type FriendRequestView = {
  id: string;
  status: string;
  message?: string;
  sender: UserSummaryView;
  receiver: UserSummaryView;
  created_at: string;
  updated_at: string;
};

export type FriendBlockView = {
  user: UserSummaryView;
  created_at: string;
};

export const toUserSummaryView = (u: UpstreamUserSummary): UserSummaryView => ({
  id: u.id,
  email: u.email,
  name: u.name,
  avatar_url: u.avatar_url,
});

export const toFriendView = (f: UpstreamFriendResponse): FriendView => ({
  user: toUserSummaryView(f.user),
  created_at: f.created_at,
});

export const toFriendRequestView = (
  r: UpstreamRequestResponse,
): FriendRequestView => ({
  id: r.id,
  status: r.status,
  message: r.message,
  sender: toUserSummaryView(r.sender),
  receiver: toUserSummaryView(r.receiver),
  created_at: r.created_at,
  updated_at: r.updated_at,
});

export const toFriendBlockView = (b: UpstreamBlockResponse): FriendBlockView => ({
  user: toUserSummaryView(b.user),
  created_at: b.created_at,
});

// ---- trip ----

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

export const toRoomPreviewView = (
  p: UpstreamRoomPreview,
): RoomPreviewView => ({
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

// ---- realtime ----

export type RealtimeTicketView = {
  ticket: string;
  expires_in: number;
};

export const toRealtimeTicketView = (
  t: UpstreamRealtimeTicketResponse,
): RealtimeTicketView => ({
  ticket: t.ticket,
  expires_in: t.expires_in,
});
