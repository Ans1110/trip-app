import { request } from "../utils";

export type ProfileUserSummary = {
  user_id: string;
  username: string;
  name: string;
  avatar_url?: string;
};

export type Profile = {
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

export type UpdateProfileInput = {
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

export type ListProfileUsers = {
  users: ProfileUserSummary[];
  next_cursor?: string;
};

export type PostAuthorSummary = {
  user_id: string;
  username?: string;
  name: string;
  avatar_url?: string;
};

export type Post = {
  id: string;
  author: PostAuthorSummary;
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

export type FeedItem = {
  id: string;
  event_type: string;
  subject_type: string;
  subject_id: string;
  published_at: string;
  post?: Post;
};

export type Feed = {
  items: FeedItem[];
  next_cursor?: string;
};

export type ProfileRecommendation = {
  users: ProfileUserSummary[];
};

export type PageQuery = { cursor?: string; limit?: number };

const pageQS = (q?: PageQuery): string => {
  if (!q) return "";
  const qs = new URLSearchParams();
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  const s = qs.toString();
  return s ? `?${s}` : "";
};

export const profileApi = {
  me: (signal?: AbortSignal) =>
    request<Profile>("/api/profiles/me", { signal }),

  updateMe: (input: UpdateProfileInput, signal?: AbortSignal) =>
    request<Profile>("/api/profiles/me", {
      method: "PATCH",
      json: input,
      signal,
    }),

  byUsername: (username: string, signal?: AbortSignal) =>
    request<Profile>(
      `/api/profiles/by-username/${encodeURIComponent(username)}`,
      { signal },
    ),

  follow: (userId: string, signal?: AbortSignal) =>
    request<null>(`/api/profiles/${encodeURIComponent(userId)}/follow`, {
      method: "POST",
      signal,
    }),

  unfollow: (userId: string, signal?: AbortSignal) =>
    request<null>(`/api/profiles/${encodeURIComponent(userId)}/follow`, {
      method: "DELETE",
      signal,
    }),

  followers: (userId: string, query?: PageQuery, signal?: AbortSignal) =>
    request<ListProfileUsers>(
      `/api/profiles/${encodeURIComponent(userId)}/followers${pageQS(query)}`,
      { signal },
    ),

  following: (userId: string, query?: PageQuery, signal?: AbortSignal) =>
    request<ListProfileUsers>(
      `/api/profiles/${encodeURIComponent(userId)}/following${pageQS(query)}`,
      { signal },
    ),

  recommendations: (limit?: number, signal?: AbortSignal) => {
    const qs = limit && limit > 0 ? `?limit=${limit}` : "";
    return request<ProfileRecommendation>(
      `/api/profiles/recommendations${qs}`,
      { signal },
    );
  },

  feed: (query?: PageQuery, signal?: AbortSignal) =>
    request<Feed>(`/api/feed${pageQS(query)}`, { signal }),
};
