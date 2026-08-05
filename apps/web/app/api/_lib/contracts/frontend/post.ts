import "server-only";
import {
  UpstreamCommentResponse,
  UpstreamListCommentsResponse,
  UpstreamListPostsResponse,
  UpstreamListRepliesResponse,
  UpstreamPostAuthorSummary,
  UpstreamPostResponse,
} from "../upstream";

export type PostAuthorSummaryView = {
  user_id: string;
  username?: string;
  name: string;
  avatar_url?: string;
};

export type PostView = {
  id: string;
  author: PostAuthorSummaryView;
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

export type ListPostsView = {
  posts: PostView[];
  next_cursor?: string;
};

export type CommentView = {
  id: string;
  post_id: string;
  parent_id?: string;
  in_reply_to_id?: string;
  author: PostAuthorSummaryView;
  content: string;
  reply_count: number;
  is_author: boolean;
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
};

export type ListCommentsView = {
  comments: CommentView[];
  next_cursor?: string;
};

export type ListRepliesView = {
  replies: CommentView[];
  next_cursor?: string;
};

export const toPostAuthorSummaryView = (
  a: UpstreamPostAuthorSummary,
): PostAuthorSummaryView => ({
  user_id: a.user_id,
  username: a.username,
  name: a.name,
  avatar_url: a.avatar_url,
});

export const toPostView = (p: UpstreamPostResponse): PostView => ({
  id: p.id,
  author: toPostAuthorSummaryView(p.author),
  title: p.title,
  content: p.content,
  cover_image: p.cover_image,
  tags: p.tags ?? [],
  like_count: p.like_count ?? 0,
  comment_count: p.comment_count ?? 0,
  is_liked: p.is_liked,
  is_author: p.is_author,
  published_at: p.published_at,
  updated_at: p.updated_at,
});

export const toListPostsView = (
  r: UpstreamListPostsResponse,
): ListPostsView => ({
  posts: (r.posts ?? []).map(toPostView),
  next_cursor: r.next_cursor,
});

export const toCommentView = (c: UpstreamCommentResponse): CommentView => ({
  id: c.id,
  post_id: c.post_id,
  parent_id: c.parent_id,
  in_reply_to_id: c.in_reply_to_id,
  author: toPostAuthorSummaryView(c.author),
  content: c.content,
  reply_count: c.reply_count ?? 0,
  is_author: c.is_author,
  is_deleted: c.is_deleted ?? false,
  created_at: c.created_at,
  updated_at: c.updated_at,
});

export const toListCommentsView = (
  r: UpstreamListCommentsResponse,
): ListCommentsView => ({
  comments: (r.comments ?? []).map(toCommentView),
  next_cursor: r.next_cursor,
});

export const toListRepliesView = (
  r: UpstreamListRepliesResponse,
): ListRepliesView => ({
  replies: (r.replies ?? []).map(toCommentView),
  next_cursor: r.next_cursor,
});
