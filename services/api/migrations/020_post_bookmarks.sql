-- +goose Up
-- Private bookmarks — one row per (user, post). Cascade on both sides so a
-- deleted post or a deleted user releases the bookmark automatically. No
-- soft-delete column: unbookmarking hard-deletes the row.
CREATE TABLE IF NOT EXISTS post.bookmarks (
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    post_id     UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);

-- Keyset pagination for the /bookmarks page — newest first per user.
CREATE INDEX IF NOT EXISTS idx_bookmarks_user_created
    ON post.bookmarks (user_id, created_at DESC, post_id DESC);

-- +goose Down
DROP TABLE IF EXISTS post.bookmarks;
