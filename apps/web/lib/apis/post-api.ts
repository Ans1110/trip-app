import { request } from "../utils";

export type PostAuthorSummary = {
  user_id: string;
  username?: string;
  name: string;
  avatar_url?: string;
};

export type PostStatus = "draft" | "published" | "archived";

export type Post = {
  id: string;
  author: PostAuthorSummary;
  title: string;
  content: string;
  cover_image?: string;
  tags: string[];
  status: PostStatus;
  like_count: number;
  comment_count: number;
  is_liked: boolean;
  is_bookmarked: boolean;
  is_author: boolean;
  published_at: string;
  updated_at: string;
};

export type ListPosts = {
  posts: Post[];
  next_cursor?: string;
};

export type Comment = {
  id: string;
  post_id: string;
  parent_id?: string;
  in_reply_to_id?: string;
  author: PostAuthorSummary;
  content: string;
  reply_count: number;
  is_author: boolean;
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
};

export type ListComments = {
  comments: Comment[];
  next_cursor?: string;
};

export type ListReplies = {
  replies: Comment[];
  next_cursor?: string;
};

export type CreatePostInput = {
  title: string;
  content: string;
  cover_image?: string;
  tags?: string[];
  status?: PostStatus;
};

export type UpdatePostInput = {
  title?: string;
  content?: string;
  cover_image?: string;
  tags?: string[];
  status?: PostStatus;
};

export type CreateCommentInput = { content: string; parent_id?: string };
export type UpdateCommentInput = { content: string };

export type ListPostsQuery = {
  author_id?: string;
  status?: PostStatus;
  cursor?: string;
  limit?: number;
};

export type BookmarksQuery = { cursor?: string; limit?: number };

export type PostPageQuery = { cursor?: string; limit?: number };

const listPostsQS = (q?: ListPostsQuery): string => {
  if (!q) return "";
  const qs = new URLSearchParams();
  if (q.author_id) qs.set("author_id", q.author_id);
  if (q.status) qs.set("status", q.status);
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  const s = qs.toString();
  return s ? `?${s}` : "";
};

const pageQS = (q?: PostPageQuery): string => {
  if (!q) return "";
  const qs = new URLSearchParams();
  if (q.cursor) qs.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) qs.set("limit", String(q.limit));
  const s = qs.toString();
  return s ? `?${s}` : "";
};

export const postApi = {
  list: (query?: ListPostsQuery, signal?: AbortSignal) =>
    request<ListPosts>(`/api/posts${listPostsQS(query)}`, { signal }),

  get: (id: string, signal?: AbortSignal) =>
    request<Post>(`/api/posts/${encodeURIComponent(id)}`, { signal }),

  create: (input: CreatePostInput, signal?: AbortSignal) =>
    request<Post>(`/api/posts`, { method: "POST", json: input, signal }),

  update: (id: string, input: UpdatePostInput, signal?: AbortSignal) =>
    request<Post>(`/api/posts/${encodeURIComponent(id)}`, {
      method: "PATCH",
      json: input,
      signal,
    }),

  remove: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/posts/${encodeURIComponent(id)}`, {
      method: "DELETE",
      signal,
    }),

  like: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/posts/${encodeURIComponent(id)}/likes`, {
      method: "POST",
      signal,
    }),

  unlike: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/posts/${encodeURIComponent(id)}/likes`, {
      method: "DELETE",
      signal,
    }),

  bookmark: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/posts/${encodeURIComponent(id)}/bookmarks`, {
      method: "POST",
      signal,
    }),

  unbookmark: (id: string, signal?: AbortSignal) =>
    request<null>(`/api/posts/${encodeURIComponent(id)}/bookmarks`, {
      method: "DELETE",
      signal,
    }),

  listBookmarks: (query?: BookmarksQuery, signal?: AbortSignal) =>
    request<ListPosts>(`/api/bookmarks${pageQS(query)}`, { signal }),

  listComments: (postId: string, query?: PostPageQuery, signal?: AbortSignal) =>
    request<ListComments>(
      `/api/posts/${encodeURIComponent(postId)}/comments${pageQS(query)}`,
      { signal },
    ),

  createComment: (
    postId: string,
    input: CreateCommentInput,
    signal?: AbortSignal,
  ) =>
    request<Comment>(`/api/posts/${encodeURIComponent(postId)}/comments`, {
      method: "POST",
      json: input,
      signal,
    }),

  updateComment: (
    postId: string,
    commentId: string,
    input: UpdateCommentInput,
    signal?: AbortSignal,
  ) =>
    request<Comment>(
      `/api/posts/${encodeURIComponent(postId)}/comments/${encodeURIComponent(commentId)}`,
      { method: "PATCH", json: input, signal },
    ),

  deleteComment: (postId: string, commentId: string, signal?: AbortSignal) =>
    request<null>(
      `/api/posts/${encodeURIComponent(postId)}/comments/${encodeURIComponent(commentId)}`,
      { method: "DELETE", signal },
    ),

  listReplies: (
    postId: string,
    commentId: string,
    query?: PostPageQuery,
    signal?: AbortSignal,
  ) =>
    request<ListReplies>(
      `/api/posts/${encodeURIComponent(postId)}/comments/${encodeURIComponent(commentId)}/replies${pageQS(query)}`,
      { signal },
    ),
};
