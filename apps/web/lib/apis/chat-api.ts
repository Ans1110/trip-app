import { request } from "../utils";

export type ChatMessageType = "text" | "image" | "video" | "file" | "system";
export type ChatRoomType = "group" | "dm";

export type ChatMessage = {
  id: string;
  room_id: string;
  sender_id: string;
  content: string;
  type: ChatMessageType;
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

export type ChatPeer = {
  id: string;
  name: string;
  email?: string;
  avatar_url?: string;
};

export type ChatRoom = {
  id: string;
  trip_id?: string;
  name: string;
  type: ChatRoomType;
  peer?: ChatPeer;
  last_message?: ChatMessage;
  unread_count: number;
  created_at: string;
};

export type ChatReadReceipt = {
  room_id: string;
  user_id: string;
  last_read_id: string;
  updated_at: string;
};

export type ChatTicket = {
  ticket: string;
  expires_in: number;
};

export type EnsureDMInput = { peer_id: string };
export type MarkReadInput = { last_read_id: string };

export type ListMessagesQuery = {
  before?: string;
  limit?: number;
};

export const chatApi = {
  listRooms: (signal?: AbortSignal) =>
    request<ChatRoom[]>("/api/chat/rooms", { signal }),

  ensureDM: (input: EnsureDMInput, signal?: AbortSignal) =>
    request<ChatRoom>("/api/chat/rooms/dm", {
      method: "POST",
      json: input,
      signal,
    }),

  ensureTripRoom: (tripId: string, signal?: AbortSignal) =>
    request<ChatRoom>(
      `/api/chat/trips/${encodeURIComponent(tripId)}/room`,
      { method: "POST", signal },
    ),

  listMessages: (
    roomId: string,
    query: ListMessagesQuery = {},
    signal?: AbortSignal,
  ) => {
    const qs = new URLSearchParams();
    if (query.before) qs.set("before", query.before);
    if (typeof query.limit === "number") qs.set("limit", String(query.limit));
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<ChatMessage[]>(
      `/api/chat/rooms/${encodeURIComponent(roomId)}/messages${suffix}`,
      { signal },
    );
  },

  listReadReceipts: (roomId: string, signal?: AbortSignal) =>
    request<ChatReadReceipt[]>(
      `/api/chat/rooms/${encodeURIComponent(roomId)}/receipts`,
      { signal },
    ),

  markRead: (roomId: string, input: MarkReadInput, signal?: AbortSignal) =>
    request<ChatReadReceipt>(
      `/api/chat/rooms/${encodeURIComponent(roomId)}/read`,
      { method: "POST", json: input, signal },
    ),

  issueRoomTicket: (roomId: string, signal?: AbortSignal) =>
    request<ChatTicket>(
      `/api/chat/rooms/${encodeURIComponent(roomId)}/ticket`,
      { method: "POST", signal },
    ),

  hideRoom: (roomId: string, signal?: AbortSignal) =>
    request<null>(
      `/api/chat/rooms/${encodeURIComponent(roomId)}`,
      { method: "DELETE", signal },
    ),
};
