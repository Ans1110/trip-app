import "server-only";
import {
  UpstreamListProfileUsersResponse,
  UpstreamProfileRecommendationResponse,
  UpstreamProfileResponse,
  UpstreamProfileUserSummary,
} from "../upstream";

export type ProfileView = {
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

export type ProfileUserSummaryView = {
  user_id: string;
  username: string;
  name: string;
  avatar_url?: string;
};

export type ListProfileUsersView = {
  users: ProfileUserSummaryView[];
  next_cursor?: string;
};

export type ProfileRecommendationView = {
  users: ProfileUserSummaryView[];
};

export const toProfileView = (p: UpstreamProfileResponse): ProfileView => ({
  user_id: p.user_id,
  username: p.username,
  name: p.name,
  bio: p.bio,
  avatar_url: p.avatar_url,
  cover_image: p.cover_image,
  travel_tags: p.travel_tags ?? [],
  social_instagram: p.social_instagram,
  social_x: p.social_x,
  social_youtube: p.social_youtube,
  social_tiktok: p.social_tiktok,
  followers_count: p.followers_count ?? 0,
  following_count: p.following_count ?? 0,
  is_following: p.is_following,
  is_self: p.is_self,
  created_at: p.created_at,
});

export const toProfileUserSummaryView = (
  u: UpstreamProfileUserSummary,
): ProfileUserSummaryView => ({
  user_id: u.user_id,
  username: u.username,
  name: u.name,
  avatar_url: u.avatar_url,
});

export const toListProfileUsersView = (
  r: UpstreamListProfileUsersResponse,
): ListProfileUsersView => ({
  users: (r.users ?? []).map(toProfileUserSummaryView),
  next_cursor: r.next_cursor,
});

export const toProfileRecommendationView = (
  r: UpstreamProfileRecommendationResponse,
): ProfileRecommendationView => ({
  users: (r.users ?? []).map(toProfileUserSummaryView),
});
