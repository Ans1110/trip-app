-- +goose Up
-- Feed becomes a read-time personalized ranking over post.posts + profile.follows.
-- The fan-out projection (feed.feed_items) and its consumer are gone; feed rows
-- are no longer written for POST_PUBLISHED events. See internal/feed for the new
-- score = recency_decay + follow_boost model.
DROP SCHEMA IF EXISTS feed CASCADE;

-- +goose Down
CREATE SCHEMA IF NOT EXISTS feed;

CREATE TABLE IF NOT EXISTS feed.feed_items (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    actor_id      UUID        NOT NULL,
    event_type    TEXT        NOT NULL,
    subject_type  TEXT        NOT NULL,
    subject_id    UUID        NOT NULL,
    published_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, event_type, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_feed_items_user_published
    ON feed.feed_items (user_id, published_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_feed_items_subject
    ON feed.feed_items (subject_type, subject_id);
