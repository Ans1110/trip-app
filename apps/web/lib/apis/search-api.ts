import { request } from "../utils";
import type { Post } from "./post-api";

export type SearchPosts = {
  query: string;
  posts: Post[];
  next_cursor?: string;
};

export type SearchPostsQuery = {
  q: string;
  cursor?: string;
  limit?: number;
};

const qs = (q: SearchPostsQuery): string => {
  const p = new URLSearchParams();
  p.set("q", q.q);
  if (q.cursor) p.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) p.set("limit", String(q.limit));
  return p.toString();
};

export const searchApi = {
  posts: (query: SearchPostsQuery, signal?: AbortSignal) =>
    request<SearchPosts>(`/api/search/posts?${qs(query)}`, { signal }),
};
