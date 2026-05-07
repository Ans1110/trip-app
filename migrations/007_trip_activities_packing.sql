-- Trip Day Plan activities and Packing Lists

CREATE TABLE IF NOT EXISTS trip.activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL,
    day INT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    location TEXT,
    place_id TEXT,
    category VARCHAR(30) NOT NULL DEFAULT 'other',
    sort_order INT NOT NULL DEFAULT 0,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activities_trip_day ON trip.activities(trip_id, day);

CREATE TABLE IF NOT EXISTS trip.packing_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_packing_lists_trip_id ON trip.packing_lists(trip_id);

CREATE TABLE IF NOT EXISTS trip.packing_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    packing_list_id UUID NOT NULL REFERENCES trip.packing_lists(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    assignee_id UUID,
    is_packed BOOLEAN NOT NULL DEFAULT FALSE,
    packed_at TIMESTAMPTZ,
    packed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_packing_items_list_id ON trip.packing_items(packing_list_id);