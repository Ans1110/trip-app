CREATE SCHEMA IF NOT EXISTS trip;

CREATE TABLE IF NOT EXISTS trip.trip (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cover_image TEXT NOT NULL DEFAULT '',
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'planning',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trip_owner_id ON trip.trip(owner_id);

CREATE TABLE IF NOT EXISTS trip.itineraries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL,
    day INT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    location TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_itineraries_trip_id ON trip.itineraries(trip_id);

CREATE TABLE IF NOT EXISTS trip.todos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL,
    assignee_id UUID,
    title TEXT NOT NULL,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    due_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_todos_trip_id ON trip.todos(trip_id);

CREATE TABLE IF NOT EXISTS trip.invite_tokens (
    token TEXT PRIMARY KEY,
    trip_id UUID NOT NULL,
    created_by UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_count INT NOT NULL DEFAULT 0,
    max_uses INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_invite_tokens_trip_id ON trip.invite_tokens(trip_id);

CREATE TABLE IF NOT EXISTS trip.members (
    trip_id    UUID NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (trip_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_trip_members_user_id ON trip.members(user_id);

