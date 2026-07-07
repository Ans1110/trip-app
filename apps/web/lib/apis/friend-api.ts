import { request } from "../utils";

export type UserSummary = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
};

export type Friend = {
  user: UserSummary;
  created_at: string;
};

export type FriendRequest = {
  id: string;
  status: string;
  message?: string;
  sender: UserSummary;
  receiver: UserSummary;
  created_at: string;
  updated_at: string;
};

export type FriendBlock = {
  user: UserSummary;
  created_at: string;
};

export type SendRequestInput = { receiver_id: string; message?: string };
export type BlockUserInput = { target_id: string };

export const friendApi = {
  list: (signal?: AbortSignal) => request<Friend[]>("/api/friends", { signal }),

  remove: (friendId: string, signal?: AbortSignal) =>
    request<null>(`/api/friends/${encodeURIComponent(friendId)}`, {
      method: "DELETE",
      signal,
    }),

  search: (q: string, limit?: number, signal?: AbortSignal) => {
    const qs = new URLSearchParams({ q });
    if (limit && limit > 0) qs.set("limit", String(limit));
    return request<UserSummary[]>(`/api/friends/search?${qs}`, { signal });
  },

  incomingRequests: (signal?: AbortSignal) =>
    request<FriendRequest[]>("/api/friends/requests", { signal }),

  outgoingRequests: (signal?: AbortSignal) =>
    request<FriendRequest[]>("/api/friends/requests/outgoing", { signal }),

  sendRequest: (input: SendRequestInput, signal?: AbortSignal) =>
    request<FriendRequest>("/api/friends/requests", {
      method: "POST",
      json: input,
      signal,
    }),

  acceptRequest: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/friends/requests/${encodeURIComponent(id)}/accept`, {
      method: "POST",
      signal,
    }),

  declineRequest: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/friends/requests/${encodeURIComponent(id)}/decline`, {
      method: "POST",
      signal,
    }),

  cancelRequest: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/friends/requests/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  listBlocks: (signal?: AbortSignal) =>
    request<FriendBlock[]>("/api/friends/blocks", { signal }),

  blockUser: (input: BlockUserInput, signal?: AbortSignal) =>
    request<null>("/api/friends/blocks", {
      method: "POST",
      json: input,
      signal,
    }),

  unblockUser: (targetId: string, signal?: AbortSignal) =>
    request<null>(`/api/friends/blocks/${encodeURIComponent(targetId)}`, {
      method: "DELETE",
      signal,
    }),
};
