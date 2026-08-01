import "server-only";
import {
  UpstreamAlbumCreateShareResponse,
  UpstreamAlbumDownloadOriginal,
  UpstreamAlbumInitUploadResponse,
  UpstreamAlbumInitUploadSlot,
  UpstreamAlbumPhoto,
  UpstreamAlbumPhotoWithURLs,
  UpstreamAlbumPublicResponse,
  UpstreamAlbumShareToken,
  UpstreamAssetResponse,
  UpstreamBlockResponse,
  UpstreamBudget,
  UpstreamCategoryStat,
  UpstreamChatMessage,
  UpstreamChatMessageType,
  UpstreamChatReadReceipt,
  UpstreamChatRoom,
  UpstreamChatRoomType,
  UpstreamChatTicketResponse,
  UpstreamEventResponse,
  UpstreamEventSource,
  UpstreamEventType,
  UpstreamEventVisibility,
  UpstreamExpense,
  UpstreamExpenseShare,
  UpstreamFriendResponse,
  UpstreamFxRate,
  UpstreamInitUploadResponse,
  UpstreamItineraryResponse,
  UpstreamJoinRoomResponse,
  UpstreamJWKResponse,
  UpstreamMediaPurpose,
  UpstreamPersonalStats,
  UpstreamPoll,
  UpstreamPollOption,
  UpstreamPollResultVisibility,
  UpstreamPollStatus,
  UpstreamPollType,
  UpstreamPresignedGetResponse,
  UpstreamRealtimeTicketResponse,
  UpstreamRequestResponse,
  UpstreamRoomBrief,
  UpstreamRoomMember,
  UpstreamRoomPreview,
  UpstreamRoomResponse,
  UpstreamSessionResponse,
  UpstreamSettlement,
  UpstreamSettlementStatus,
  UpstreamSplitStrategy,
  UpstreamTodoResponse,
  UpstreamTOTPSetupResponse,
  UpstreamTripBalance,
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

export const toFriendBlockView = (
  b: UpstreamBlockResponse,
): FriendBlockView => ({
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

// ---- calendar ----

export type EventVisibilityView = UpstreamEventVisibility;
export type EventSourceView = UpstreamEventSource;
export type EventTypeView = UpstreamEventType;

export type CalendarEventView = {
  id: string;
  created_by: string;
  source_type: EventSourceView;
  source_id?: string;
  event_type: EventTypeView;
  visibility: EventVisibilityView;
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

export const toCalendarEventView = (
  e: UpstreamEventResponse,
): CalendarEventView => ({
  id: e.id,
  created_by: e.created_by,
  source_type: e.source_type,
  source_id: e.source_id,
  event_type: e.event_type,
  visibility: e.visibility,
  title: e.title,
  description: e.description,
  location: e.location,
  start_at: e.start_at,
  end_at: e.end_at,
  time_zone: e.time_zone,
  all_day: e.all_day,
  color: e.color,
  version: e.version,
  created_at: e.created_at,
  updated_at: e.updated_at,
});

// ---- media ----

export type MediaPurposeView = UpstreamMediaPurpose;

export type InitUploadView = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type AssetView = {
  id: string;
  purpose: MediaPurposeView;
  mime: string;
  bytes: number;
  width?: number;
  height?: number;
  created_at: string;
};

export type PresignedGetView = {
  url: string;
  expires_at: string;
};

export const toInitUploadView = (
  r: UpstreamInitUploadResponse,
): InitUploadView => ({
  upload_id: r.upload_id,
  upload_url: r.upload_url,
  method: r.method,
  headers: r.headers,
  expires_at: r.expires_at,
});

export const toAssetView = (a: UpstreamAssetResponse): AssetView => ({
  id: a.id,
  purpose: a.purpose,
  mime: a.mime,
  bytes: a.bytes,
  width: a.width,
  height: a.height,
  created_at: a.created_at,
});

export const toPresignedGetView = (
  r: UpstreamPresignedGetResponse,
): PresignedGetView => ({
  url: r.url,
  expires_at: r.expires_at,
});

// ---- chat ----

export type ChatMessageTypeView = UpstreamChatMessageType;
export type ChatRoomTypeView = UpstreamChatRoomType;

export type ChatMessageView = {
  id: string;
  room_id: string;
  sender_id: string;
  content: string;
  type: ChatMessageTypeView;
  reply_to?: string;
  media_id?: string;
  media_mime?: string;
  media_bytes?: number;
  media_filename?: string;
  client_msg_id?: string;
  is_edited: boolean;
  is_deleted: boolean;
  is_pinned: boolean;
  created_at: string;
};

export type ChatPeerView = {
  id: string;
  name: string;
  email?: string;
  avatar_url?: string;
};

export type ChatRoomView = {
  id: string;
  trip_id?: string;
  name: string;
  type: ChatRoomTypeView;
  peer?: ChatPeerView;
  last_message?: ChatMessageView;
  unread_count: number;
  created_at: string;
};

export type ChatReadReceiptView = {
  room_id: string;
  user_id: string;
  last_read_id: string;
  updated_at: string;
};

export type ChatTicketView = {
  ticket: string;
  expires_in: number;
};

export const toChatMessageView = (m: UpstreamChatMessage): ChatMessageView => ({
  id: m.id,
  room_id: m.room_id,
  sender_id: m.sender_id,
  content: m.content,
  type: m.type,
  reply_to: m.reply_to,
  media_id: m.media_id,
  media_mime: m.media_mime,
  media_bytes: m.media_bytes,
  media_filename: m.media_filename,
  client_msg_id: m.client_msg_id,
  is_edited: m.is_edited,
  is_deleted: m.is_deleted,
  is_pinned: m.is_pinned,
  created_at: m.created_at,
});

export const toChatRoomView = (r: UpstreamChatRoom): ChatRoomView => ({
  id: r.id,
  trip_id: r.trip_id,
  name: r.name,
  type: r.type,
  peer: r.peer
    ? {
        id: r.peer.id,
        name: r.peer.name,
        email: r.peer.email,
        avatar_url: r.peer.avatar_url,
      }
    : undefined,
  last_message: r.last_message ? toChatMessageView(r.last_message) : undefined,
  unread_count: r.unread_count ?? 0,
  created_at: r.created_at,
});

export const toChatReadReceiptView = (
  r: UpstreamChatReadReceipt,
): ChatReadReceiptView => ({
  room_id: r.room_id,
  user_id: r.user_id,
  last_read_id: r.last_read_id,
  updated_at: r.updated_at,
});

export const toChatTicketView = (
  t: UpstreamChatTicketResponse,
): ChatTicketView => ({
  ticket: t.ticket,
  expires_in: t.expires_in,
});

// ---- vote ----

export type PollTypeView = UpstreamPollType;
export type PollStatusView = UpstreamPollStatus;
export type PollResultVisibilityView = UpstreamPollResultVisibility;

export type PollOptionView = {
  id: string;
  text: string;
  meta?: Record<string, unknown>;
  added_by: string;
  sort_order: number;
  created_at: string;
  count: number;
  voters?: string[];
};

export type PollView = {
  id: string;
  trip_id: string;
  created_by: string;
  type: PollTypeView;
  title: string;
  description?: string;
  is_anonymous: boolean;
  max_choices: number;
  allow_option_add: boolean;
  result_visibility: PollResultVisibilityView;
  deadline_at?: string;
  status: PollStatusView;
  closed_at?: string;
  created_at: string;
  updated_at: string;
  options: PollOptionView[];
  my_choices: string[];
  results_visible: boolean;
  total_voters: number;
};

export const toPollOptionView = (o: UpstreamPollOption): PollOptionView => ({
  id: o.id,
  text: o.text,
  meta: o.meta,
  added_by: o.added_by,
  sort_order: o.sort_order,
  created_at: o.created_at,
  count: o.count ?? 0,
  voters: o.voters,
});

export const toPollView = (p: UpstreamPoll): PollView => ({
  id: p.id,
  trip_id: p.trip_id,
  created_by: p.created_by,
  type: p.type,
  title: p.title,
  description: p.description,
  is_anonymous: p.is_anonymous,
  max_choices: p.max_choices,
  allow_option_add: p.allow_option_add,
  result_visibility: p.result_visibility,
  deadline_at: p.deadline_at,
  status: p.status,
  closed_at: p.closed_at,
  created_at: p.created_at,
  updated_at: p.updated_at,
  options: (p.options ?? []).map(toPollOptionView),
  my_choices: p.my_choices ?? [],
  results_visible: p.results_visible,
  total_voters: p.total_voters ?? 0,
});

// ---- finance ----
export type SplitStrategyView = UpstreamSplitStrategy;
export type SettlementStatusView = UpstreamSettlementStatus;

export type ExpenseShareView = {
  user_id: string;
  amount: string;
  amount_base: string;
  share_pct?: string;
  paid_at?: string;
};

export type ExpenseView = {
  id: string;
  trip_id: string;
  paid_by: string;
  amount: string;
  currency: string;
  amount_base: string;
  base_currency: string;
  rate_to_base?: string;
  description: string;
  category: string;
  split_strategy: SplitStrategyView;
  occurred_at: string;
  receipt_asset_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  shares: ExpenseShareView[];
};

export type BudgetView = {
  id: string;
  trip_id: string;
  category: string;
  amount: string;
  currency: string;
  spent_base: string;
  remaining_base: string;
  over_budget: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
};

export type SettlementView = {
  id: string;
  trip_id: string;
  payer_id: string;
  payee_id: string;
  amount: string;
  currency: string;
  status: SettlementStatusView;
  note: string;
  created_by: string;
  created_at: string;
  confirmed_at?: string;
  cancelled_at?: string;
};

export type FxRateView = {
  base: string;
  quote: string;
  rate: string;
  as_of: string;
  source: string;
};

export type CategoryStatView = {
  category: string;
  amount_base: string;
  count: number;
};

export type PersonalStatsView = {
  trip_id: string;
  user_id: string;
  base_currency: string;
  total_paid: string;
  total_owed: string;
  net_balance: string;
  by_category: CategoryStatView[];
};

export type TripBalanceView = {
  user_id: string;
  paid: string;
  owed: string;
  net: string;
};

export const toExpenseShareView = (
  s: UpstreamExpenseShare,
): ExpenseShareView => ({
  user_id: s.user_id,
  amount: s.amount,
  amount_base: s.amount_base,
  share_pct: s.share_pct,
  paid_at: s.paid_at,
});

export const toExpenseView = (e: UpstreamExpense): ExpenseView => ({
  id: e.id,
  trip_id: e.trip_id,
  paid_by: e.paid_by,
  amount: e.amount,
  currency: e.currency,
  amount_base: e.amount_base,
  base_currency: e.base_currency,
  rate_to_base: e.rate_to_base,
  description: e.description,
  category: e.category,
  split_strategy: e.split_strategy,
  occurred_at: e.occurred_at,
  receipt_asset_id: e.receipt_asset_id,
  created_by: e.created_by,
  created_at: e.created_at,
  updated_at: e.updated_at,
  shares: (e.shares ?? []).map(toExpenseShareView),
});

export const toBudgetView = (b: UpstreamBudget): BudgetView => ({
  id: b.id,
  trip_id: b.trip_id,
  category: b.category,
  amount: b.amount,
  currency: b.currency,
  spent_base: b.spent_base,
  remaining_base: b.remaining_base,
  over_budget: b.over_budget,
  created_by: b.created_by,
  created_at: b.created_at,
  updated_at: b.updated_at,
});

export const toSettlementView = (s: UpstreamSettlement): SettlementView => ({
  id: s.id,
  trip_id: s.trip_id,
  payer_id: s.payer_id,
  payee_id: s.payee_id,
  amount: s.amount,
  currency: s.currency,
  status: s.status,
  note: s.note,
  created_by: s.created_by,
  created_at: s.created_at,
  confirmed_at: s.confirmed_at,
  cancelled_at: s.cancelled_at,
});

export const toFxRateView = (r: UpstreamFxRate): FxRateView => ({
  base: r.base,
  quote: r.quote,
  rate: r.rate,
  as_of: r.as_of,
  source: r.source,
});

export const toCategoryStatView = (
  c: UpstreamCategoryStat,
): CategoryStatView => ({
  category: c.category,
  amount_base: c.amount_base,
  count: c.count ?? 0,
});

export const toPersonalStatsView = (
  s: UpstreamPersonalStats,
): PersonalStatsView => ({
  trip_id: s.trip_id,
  user_id: s.user_id,
  base_currency: s.base_currency,
  total_paid: s.total_paid,
  total_owed: s.total_owed,
  net_balance: s.net_balance,
  by_category: (s.by_category ?? []).map(toCategoryStatView),
});

export const toTripBalanceView = (b: UpstreamTripBalance): TripBalanceView => ({
  user_id: b.user_id,
  paid: b.paid,
  owed: b.owed,
  net: b.net,
});

// ---- album ----

export type AlbumInitUploadSlotView = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type AlbumInitUploadView = {
  slots: AlbumInitUploadSlotView[];
};

export type AlbumPhotoView = {
  id: string;
  trip_id: string;
  media_id: string;
  thumb_small_id?: string;
  thumb_medium_id?: string;
  added_by: string;
  taken_at: string;
  latitude?: string;
  longitude?: string;
  caption: string;
  created_at: string;
};

export type AlbumPhotoWithURLsView = AlbumPhotoView & {
  original_url: string;
  thumb_small_url?: string;
  thumb_medium_url?: string;
};

export type AlbumShareTokenView = {
  id: string;
  trip_id: string;
  created_by: string;
  expires_at?: string;
  revoked_at?: string;
  revoked_by?: string;
  last_accessed_at?: string;
  created_at: string;
};

// AlbumCreateShareView carries the plaintext token exactly once — callers must
// surface it to the user immediately, since only the SHA-256 hash is persisted.
export type AlbumCreateShareView = {
  token: string;
  share: AlbumShareTokenView;
};

export type AlbumDownloadOriginalView = {
  url: string;
  expires_at: string;
};

export type AlbumPublicView = {
  photos: AlbumPhotoWithURLsView[];
  share: AlbumShareTokenView;
};

export const toAlbumInitUploadSlotView = (
  s: UpstreamAlbumInitUploadSlot,
): AlbumInitUploadSlotView => ({
  upload_id: s.upload_id,
  upload_url: s.upload_url,
  method: s.method,
  headers: s.headers,
  expires_at: s.expires_at,
});

export const toAlbumInitUploadView = (
  r: UpstreamAlbumInitUploadResponse,
): AlbumInitUploadView => ({
  slots: (r.slots ?? []).map(toAlbumInitUploadSlotView),
});

export const toAlbumPhotoView = (p: UpstreamAlbumPhoto): AlbumPhotoView => ({
  id: p.id,
  trip_id: p.trip_id,
  media_id: p.media_id,
  thumb_small_id: p.thumb_small_id,
  thumb_medium_id: p.thumb_medium_id,
  added_by: p.added_by,
  taken_at: p.taken_at,
  latitude: p.latitude,
  longitude: p.longitude,
  caption: p.caption,
  created_at: p.created_at,
});

export const toAlbumPhotoWithURLsView = (
  p: UpstreamAlbumPhotoWithURLs,
): AlbumPhotoWithURLsView => ({
  ...toAlbumPhotoView(p),
  original_url: p.original_url,
  thumb_small_url: p.thumb_small_url,
  thumb_medium_url: p.thumb_medium_url,
});

export const toAlbumShareTokenView = (
  s: UpstreamAlbumShareToken,
): AlbumShareTokenView => ({
  id: s.id,
  trip_id: s.trip_id,
  created_by: s.created_by,
  expires_at: s.expires_at,
  revoked_at: s.revoked_at,
  revoked_by: s.revoked_by,
  last_accessed_at: s.last_accessed_at,
  created_at: s.created_at,
});

export const toAlbumCreateShareView = (
  r: UpstreamAlbumCreateShareResponse,
): AlbumCreateShareView => ({
  token: r.token,
  share: toAlbumShareTokenView(r.share),
});

export const toAlbumDownloadOriginalView = (
  r: UpstreamAlbumDownloadOriginal,
): AlbumDownloadOriginalView => ({
  url: r.url,
  expires_at: r.expires_at,
});

export const toAlbumPublicView = (
  r: UpstreamAlbumPublicResponse,
): AlbumPublicView => ({
  photos: (r.photos ?? []).map(toAlbumPhotoWithURLsView),
  share: toAlbumShareTokenView(r.share),
});
