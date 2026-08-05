-- +goose Up
-- Preserves the actual comment the user clicked "Reply" on, even when
-- storage flattens parent_id to the thread root. The FE uses this to
-- render replies immediately after their target instead of at the
-- bottom of the thread.
ALTER TABLE post.comments
    ADD COLUMN in_reply_to_id UUID REFERENCES post.comments(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE post.comments DROP COLUMN IF EXISTS in_reply_to_id;
