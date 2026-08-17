-- +goose Up
-- Location module (M1): user-scoped saved places with optional trip binding.
-- Search / nearby / route / weather endpoints are pass-through proxies against
-- Google Maps + OpenWeatherMap and don't need their own tables — results are
-- cached in Redis, not persisted.
CREATE SCHEMA IF NOT EXISTS location;

CREATE TABLE IF NOT EXISTS location.saved_places (
    id           UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID             NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    trip_id      UUID,
    name         TEXT             NOT NULL,
    address      TEXT             NOT NULL DEFAULT '',
    lat          DOUBLE PRECISION NOT NULL,
    lng          DOUBLE PRECISION NOT NULL,
    place_id     TEXT             NOT NULL DEFAULT '',
    category     VARCHAR(32)      NOT NULL DEFAULT '',
    notes        TEXT             NOT NULL DEFAULT '',
    color        VARCHAR(16)      NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ      NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    CHECK (lat >= -90 AND lat <= 90),
    CHECK (lng >= -180 AND lng <= 180)
);

CREATE INDEX IF NOT EXISTS idx_saved_places_user
    ON location.saved_places (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_saved_places_trip
    ON location.saved_places (trip_id, created_at DESC)
    WHERE deleted_at IS NULL AND trip_id IS NOT NULL;

-- Prevent the same user saving the same Google place twice into the same
-- context (either their global list, or a specific trip) — but scoped by
-- category so the /weather and /location lists can independently save the
-- same place under different tags.
CREATE UNIQUE INDEX IF NOT EXISTS uq_saved_places_user_trip_placeid_cat
    ON location.saved_places (
        user_id,
        COALESCE(trip_id, '00000000-0000-0000-0000-000000000000'::uuid),
        place_id,
        category
    )
    WHERE deleted_at IS NULL AND place_id <> '';

-- +goose Down
DROP SCHEMA IF EXISTS location CASCADE;
