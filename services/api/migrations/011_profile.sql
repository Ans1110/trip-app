-- +goose Up
-- Profile module: personal page, follow graph, and fan-out feed. All rows are
-- keyed by auth.users(id) so the profile is 1:1 with a user. Username lives
-- here (not auth.users) so identity concerns stay in auth and social concerns
-- stay here. Follow counts are app-managed inside the same tx as follows —
-- no triggers, so callers can reason about consistency by reading the code.
CREATE SCHEMA IF NOT EXISTS profile;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS profile.profiles (
    user_id          UUID        PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    username         CITEXT      NOT NULL UNIQUE,
    bio              TEXT        NOT NULL DEFAULT '',
    avatar_url       TEXT        NOT NULL DEFAULT '',
    cover_image      TEXT        NOT NULL DEFAULT '',
    travel_tags      TEXT[]      NOT NULL DEFAULT '{}',
    social_instagram TEXT        NOT NULL DEFAULT '',
    social_x         TEXT        NOT NULL DEFAULT '',
    social_youtube   TEXT        NOT NULL DEFAULT '',
    social_tiktok    TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS profile.follows (
    follower_id UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    followee_id UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id <> followee_id)
);

-- Newest-first queries: "who follows me" and "who I follow".
CREATE INDEX IF NOT EXISTS idx_follows_followee_created
    ON profile.follows (followee_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_follows_follower_created
    ON profile.follows (follower_id, created_at DESC);

-- Denormalized counters written in the same tx as follows/unfollows.
-- Not driven by triggers so the write path is explicit and testable.
CREATE TABLE IF NOT EXISTS profile.follow_counts (
    user_id         UUID        PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    followers_count INTEGER     NOT NULL DEFAULT 0,
    following_count INTEGER     NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (followers_count >= 0),
    CHECK (following_count >= 0)
);

-- Fan-out-on-write feed. One row per (subscriber, event). Populated by the
-- profile consumer reading TRIP_PUBLISHED events from platform.outbox. The
-- (user_id, published_at DESC, id) index supports keyset pagination.
CREATE TABLE IF NOT EXISTS profile.feed (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    actor_id      UUID        NOT NULL,
    event_type    TEXT        NOT NULL,
    trip_id       UUID,
    published_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, actor_id, event_type, trip_id)
);

CREATE INDEX IF NOT EXISTS idx_feed_user_published
    ON profile.feed (user_id, published_at DESC, id);

-- +goose Down
DROP SCHEMA IF EXISTS profile CASCADE;
