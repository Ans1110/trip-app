import "server-only";
import { UpstreamPostResponse } from "./post";

export type UpstreamFeedResponse = {
  posts: UpstreamPostResponse[];
  next_cursor?: string;
};
