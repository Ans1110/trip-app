-- +goose Up
-- Optimistic concurrency control for realtime LWW ops.
-- Each itinerary / todo carries a monotonic `version` int. Clients send the
-- version they last saw; the realtime service updates only when versions match
-- and bumps version+1 atomically. Mismatch -> STALE_VERSION error -> client
-- re-fetches.

ALTER TABLE trip.itineraries
    ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE trip.todos
    ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE trip.itineraries DROP COLUMN IF EXISTS version;
ALTER TABLE trip.todos       DROP COLUMN IF EXISTS version;
