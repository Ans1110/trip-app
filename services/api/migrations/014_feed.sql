-- +goose Up
-- Dedicated feed projection. Rows are written by the feed fan-out consumer
-- when POST_PUBLISHED (or future domain) events land on events:post; the
-- (user_id, published_at DESC, id) index supports keyset pagination for the
-- home feed. Feed is a projection only — sources of truth live in post/.
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

-- +goose Down
DROP SCHEMA IF EXISTS feed CASCADE;
