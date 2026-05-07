CREATE SCHEMA IF NOT EXISTS chat;

CREATE TABLE IF NOT EXISTS chat.room (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID,
    name TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'group',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_room_trip_id ON chat.room(trip_id);

CREATE TABLE IF NOT EXISTS chat.message (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL,
    sender_id UUID NOT NULL,
    content TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'text',
    reply_to UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_message_room_id ON chat.message(room_id);
CREATE INDEX IF NOT EXISTS idx_chat_message_created_at ON chat.message(room_id, created_at DESC);

CREATE TABLE IF NOT EXISTS chat.read_receipts (
    room_id UUID NOT NULL,
    user_id UUID NOT NULL,
    last_read_id UUID NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id)
);