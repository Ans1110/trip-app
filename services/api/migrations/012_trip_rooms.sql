-- Room-centric collaboration model.
-- "Friend" and "Room" are decoupled: room membership has nothing to do with
-- friendship. Trip ownership stays on trip.owner_id; room admin/member is a
-- role on room_members (no separate admin_id column).

-- Legacy trip-level membership and invites are replaced by room-scoped ones.
DROP TABLE IF EXISTS trip.invite_tokens;
DROP TABLE IF EXISTS trip.members;

CREATE TABLE IF NOT EXISTS trip.rooms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     UUID NOT NULL UNIQUE REFERENCES trip.trip(id) ON DELETE CASCADE,
    room_code   CHAR(8) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rooms_trip_id ON trip.rooms(trip_id);

CREATE TABLE IF NOT EXISTS trip.room_members (
    room_id    UUID NOT NULL REFERENCES trip.rooms(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id),
    CHECK (role IN ('admin', 'member'))
);

CREATE INDEX IF NOT EXISTS idx_room_members_user_id ON trip.room_members(user_id);

CREATE TABLE IF NOT EXISTS trip.room_invite_tokens (
    token        TEXT PRIMARY KEY,
    room_id      UUID NOT NULL REFERENCES trip.rooms(id) ON DELETE CASCADE,
    created_by   UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    expires_at   TIMESTAMPTZ NOT NULL,
    max_uses     INT NOT NULL DEFAULT 0,  -- 0 = unlimited
    used_count   INT NOT NULL DEFAULT 0,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_room_invite_tokens_room_id ON trip.room_invite_tokens(room_id);

-- Soft delete for trip, itineraries, todos. GORM populates these via the
-- DeletedAt field on the struct; soft-deleted rows are filtered out of reads
-- automatically.
ALTER TABLE trip.trip          ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE trip.itineraries   ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE trip.todos         ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_trip_deleted_at        ON trip.trip(deleted_at);
CREATE INDEX IF NOT EXISTS idx_itineraries_deleted_at ON trip.itineraries(deleted_at);
CREATE INDEX IF NOT EXISTS idx_todos_deleted_at       ON trip.todos(deleted_at);

-- Track who created each itinerary item / todo so the service can enforce
-- "creator OR admin OR trip owner can edit/delete" without leaking permissions
-- through the API surface.
ALTER TABLE trip.itineraries ADD COLUMN IF NOT EXISTS created_by UUID;
ALTER TABLE trip.todos       ADD COLUMN IF NOT EXISTS created_by UUID;
