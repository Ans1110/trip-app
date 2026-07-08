-- +goose Up
-- Media Service schema. Uploads go through a two-step handshake:
--   1) POST /media/upload/init  -> creates a media.upload_sessions row + returns
--      a short-lived presigned PUT URL. Object key is server-generated.
--   2) POST /media/upload/complete -> API does HeadObject against the bucket to
--      pull the *authoritative* size/mime/etag, then promotes the session to
--      a media.assets row. Client metadata is never trusted.
--
-- Incomplete sessions are GC'd by a background loop that RemoveObject's any
-- orphaned upload before deleting the session row.
--
-- chat.message.media_public_id (opaque Cloudinary handle) is retired in favor
-- of chat.message.media_id, a UUID FK into media.assets. ON DELETE SET NULL:
-- deleting a media asset does not cascade into the chat history, so we can
-- soft-delete media without wiping messages that reference them.

CREATE SCHEMA IF NOT EXISTS media;

CREATE TABLE IF NOT EXISTS media.assets (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID        NOT NULL,
    purpose     TEXT        NOT NULL,             -- 'chat' | 'avatar' | 'cover' | 'todo'
    bucket      TEXT        NOT NULL,
    object_key  TEXT        NOT NULL,
    mime        TEXT        NOT NULL,             -- from S3 HEAD, not client
    bytes       BIGINT      NOT NULL,             -- from S3 HEAD
    width       INTEGER,                          -- optional; populated later by processor
    height      INTEGER,
    etag        TEXT        NOT NULL,             -- S3 ETag; identity check on future dedupe
    sha256      BYTEA,                            -- fixed 32-byte digest; NULL until processor computes
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (bucket, object_key),
    CHECK (sha256 IS NULL OR octet_length(sha256) = 32)
);

CREATE INDEX IF NOT EXISTS idx_media_assets_owner
    ON media.assets (owner_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_media_assets_purpose
    ON media.assets (purpose, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS media.upload_sessions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id          UUID        NOT NULL,
    purpose           TEXT        NOT NULL,
    bucket            TEXT        NOT NULL,
    object_key        TEXT        NOT NULL,
    expected_mime     TEXT        NOT NULL,
    expected_max_bytes BIGINT     NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ NOT NULL,
    completed_at      TIMESTAMPTZ,
    media_id          UUID        REFERENCES media.assets(id) ON DELETE SET NULL,
    UNIQUE (bucket, object_key)
);

-- Partial index: GC scans only pending sessions.
CREATE INDEX IF NOT EXISTS idx_media_upload_sessions_pending
    ON media.upload_sessions (expires_at)
    WHERE completed_at IS NULL;

-- Chat: migrate media reference from opaque Cloudinary handle → media.assets FK.
-- ON DELETE SET NULL keeps chat history intact if the media row is later
-- removed (per policy: never synchronously delete media referenced by chat).
ALTER TABLE chat.message
    ADD COLUMN IF NOT EXISTS media_id UUID REFERENCES media.assets(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_chat_message_media_id
    ON chat.message (media_id)
    WHERE media_id IS NOT NULL;

ALTER TABLE chat.message
    DROP COLUMN IF EXISTS media_public_id;

-- +goose Down
ALTER TABLE chat.message
    ADD COLUMN IF NOT EXISTS media_public_id TEXT;
DROP INDEX IF EXISTS chat.idx_chat_message_media_id;
ALTER TABLE chat.message DROP COLUMN IF EXISTS media_id;

DROP INDEX IF EXISTS media.idx_media_upload_sessions_pending;
DROP TABLE IF EXISTS media.upload_sessions;
DROP INDEX IF EXISTS media.idx_media_assets_purpose;
DROP INDEX IF EXISTS media.idx_media_assets_owner;
DROP TABLE IF EXISTS media.assets;
DROP SCHEMA IF EXISTS media;
