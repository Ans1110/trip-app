-- +goose Up
-- Travel post module: publish/edit/delete travel posts, like, comment, bookmark.
-- Denormalized like_count/comment_count/reply_count are maintained inside the
-- same tx as the underlying mutation so reads never require a COUNT(*) scan;
-- Redis caches the like count separately (post:{id}:likes) with the DB row as
-- the source of truth for cold reads and cache rebuilds.
CREATE SCHEMA IF NOT EXISTS post;

-- array_to_string(anyarray, text) is STABLE per pg_proc, which disqualifies
-- it from GENERATED ALWAYS AS. Wrap it in a user-declared IMMUTABLE helper
-- specialized to text[] — safe because textout is itself IMMUTABLE.
CREATE OR REPLACE FUNCTION post.tags_to_text(tags text[])
RETURNS text
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$ SELECT COALESCE(array_to_string(tags, ' '), '') $$;

CREATE TABLE IF NOT EXISTS post.posts (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id      UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title          TEXT        NOT NULL,
    content        TEXT        NOT NULL DEFAULT '',
    cover_image    TEXT        NOT NULL DEFAULT '',
    tags           TEXT[]      NOT NULL DEFAULT '{}',
    status         TEXT        NOT NULL DEFAULT 'published',
    like_count     INTEGER     NOT NULL DEFAULT 0,
    comment_count  INTEGER     NOT NULL DEFAULT 0,
    published_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    -- Generated tsvector for the search module. 'simple' config avoids
    -- language-specific stemming so mixed-language travel content behaves
    -- predictably. Title carries weight A, content B, tags C — a tag match
    -- doesn't outrank an in-body mention of the same term.
    search_vector  tsvector    GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(content, '')), 'B') ||
        setweight(to_tsvector('simple', post.tags_to_text(tags)), 'C')
    ) STORED,
    CHECK (like_count >= 0),
    CHECK (comment_count >= 0),
    CHECK (status IN ('draft', 'published', 'archived'))
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

-- Own-profile filter path: (author, status, ordered by published_at) so the
-- profile "Drafts" / "Archived" tabs stay index-only.
CREATE INDEX IF NOT EXISTS idx_posts_author_status_published
    ON post.posts (author_id, status, published_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS post.likes (
    post_id     UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_likes_user_created
    ON post.likes (user_id, created_at DESC);

-- Single-level threading: parent_id points at a top-level comment or NULL.
-- in_reply_to_id preserves the actual comment the user clicked "Reply" on,
-- even when storage flattens parent_id to the thread root — the FE uses it
-- to render replies immediately after their target instead of at the bottom.
CREATE TABLE IF NOT EXISTS post.comments (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id        UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    author_id      UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    parent_id      UUID        REFERENCES post.comments(id) ON DELETE CASCADE,
    in_reply_to_id UUID        REFERENCES post.comments(id) ON DELETE SET NULL,
    content        TEXT        NOT NULL,
    reply_count    INTEGER     NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    CONSTRAINT reply_count_nonneg CHECK (reply_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_comments_post_created
    ON post.comments (post_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_parent_created
    ON post.comments (parent_id, created_at ASC, id ASC)
    WHERE deleted_at IS NULL AND parent_id IS NOT NULL;

-- Private bookmarks — one row per (user, post). Cascade on both sides so a
-- deleted post or a deleted user releases the bookmark automatically. No
-- soft-delete column: unbookmarking hard-deletes the row.
CREATE TABLE IF NOT EXISTS post.bookmarks (
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    post_id     UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_user_created
    ON post.bookmarks (user_id, created_at DESC, post_id DESC);

-- +goose Down
DROP SCHEMA IF EXISTS post CASCADE;
