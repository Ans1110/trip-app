-- +goose Up
-- A trip has one implicit album; each photo references a media.assets row for
-- the original plus optional thumbnail rows. Share tokens grant unauthenticated
-- read access; only the SHA-256 hash is stored (plaintext returned once on
-- create), so a DB dump doesn't hand out working links.
CREATE SCHEMA IF NOT EXISTS album;

CREATE TABLE IF NOT EXISTS album.album_photo (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id         UUID        NOT NULL,
    media_id        UUID        NOT NULL REFERENCES media.assets(id) ON DELETE RESTRICT,
    thumb_small_id  UUID        REFERENCES media.assets(id) ON DELETE SET NULL,
    thumb_medium_id UUID        REFERENCES media.assets(id) ON DELETE SET NULL,
    added_by        UUID        NOT NULL,
    -- Prefers EXIF DateTimeOriginal, falls back to upload time.
    taken_at        TIMESTAMPTZ NOT NULL,
    latitude        NUMERIC(9,6),
    longitude       NUMERIC(9,6),
    caption         TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (trip_id, media_id)
);

CREATE INDEX IF NOT EXISTS idx_album_photo_trip_taken
    ON album.album_photo (trip_id, taken_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_album_photo_trip_added
    ON album.album_photo (trip_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS album.share_token (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id          UUID        NOT NULL,
    token_hash       BYTEA       NOT NULL,
    created_by       UUID        NOT NULL,
    expires_at       TIMESTAMPTZ,
    revoked_at       TIMESTAMPTZ,
    revoked_by       UUID,
    last_accessed_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (octet_length(token_hash) = 32),
    UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_album_share_token_trip
    ON album.share_token (trip_id, created_at DESC)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP SCHEMA IF EXISTS album CASCADE;
