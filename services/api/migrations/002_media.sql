-- +goose Up
CREATE SCHEMA IF NOT EXISTS media;

CREATE TABLE IF NOT EXISTS media.assets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID        NOT NULL,
    purpose    TEXT        NOT NULL,
    bucket     TEXT        NOT NULL,
    object_key TEXT        NOT NULL,
    mime       TEXT        NOT NULL,
    bytes      BIGINT      NOT NULL,
    width      INTEGER,
    height     INTEGER,
    etag       TEXT        NOT NULL,
    sha256     BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
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
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id           UUID        NOT NULL,
    purpose            TEXT        NOT NULL,
    bucket             TEXT        NOT NULL,
    object_key         TEXT        NOT NULL,
    expected_mime      TEXT        NOT NULL,
    expected_max_bytes BIGINT      NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    completed_at       TIMESTAMPTZ,
    media_id           UUID        REFERENCES media.assets(id) ON DELETE SET NULL,
    UNIQUE (bucket, object_key)
);

CREATE INDEX IF NOT EXISTS idx_media_upload_sessions_pending
    ON media.upload_sessions (expires_at)
    WHERE completed_at IS NULL;

-- +goose Down
DROP SCHEMA IF EXISTS media CASCADE;
