import "server-only";
import { UpstreamFeedItem, UpstreamFeedResponse } from "../upstream";
import { PostView, toPostView } from "./post";

export type FeedItemView = {
  id: string;
  event_type: string;
  subject_type: string;
  subject_id: string;
  published_at: string;
  post?: PostView;
};

export type FeedView = {
  items: FeedItemView[];
  next_cursor?: string;
};

export const toFeedItemView = (i: UpstreamFeedItem): FeedItemView => ({
  id: i.id,
  event_type: i.event_type,
  subject_type: i.subject_type,
  subject_id: i.subject_id,
  published_at: i.published_at,
  post: i.post ? toPostView(i.post) : undefined,
});

export const toFeedView = (r: UpstreamFeedResponse): FeedView => ({
  items: (r.items ?? []).map(toFeedItemView),
  next_cursor: r.next_cursor,
});
