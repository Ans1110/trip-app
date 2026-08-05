import "server-only";

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
