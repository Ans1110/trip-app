import "server-only";
import { UpstreamFeedResponse } from "../upstream";
import { PostView, toPostView } from "./post";

export type FeedView = {
  posts: PostView[];
  next_cursor?: string;
};

export const toFeedView = (r: UpstreamFeedResponse): FeedView => ({
  posts: (r.posts ?? []).map(toPostView),
  next_cursor: r.next_cursor,
});
