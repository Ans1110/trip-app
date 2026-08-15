-- +goose Up
CREATE SCHEMA IF NOT EXISTS trip;

-- visibility: trips are private by default; owners can toggle to public,
-- which fans out a TRIP_PUBLISHED event via platform.outbox that the profile
-- module consumes to build follower feeds.
CREATE TABLE IF NOT EXISTS trip.trip (
    id            UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      UUID             NOT NULL,
    title         TEXT             NOT NULL,
    description   TEXT             NOT NULL DEFAULT '',
    cover_image   TEXT             NOT NULL DEFAULT '',
    start_date    DATE             NOT NULL,
    end_date      DATE             NOT NULL,
    status        TEXT             NOT NULL DEFAULT 'planning',
    visibility    TEXT             NOT NULL DEFAULT 'private',
    time_zone     VARCHAR(64)      NOT NULL DEFAULT 'UTC',
    base_currency CHAR(3)          NOT NULL DEFAULT 'TWD',
    -- Trip-level location: single place the trip is centered on (used by the
    -- forecast strip on the trip detail page and shown next to the title on
    -- create/edit). Optional.
    location      TEXT             NOT NULL DEFAULT '',
    latitude      DOUBLE PRECISION,
    longitude     DOUBLE PRECISION,
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT trip_visibility_check CHECK (visibility IN ('private', 'public')),
    CONSTRAINT trip_latitude_check   CHECK (latitude  IS NULL OR (latitude  >= -90  AND latitude  <= 90)),
    CONSTRAINT trip_longitude_check  CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE INDEX IF NOT EXISTS idx_trip_owner_id   ON trip.trip(owner_id);
CREATE INDEX IF NOT EXISTS idx_trip_deleted_at ON trip.trip(deleted_at);
CREATE INDEX IF NOT EXISTS idx_trip_visibility_public
    ON trip.trip (visibility, created_at DESC)
    WHERE visibility = 'public' AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS trip.itineraries (
    id          UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     UUID             NOT NULL,
    day         INT              NOT NULL,
    title       TEXT             NOT NULL DEFAULT '',
    description TEXT             NOT NULL DEFAULT '',
    start_time  TIMESTAMPTZ,
    end_time    TIMESTAMPTZ,
    location    TEXT             NOT NULL DEFAULT '',
    latitude    DOUBLE PRECISION,
    longitude   DOUBLE PRECISION,
    sort_order  INT              NOT NULL DEFAULT 0,
    created_by  UUID,
    version     INTEGER          NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ      NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_itineraries_trip_id    ON trip.itineraries(trip_id);
CREATE INDEX IF NOT EXISTS idx_itineraries_deleted_at ON trip.itineraries(deleted_at);

CREATE TABLE IF NOT EXISTS trip.todos (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id      UUID        NOT NULL,
    assignee_id  UUID,
    title        TEXT        NOT NULL,
    is_completed BOOLEAN     NOT NULL DEFAULT FALSE,
    due_date     TIMESTAMPTZ,
    priority     TEXT        NOT NULL DEFAULT 'normal',
    tags         TEXT[]      NOT NULL DEFAULT '{}',
    sort_order   INTEGER     NOT NULL DEFAULT 0,
    created_by   UUID,
    version      INTEGER     NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    CONSTRAINT todos_priority_check CHECK (priority IN ('low', 'normal', 'high'))
);

CREATE INDEX IF NOT EXISTS idx_todos_trip_id    ON trip.todos(trip_id);
CREATE INDEX IF NOT EXISTS idx_todos_deleted_at ON trip.todos(deleted_at);
CREATE INDEX IF NOT EXISTS idx_todos_trip_sort  ON trip.todos(trip_id, sort_order);

CREATE TABLE IF NOT EXISTS trip.activities (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     UUID        NOT NULL,
    day         INT         NOT NULL,
    title       TEXT        NOT NULL DEFAULT '',
    description TEXT        NOT NULL DEFAULT '',
    start_time  TIMESTAMPTZ,
    end_time    TIMESTAMPTZ,
    location    TEXT,
    place_id    TEXT,
    category    VARCHAR(30) NOT NULL DEFAULT 'other',
    sort_order  INT         NOT NULL DEFAULT 0,
    created_by  UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activities_trip_day ON trip.activities(trip_id, day);

CREATE TABLE IF NOT EXISTS trip.packing_lists (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id    UUID        NOT NULL,
    name       TEXT        NOT NULL DEFAULT '',
    created_by UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_packing_lists_trip_id ON trip.packing_lists(trip_id);

CREATE TABLE IF NOT EXISTS trip.packing_items (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    packing_list_id UUID        NOT NULL REFERENCES trip.packing_lists(id) ON DELETE CASCADE,
    label           TEXT        NOT NULL,
    quantity        INT         NOT NULL DEFAULT 1,
    assignee_id     UUID,
    is_packed       BOOLEAN     NOT NULL DEFAULT FALSE,
    packed_at       TIMESTAMPTZ,
    packed_by       UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_packing_items_list_id ON trip.packing_items(packing_list_id);

CREATE TABLE IF NOT EXISTS trip.rooms (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id    UUID        NOT NULL UNIQUE REFERENCES trip.trip(id) ON DELETE CASCADE,
    room_code  CHAR(8)     NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rooms_trip_id ON trip.rooms(trip_id);

CREATE TABLE IF NOT EXISTS trip.room_members (
    room_id   UUID        NOT NULL REFERENCES trip.rooms(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role      VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id),
    CHECK (role IN ('admin', 'member'))
);

CREATE INDEX IF NOT EXISTS idx_room_members_user_id ON trip.room_members(user_id);

-- +goose Down
DROP SCHEMA IF EXISTS trip CASCADE;
