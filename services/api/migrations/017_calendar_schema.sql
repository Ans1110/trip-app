-- +goose Up
-- Calendar is an independent bounded context. It has NO calendars table —
-- each user's calendar is derived from `calendar.event_members` (the events
-- attached to them) plus `visibility`-based overlays for room / friend views.
--
-- source_type + source_id lets us plug new sources in later (flight, hotel,
-- booking...) without a schema change. Only ('user','trip') are used in v1;
-- others are values-only, no FK enforcement.
--
-- visibility is the ONLY authorization model — event_members controls "whose
-- calendar contains this event", not "who can see it".
CREATE SCHEMA IF NOT EXISTS calendar;

CREATE TABLE IF NOT EXISTS calendar.events (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by   UUID         NOT NULL,
    source_type  VARCHAR(32)  NOT NULL DEFAULT 'user',
    source_id    UUID,
    event_type   VARCHAR(32)  NOT NULL DEFAULT 'general',
    visibility   VARCHAR(16)  NOT NULL,
    title        TEXT         NOT NULL,
    description  TEXT         NOT NULL DEFAULT '',
    location     TEXT         NOT NULL DEFAULT '',
    start_at     TIMESTAMPTZ  NOT NULL,
    end_at       TIMESTAMPTZ  NOT NULL,
    time_zone    VARCHAR(64)  NOT NULL DEFAULT '',
    all_day      BOOLEAN      NOT NULL DEFAULT FALSE,
    color        VARCHAR(16)  NOT NULL DEFAULT '',
    version      INTEGER      NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    CHECK (end_at >= start_at),
    CHECK (visibility IN ('private','room','friends','public')),
    CHECK ((source_type = 'user') = (source_id IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_calendar_events_source
    ON calendar.events (source_type, source_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_calendar_events_created_by
    ON calendar.events (created_by)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_calendar_events_start_at
    ON calendar.events (start_at)
    WHERE deleted_at IS NULL;

-- One canonical shared event per trip.
CREATE UNIQUE INDEX IF NOT EXISTS uq_calendar_events_source_trip
    ON calendar.events (source_id)
    WHERE source_type = 'trip' AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS calendar.event_members (
    event_id   UUID         NOT NULL REFERENCES calendar.events(id) ON DELETE CASCADE,
    user_id    UUID         NOT NULL,
    role       VARCHAR(16)  NOT NULL DEFAULT 'member',
    joined_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id),
    CHECK (role IN ('owner','member'))
);

CREATE INDEX IF NOT EXISTS idx_calendar_event_members_user
    ON calendar.event_members (user_id);

-- Trip owns a time_zone so the auto-synced calendar event knows how to
-- interpret StartDate / EndDate as wall-clock day bounds.
ALTER TABLE trip.trip
    ADD COLUMN IF NOT EXISTS time_zone VARCHAR(64) NOT NULL DEFAULT 'UTC';

-- +goose Down
ALTER TABLE trip.trip DROP COLUMN IF EXISTS time_zone;
DROP TABLE IF EXISTS calendar.event_members;
DROP TABLE IF EXISTS calendar.events;
DROP SCHEMA IF EXISTS calendar;
