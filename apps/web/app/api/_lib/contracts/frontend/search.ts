import "server-only";
import { UpstreamSearchPostsResponse } from "../upstream";
import { PostView, toPostView } from "./post";

export type SearchPostsView = {
  query: string;
  posts: PostView[];
  next_cursor?: string;
};

export const toSearchPostsView = (
  r: UpstreamSearchPostsResponse,
): SearchPostsView => ({
  query: r.query,
  posts: (r.posts ?? []).map(toPostView),
  next_cursor: r.next_cursor,
});
