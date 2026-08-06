import { request } from "../utils";
import type { Post } from "./post-api";

export type Feed = {
  posts: Post[];
  next_cursor?: string;
};

export type FeedPageQuery = { cursor?: string; limit?: number };

const qs = (q?: FeedPageQuery): string => {
  if (!q) return "";
  const p = new URLSearchParams();
  if (q.cursor) p.set("cursor", q.cursor);
  if (q.limit && q.limit > 0) p.set("limit", String(q.limit));
  const s = p.toString();
  return s ? `?${s}` : "";
};

export const feedApi = {
  list: (query?: FeedPageQuery, signal?: AbortSignal) =>
    request<Feed>(`/api/feed${qs(query)}`, { signal }),
};
