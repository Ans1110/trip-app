-- +goose Up
-- Public travel footprint: trips are private by default; owners can toggle
-- to public, which fans out a TRIP_PUBLISHED event via platform.outbox that
-- the profile module consumes to build follower feeds.
ALTER TABLE trip.trip
    ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'private';

ALTER TABLE trip.trip
    DROP CONSTRAINT IF EXISTS trip_visibility_check;

ALTER TABLE trip.trip
    ADD CONSTRAINT trip_visibility_check CHECK (visibility IN ('private', 'public'));

CREATE INDEX IF NOT EXISTS idx_trip_visibility_public
    ON trip.trip (visibility, created_at DESC)
    WHERE visibility = 'public' AND deleted_at IS NULL;

-- +goose Down
ALTER TABLE trip.trip DROP CONSTRAINT IF EXISTS trip_visibility_check;
ALTER TABLE trip.trip DROP COLUMN IF EXISTS visibility;
DROP INDEX IF EXISTS trip.idx_trip_visibility_public;
