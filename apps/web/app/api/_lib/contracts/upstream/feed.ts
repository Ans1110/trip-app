import "server-only";
import { UpstreamPostResponse } from "./post";

export type UpstreamFeedItem = {
  id: string;
  event_type: string;
  subject_type: string;
  subject_id: string;
  published_at: string;
  post?: UpstreamPostResponse;
};

export type UpstreamFeedResponse = {
  items: UpstreamFeedItem[];
  next_cursor?: string;
};
