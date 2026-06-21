-- +goose Up
-- Adds geocoded location to itinerary items and priority/tags/sort_order to
-- todos. Itinerary already has sort_order; todos picks it up here so the
-- frontend can drag-reorder.

ALTER TABLE trip.itineraries
    ADD COLUMN IF NOT EXISTS latitude  DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;

ALTER TABLE trip.todos
    ADD COLUMN IF NOT EXISTS priority   TEXT       NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS tags       TEXT[]     NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS sort_order INTEGER    NOT NULL DEFAULT 0;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'todos_priority_check'
          AND table_schema = 'trip'
    ) THEN
        ALTER TABLE trip.todos
            ADD CONSTRAINT todos_priority_check
            CHECK (priority IN ('low', 'normal', 'high'));
    END IF;
END$$;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_todos_trip_sort ON trip.todos(trip_id, sort_order);
