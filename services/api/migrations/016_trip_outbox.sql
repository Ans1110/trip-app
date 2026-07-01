-- +goose Up
-- Transactional Outbox for realtime ops. When the realtime hub applies a
-- version-checked LWW update to an itinerary/todo row, it inserts an outbox
-- row in the SAME Postgres transaction. A background dispatcher polls
-- unclaimed rows (FOR UPDATE SKIP LOCKED) and XADDs them to Redis Stream
-- `events:trip` in pipelined batches. This closes the "wrote to Postgres but
-- crashed before XADD" hole and gives at-least-once delivery.
--
-- payload is stored as jsonb so consumers can query by op_type/from/etc.
-- without decoding a text blob.

CREATE TABLE IF NOT EXISTS trip.outbox (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id  UUID,                                   -- e.g. trip_id
    stream        TEXT        NOT NULL,                   -- Redis stream target
    trace_id      TEXT        NOT NULL DEFAULT '',
    payload       JSONB       NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at TIMESTAMPTZ,
    attempts      INTEGER     NOT NULL DEFAULT 0,
    last_error    TEXT        NOT NULL DEFAULT ''
);

-- Partial index: only unclaimed rows are indexed, so ORDER BY created_at
-- polling stays fast as the table grows (dispatched rows are effectively
-- archived — they can be periodically pruned by a separate GC job).
CREATE INDEX IF NOT EXISTS idx_trip_outbox_pending
    ON trip.outbox (created_at)
    WHERE dispatched_at IS NULL;

-- Aggregate lookup for debug / replay by trip.
CREATE INDEX IF NOT EXISTS idx_trip_outbox_aggregate
    ON trip.outbox (aggregate_id);

-- +goose Down
DROP TABLE IF EXISTS trip.outbox;
