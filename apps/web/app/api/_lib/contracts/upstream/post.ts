import "server-only";

export type UpstreamPostAuthorSummary = {
  user_id: string;
  username?: string;
  name: string;
  avatar_url?: string;
};

export type UpstreamPostStatus = "draft" | "published" | "archived";

export type UpstreamPostResponse = {
  id: string;
  author: UpstreamPostAuthorSummary;
  title: string;
  content: string;
  cover_image?: string;
  tags: string[];
  status: UpstreamPostStatus;
  like_count: number;
  comment_count: number;
  is_liked: boolean;
  is_bookmarked: boolean;
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
  status?: UpstreamPostStatus;
};

export type UpstreamUpdatePostPayload = {
  title?: string;
  content?: string;
  cover_image?: string;
  tags?: string[];
  status?: UpstreamPostStatus;
};

export type UpstreamCommentResponse = {
  id: string;
  post_id: string;
  parent_id?: string;
  in_reply_to_id?: string;
  author: UpstreamPostAuthorSummary;
  content: string;
  reply_count: number;
  is_author: boolean;
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
};

export type UpstreamListCommentsResponse = {
  comments: UpstreamCommentResponse[];
  next_cursor?: string;
};

export type UpstreamListRepliesResponse = {
  replies: UpstreamCommentResponse[];
  next_cursor?: string;
};

export type UpstreamCreateCommentPayload = {
  content: string;
  parent_id?: string;
};
export type UpstreamUpdateCommentPayload = { content: string };

export type UpstreamListPostsQuery = {
  author_id?: string;
  status?: UpstreamPostStatus;
  cursor?: string;
  limit?: number;
};

export type UpstreamBookmarksQuery = {
  cursor?: string;
  limit?: number;
};
