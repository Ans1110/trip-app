import "server-only";

export type UpstreamRegisterRequest = {
  email: string;
  password: string;
  name: string;
};

export type UpstreamLoginRequest = {
  email: string;
  password: string;
  totp_code?: string;
};

export type UpstreamGoogleOAuthRequest = { id_token: string };
export type UpstreamGithubOAuthRequest = { code: string };
export type UpstreamFacebookOAuthRequest = { access_token: string };

export type UpstreamRefreshTokenRequest = { refresh_token: string };
export type UpstreamLogoutRequest = { refresh_token: string };

export type UpstreamVerifyEmailRequest = { token: string };
export type UpstreamResendVerificationRequest = { email: string };
export type UpstreamForgotPasswordRequest = { email: string };
export type UpstreamResetPasswordRequest = { token: string; password: string };

export type UpstreamChangePasswordRequest = {
  current_password: string;
  new_password: string;
};

export type UpstreamVerifyTOTPRequest = { totp_code: string };

export type UpstreamUserResponse = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_verified: boolean;
  created_at: string;
};

export type UpstreamSessionResponse = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type?: string;
  user: UpstreamUserResponse;
  requires_totp?: boolean;
  requires_verification?: boolean;
};

export type UpstreamTOTPSetupResponse = {
  secret: string;
  provisioning_url: string;
};

export type UpstreamUserSession = {
  id: string;
  device_name: string;
  device_type: "web" | "ios" | "android" | "";
  ip_address: string;
  user_agent: string;
  last_active_at: string;
  expires_at: string;
  created_at: string;
};

export type UpstreamJWK = {
  kty: string;
  use?: string;
  kid: string;
  alg?: string;
  n?: string;
  e?: string;
};

export type UpstreamJWKResponse = {
  keys: UpstreamJWK[];
};

// ---- friend ----

export type UpstreamUserSummary = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
};

export type UpstreamFriendResponse = {
  user: UpstreamUserSummary;
  created_at: string;
};

export type UpstreamRequestResponse = {
  id: string;
  status: string;
  message?: string;
  sender: UpstreamUserSummary;
  receiver: UpstreamUserSummary;
  created_at: string;
  updated_at: string;
};

export type UpstreamBlockResponse = {
  user: UpstreamUserSummary;
  created_at: string;
};

export type UpstreamSendFriendRequestPayload = {
  receiver_id: string;
  message?: string;
};

export type UpstreamBlockPayload = {
  target_id: string;
};

// ---- trip ----

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
};

export type UpstreamUpdateTripPayload = {
  title?: string;
  description?: string;
  cover_image?: string;
  start_date?: string;
  end_date?: string;
  status?: UpstreamTripStatus;
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

export type UpstreamRealtimeTicketResponse = {
  ticket: string;
  expires_in: number;
};

// ---- calendar ----

export type UpstreamEventVisibility = "private" | "room" | "friends" | "public";
export type UpstreamEventSource = "user" | "trip";
export type UpstreamEventType =
  | "general"
  | "trip"
  | "flight"
  | "hotel"
  | "meeting";

export type UpstreamEventResponse = {
  id: string;
  created_by: string;
  source_type: UpstreamEventSource;
  source_id?: string;
  event_type: UpstreamEventType;
  visibility: UpstreamEventVisibility;
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

export type UpstreamCreateEventPayload = {
  title: string;
  description?: string;
  location?: string;
  start_at: string;
  end_at: string;
  time_zone?: string;
  all_day?: boolean;
  color?: string;
  event_type?: UpstreamEventType;
  visibility: "private" | "friends";
};

export type UpstreamUpdateEventPayload = {
  title?: string;
  description?: string;
  location?: string;
  start_at?: string;
  end_at?: string;
  time_zone?: string;
  all_day?: boolean;
  color?: string;
  event_type?: UpstreamEventType;
  visibility?: "private" | "friends" | "public";
  version: number;
};

export type UpstreamListEventsQuery = {
  from?: string;
  to?: string;
};

// ---- media ----

export type UpstreamMediaPurpose = "chat" | "avatar" | "cover" | "todo";

export type UpstreamInitUploadPayload = {
  purpose: UpstreamMediaPurpose;
  mime: string;
  bytes: number;
};

export type UpstreamInitUploadResponse = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type UpstreamCompleteUploadPayload = {
  upload_id: string;
};

export type UpstreamAssetResponse = {
  id: string;
  purpose: UpstreamMediaPurpose;
  mime: string;
  bytes: number;
  width?: number;
  height?: number;
  created_at: string;
};

export type UpstreamPresignedGetResponse = {
  url: string;
  expires_at: string;
};

// ---- chat ----

export type UpstreamChatMessageType =
  | "text"
  | "image"
  | "video"
  | "file"
  | "system";

export type UpstreamChatRoomType = "group" | "dm";

export type UpstreamChatMessage = {
  id: string;
  room_id: string;
  sender_id: string;
  content: string;
  type: UpstreamChatMessageType;
  reply_to?: string;
  media_id?: string;
  media_mime?: string;
  media_bytes?: number;
  client_msg_id?: string;
  is_edited: boolean;
  is_deleted: boolean;
  is_pinned: boolean;
  created_at: string;
};

export type UpstreamChatRoom = {
  id: string;
  trip_id?: string;
  name: string;
  type: UpstreamChatRoomType;
  last_message?: UpstreamChatMessage;
  unread_count: number;
  created_at: string;
};

export type UpstreamChatReadReceipt = {
  room_id: string;
  user_id: string;
  last_read_id: string;
  updated_at: string;
};

export type UpstreamEnsureDMPayload = {
  peer_id: string;
};

export type UpstreamMarkReadPayload = {
  last_read_id: string;
};

export type UpstreamChatTicketResponse = {
  ticket: string;
  expires_in: number;
};

export type UpstreamListChatMessagesQuery = {
  before?: string;
  limit?: number;
};
