-- +goose Up
-- Travel post module: publish/edit/delete travel posts, like, comment.
-- Denormalized like_count/comment_count are maintained inside the same tx
-- as the underlying mutation so reads never require a COUNT(*) scan; Redis
-- caches the like count separately (post:{id}:likes) with the DB row as the
-- source of truth for cold reads and cache rebuilds.
CREATE SCHEMA IF NOT EXISTS post;

CREATE TABLE IF NOT EXISTS post.posts (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id      UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title          TEXT        NOT NULL,
    content        TEXT        NOT NULL DEFAULT '',
    cover_image    TEXT        NOT NULL DEFAULT '',
    tags           TEXT[]      NOT NULL DEFAULT '{}',
    like_count     INTEGER     NOT NULL DEFAULT 0,
    comment_count  INTEGER     NOT NULL DEFAULT 0,
    published_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    -- Generated tsvector for the search module. 'simple' config avoids
    -- language-specific stemming so mixed-language travel content behaves
    -- predictably. Title carries weight A, content weight B.
    search_vector  tsvector    GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(content, '')), 'B')
    ) STORED,
    CHECK (like_count >= 0),
    CHECK (comment_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_posts_author_published
    ON post.posts (author_id, published_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_posts_published_id
    ON post.posts (published_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_posts_search
    ON post.posts USING GIN (search_vector)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS post.likes (
    post_id     UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_likes_user_created
    ON post.likes (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS post.comments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    author_id   UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    content     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_comments_post_created
    ON post.comments (post_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP SCHEMA IF EXISTS post CASCADE;
