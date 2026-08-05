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

export type UpstreamMediaPurpose =
  | "chat"
  | "avatar"
  | "cover"
  | "todo"
  | "album"
  | "album_thumb";

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
  media_filename?: string;
  client_msg_id?: string;
  is_edited: boolean;
  is_deleted: boolean;
  is_pinned: boolean;
  created_at: string;
};

export type UpstreamChatPeer = {
  id: string;
  name: string;
  email?: string;
  avatar_url?: string;
};

export type UpstreamChatRoom = {
  id: string;
  trip_id?: string;
  name: string;
  type: UpstreamChatRoomType;
  peer?: UpstreamChatPeer;
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

// ---- vote ----

export type UpstreamPollType = "location" | "time" | "custom";
export type UpstreamPollStatus = "open" | "closed";
export type UpstreamPollResultVisibility =
  | "always"
  | "after_vote"
  | "after_deadline";

export type UpstreamPollOption = {
  id: string;
  text: string;
  meta?: Record<string, unknown>;
  added_by: string;
  sort_order: number;
  created_at: string;
  count: number;
  voters?: string[];
};

export type UpstreamPoll = {
  id: string;
  trip_id: string;
  created_by: string;
  type: UpstreamPollType;
  title: string;
  description?: string;
  is_anonymous: boolean;
  max_choices: number;
  allow_option_add: boolean;
  result_visibility: UpstreamPollResultVisibility;
  deadline_at?: string;
  status: UpstreamPollStatus;
  closed_at?: string;
  created_at: string;
  updated_at: string;
  options: UpstreamPollOption[];
  my_choices: string[];
  results_visible: boolean;
  total_voters: number;
};

export type UpstreamPollOptionInput = {
  text: string;
  meta?: Record<string, unknown>;
};

export type UpstreamCreatePollPayload = {
  type?: UpstreamPollType | "";
  title: string;
  description?: string;
  is_anonymous?: boolean;
  max_choices?: number;
  allow_option_add?: boolean;
  result_visibility?: UpstreamPollResultVisibility | "";
  deadline_at?: string;
  options?: UpstreamPollOptionInput[];
};

export type UpstreamAddPollOptionPayload = {
  text: string;
  meta?: Record<string, unknown>;
};

export type UpstreamCastVotePayload = {
  option_ids: string[];
};

// ---- finance ----

// Go's shopspring/decimal marshals to JSON strings, so every monetary field is
// a decimal-formatted string ("12.50"), not a number.
export type UpstreamSplitStrategy = "equal" | "custom" | "percentage";
export type UpstreamSettlementStatus = "proposed" | "confirmed" | "cancelled";

export type UpstreamShareInput = {
  user_id: string;
  amount?: string;
  pct?: string;
};

export type UpstreamCreateExpensePayload = {
  paid_by: string;
  amount: string;
  currency: string;
  description?: string;
  category?: string;
  split_strategy: UpstreamSplitStrategy;
  participants: string[];
  shares?: UpstreamShareInput[];
  occurred_at?: string;
  receipt_asset_id?: string;
};

export type UpstreamUpdateExpensePayload = {
  paid_by?: string;
  amount?: string;
  currency?: string;
  description?: string;
  category?: string;
  split_strategy?: UpstreamSplitStrategy;
  participants?: string[];
  shares?: UpstreamShareInput[];
  occurred_at?: string;
  receipt_asset_id?: string;
  clear_receipt?: boolean;
};

export type UpstreamUpsertBudgetPayload = {
  category: string;
  amount: string;
  currency: string;
};

export type UpstreamUpsertFxRatePayload = {
  base: string;
  quote: string;
  rate: string;
  as_of?: string;
};

export type UpstreamConfirmSettlementPayload = {
  note?: string;
};

export type UpstreamManualSettlementIn = {
  payer_id: string;
  payee_id: string;
  amount: string;
  note?: string;
};

export type UpstreamProposeSettlementPayload = {
  auto: boolean;
  manual?: UpstreamManualSettlementIn[];
};

export type UpstreamExpenseShare = {
  user_id: string;
  amount: string;
  amount_base: string;
  share_pct?: string;
  paid_at?: string;
};

export type UpstreamSetSharePaidPayload = {
  paid: boolean;
};

export type UpstreamExpense = {
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
  split_strategy: UpstreamSplitStrategy;
  occurred_at: string;
  receipt_asset_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  shares: UpstreamExpenseShare[];
};

export type UpstreamBudget = {
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

export type UpstreamSettlement = {
  id: string;
  trip_id: string;
  payer_id: string;
  payee_id: string;
  amount: string;
  currency: string;
  status: UpstreamSettlementStatus;
  note: string;
  created_by: string;
  created_at: string;
  confirmed_at?: string;
  cancelled_at?: string;
};

export type UpstreamFxRate = {
  base: string;
  quote: string;
  rate: string;
  as_of: string;
  source: string;
};

export type UpstreamCategoryStat = {
  category: string;
  amount_base: string;
  count: number;
};

export type UpstreamPersonalStats = {
  trip_id: string;
  user_id: string;
  base_currency: string;
  total_paid: string;
  total_owed: string;
  net_balance: string;
  by_category: UpstreamCategoryStat[];
};

export type UpstreamTripBalance = {
  user_id: string;
  paid: string;
  owed: string;
  net: string;
};

export type UpstreamListFxRatesQuery = {
  base?: string;
};

export type UpstreamExportCsvQuery = {
  from?: string;
  to?: string;
};

// ---- album ----

export type UpstreamAlbumInitUploadItem = {
  mime: string;
  bytes: number;
};

export type UpstreamAlbumInitUploadPayload = {
  items: UpstreamAlbumInitUploadItem[];
};

export type UpstreamAlbumInitUploadSlot = {
  upload_id: string;
  upload_url: string;
  method: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type UpstreamAlbumInitUploadResponse = {
  slots: UpstreamAlbumInitUploadSlot[];
};

export type UpstreamAlbumCompletePhotoPayload = {
  upload_id: string;
  caption?: string;
};

export type UpstreamAlbumUpdatePhotoPayload = {
  caption?: string;
};

export type UpstreamAlbumPhoto = {
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

export type UpstreamAlbumPhotoWithURLs = UpstreamAlbumPhoto & {
  original_url: string;
  thumb_small_url?: string;
  thumb_medium_url?: string;
};

export type UpstreamAlbumShareToken = {
  id: string;
  trip_id: string;
  created_by: string;
  expires_at?: string;
  revoked_at?: string;
  revoked_by?: string;
  last_accessed_at?: string;
  created_at: string;
};

export type UpstreamAlbumCreateSharePayload = {
  expires_at?: string;
};

export type UpstreamAlbumCreateShareResponse = {
  token: string;
  share: UpstreamAlbumShareToken;
};

export type UpstreamAlbumDownloadOriginal = {
  url: string;
  expires_at: string;
};

export type UpstreamAlbumPublicResponse = {
  photos: UpstreamAlbumPhotoWithURLs[];
  share: UpstreamAlbumShareToken;
};

// ---- profile ----

export type UpstreamProfileResponse = {
  user_id: string;
  username: string;
  name: string;
  bio: string;
  avatar_url: string;
  cover_image: string;
  travel_tags: string[];
  social_instagram?: string;
  social_x?: string;
  social_youtube?: string;
  social_tiktok?: string;
  followers_count: number;
  following_count: number;
  is_following: boolean;
  is_self: boolean;
  created_at: string;
};

export type UpstreamUpdateProfilePayload = {
  username?: string;
  bio?: string;
  avatar_url?: string;
  cover_image?: string;
  travel_tags?: string[];
  social_instagram?: string;
  social_x?: string;
  social_youtube?: string;
  social_tiktok?: string;
};

export type UpstreamProfileUserSummary = {
  user_id: string;
  username: string;
  name: string;
  avatar_url?: string;
};

export type UpstreamListProfileUsersResponse = {
  users: UpstreamProfileUserSummary[];
  next_cursor?: string;
};

export type UpstreamProfileRecommendationResponse = {
  users: UpstreamProfileUserSummary[];
};

export type UpstreamProfilePageQuery = {
  cursor?: string;
  limit?: number;
};

// ---- post ----

export type UpstreamPostAuthorSummary = {
  user_id: string;
  username?: string;
  name: string;
  avatar_url?: string;
};

export type UpstreamPostResponse = {
  id: string;
  author: UpstreamPostAuthorSummary;
  title: string;
  content: string;
  cover_image?: string;
  tags: string[];
  like_count: number;
  comment_count: number;
  is_liked: boolean;
  is_author: boolean;
  published_at: string;
  updated_at: string;
};

export type UpstreamListPostsResponse = {
  posts: UpstreamPostResponse[];
  next_cursor?: string;
};

export type UpstreamCreatePostPayload = {
  title: string;
  content: string;
  cover_image?: string;
  tags?: string[];
};

export type UpstreamUpdatePostPayload = {
  title?: string;
  content?: string;
  cover_image?: string;
  tags?: string[];
};

export type UpstreamCommentResponse = {
  id: string;
  post_id: string;
  author: UpstreamPostAuthorSummary;
  content: string;
  is_author: boolean;
  created_at: string;
  updated_at: string;
};

export type UpstreamListCommentsResponse = {
  comments: UpstreamCommentResponse[];
  next_cursor?: string;
};

export type UpstreamCreateCommentPayload = { content: string };
export type UpstreamUpdateCommentPayload = { content: string };

export type UpstreamListPostsQuery = {
  author_id?: string;
  cursor?: string;
  limit?: number;
};

// ---- feed ----

export type UpstreamFeedItem = {
  id: string;
  event_type: string;
  subject_type: string;
  subject_id: string;
  published_at: string;
  post?: UpstreamPostResponse;
};

export type UpstreamFeedResponse = {
  items: UpstreamFeedItem[];
  next_cursor?: string;
};

// ---- search ----

export type UpstreamSearchPostsResponse = {
  query: string;
  posts: UpstreamPostResponse[];
  next_cursor?: string;
};

export type UpstreamSearchPostsQuery = {
  q: string;
  cursor?: string;
  limit?: number;
};
