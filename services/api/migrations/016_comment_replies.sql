-- +goose Up
-- Single-level threading: parent_id points at a top-level comment or NULL.
-- Denormalized reply_count on the parent row so list queries can render
-- "N replies" without a per-parent COUNT(*), and so tombstones (soft-deleted
-- parents with surviving replies) can be detected in the same WHERE clause.
ALTER TABLE post.comments
    ADD COLUMN parent_id UUID REFERENCES post.comments(id) ON DELETE CASCADE;

ALTER TABLE post.comments
    ADD COLUMN reply_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE post.comments
    ADD CONSTRAINT reply_count_nonneg CHECK (reply_count >= 0);

CREATE INDEX IF NOT EXISTS idx_comments_parent_created
    ON post.comments (parent_id, created_at ASC, id ASC)
    WHERE deleted_at IS NULL AND parent_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS post.idx_comments_parent_created;
ALTER TABLE post.comments DROP CONSTRAINT IF EXISTS reply_count_nonneg;
ALTER TABLE post.comments DROP COLUMN IF EXISTS reply_count;
ALTER TABLE post.comments DROP COLUMN IF EXISTS parent_id;
