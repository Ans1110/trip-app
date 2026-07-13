-- +goose Up
-- Snapshot the original filename on the chat message so file bubbles can
-- render "quarterly-report.pdf" instead of "application/pdf". Mirrors the
-- existing pattern of caching media_mime/media_bytes on the row so history
-- rendering doesn't need to hit media.assets.
ALTER TABLE chat.message
    ADD COLUMN IF NOT EXISTS media_filename TEXT NULL;

-- +goose Down
ALTER TABLE chat.message
    DROP COLUMN IF EXISTS media_filename;
