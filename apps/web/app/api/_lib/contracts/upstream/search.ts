import "server-only";
import { UpstreamPostResponse } from "./post";

export type UpstreamSearchPostsResponse = {
  query: string;
  posts: UpstreamPostResponse[];
  next_cursor?: string;
};

export type UpstreamSearchPostsQuery = {
  q: string;
  cursor?: string;
  limit?: number;
};
