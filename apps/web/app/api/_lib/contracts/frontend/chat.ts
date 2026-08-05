import "server-only";
import {
  UpstreamChatMessage,
  UpstreamChatMessageType,
  UpstreamChatReadReceipt,
  UpstreamChatRoom,
  UpstreamChatRoomType,
  UpstreamChatTicketResponse,
} from "../upstream";

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
