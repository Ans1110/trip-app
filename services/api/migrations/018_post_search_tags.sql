-- +goose Up
-- Include tags in the FTS vector so tag queries hit the same GIN index as
-- title/content queries. Tags carry weight C — below title (A) and content
-- (B) — so a tag match doesn't outrank an in-body mention of the same term.
--
-- array_to_string(anyarray, text) is STABLE per pg_proc, which disqualifies
-- it from GENERATED ALWAYS AS. Wrap it in a user-declared IMMUTABLE helper
-- specialized to text[] — safe because textout is itself IMMUTABLE.
CREATE OR REPLACE FUNCTION post.tags_to_text(tags text[])
RETURNS text
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$ SELECT COALESCE(array_to_string(tags, ' '), '') $$;

ALTER TABLE post.posts DROP COLUMN search_vector;

ALTER TABLE post.posts
    ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(content, '')), 'B') ||
        setweight(to_tsvector('simple', post.tags_to_text(tags)), 'C')
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_posts_search
    ON post.posts USING GIN (search_vector)
    WHERE deleted_at IS NULL;

-- +goose Down
ALTER TABLE post.posts DROP COLUMN search_vector;

ALTER TABLE post.posts
    ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(content, '')), 'B')
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_posts_search
    ON post.posts USING GIN (search_vector)
    WHERE deleted_at IS NULL;

DROP FUNCTION IF EXISTS post.tags_to_text(text[]);
