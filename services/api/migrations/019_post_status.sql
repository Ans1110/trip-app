-- +goose Up
-- Add a status lifecycle to post.posts. Existing rows are backfilled to
-- 'published' by the column default. Feed/search hydration filters to
-- published only; author-owned lists can pass through drafts/archived.
ALTER TABLE post.posts
    ADD COLUMN status TEXT NOT NULL DEFAULT 'published'
    CHECK (status IN ('draft', 'published', 'archived'));

-- Own-profile filter path: (author, status, ordered by published_at) so the
-- profile "Drafts" / "Archived" tabs stay index-only.
CREATE INDEX IF NOT EXISTS idx_posts_author_status_published
    ON post.posts (author_id, status, published_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_posts_author_status_published;
ALTER TABLE post.posts DROP COLUMN status;
