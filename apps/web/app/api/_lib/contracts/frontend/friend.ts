import "server-only";
import {
  UpstreamBlockResponse,
  UpstreamFriendResponse,
  UpstreamRequestResponse,
  UpstreamUserSummary,
} from "../upstream";

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
