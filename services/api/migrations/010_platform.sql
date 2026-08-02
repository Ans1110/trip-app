-- +goose Up
-- Shared transactional outbox for realtime/event fan-out. Any bounded
-- context (trip, calendar, profile, ...) writes rows here in the same tx as
-- its domain mutation; the realtime dispatcher polls, XADDs to the row's
-- target Redis stream, and marks the row dispatched.
--
-- aggregate_type + aggregate_id lets consumers route by domain without
-- unmarshalling payload; op_type carries the wire operation (e.g.
-- "ITINERARY_UPDATE", "USER_FOLLOWED"). actor_id is nullable for system
-- events. payload is opaque JSONB owned by the producing context.
CREATE SCHEMA IF NOT EXISTS platform;

CREATE TABLE IF NOT EXISTS platform.outbox (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT        NOT NULL,
    aggregate_id   UUID,
    op_type        TEXT        NOT NULL,
    stream         TEXT        NOT NULL,
    actor_id       UUID,
    trace_id       TEXT        NOT NULL DEFAULT '',
    payload        JSONB       NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at  TIMESTAMPTZ,
    attempts       INTEGER     NOT NULL DEFAULT 0,
    last_error     TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_platform_outbox_pending
    ON platform.outbox (created_at)
    WHERE dispatched_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_platform_outbox_aggregate
    ON platform.outbox (aggregate_type, aggregate_id);

-- +goose Down
DROP SCHEMA IF EXISTS platform CASCADE;
