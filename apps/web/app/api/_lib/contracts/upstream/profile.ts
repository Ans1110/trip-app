import "server-only";

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
