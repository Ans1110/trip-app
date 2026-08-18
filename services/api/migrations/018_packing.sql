-- +goose Up
-- Packing checklist (行李清單): trip-scoped, shared across all room members.
-- Every item belongs to a single trip; realtime fanout piggy-backs on the trip
-- WS channel so a member's socket receives packing edits on the same stream as
-- itinerary / vote updates. Packed state is per-user: presence of a row in
-- packing.item_pack means that user has packed that item.
CREATE SCHEMA IF NOT EXISTS packing;

CREATE TABLE IF NOT EXISTS packing.item (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id      UUID        NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    created_by   UUID        NOT NULL,
    name         TEXT        NOT NULL,
    quantity     INTEGER     NOT NULL DEFAULT 1,
    category     VARCHAR(32) NOT NULL DEFAULT '',
    note         TEXT        NOT NULL DEFAULT '',
    sort_order   INTEGER     NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (quantity >= 1)
);

CREATE INDEX IF NOT EXISTS idx_packing_item_trip
    ON packing.item (trip_id, sort_order ASC, created_at ASC);

CREATE TABLE IF NOT EXISTS packing.item_pack (
    item_id   UUID        NOT NULL REFERENCES packing.item(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL,
    packed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (item_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_packing_item_pack_user
    ON packing.item_pack (user_id);

-- +goose Down
DROP SCHEMA IF EXISTS packing CASCADE;
