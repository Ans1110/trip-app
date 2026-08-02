-- +goose Up
CREATE SCHEMA IF NOT EXISTS chat;

CREATE TABLE IF NOT EXISTS chat.room (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     UUID,
    name        TEXT        NOT NULL DEFAULT '',
    type        TEXT        NOT NULL DEFAULT 'group',
    dm_pair_key TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_room_trip_id ON chat.room(trip_id);

-- Enforced only on 'dm' rooms so group rooms are unaffected.
CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_room_dm_pair
    ON chat.room(dm_pair_key)
    WHERE type = 'dm' AND dm_pair_key IS NOT NULL;

-- Group rooms are 1:1 with trips; DMs carry trip_id IS NULL so are excluded.
CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_room_trip
    ON chat.room(trip_id)
    WHERE type = 'group' AND trip_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS chat.message (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id        UUID        NOT NULL,
    sender_id      UUID        NOT NULL,
    content        TEXT        NOT NULL,
    type           TEXT        NOT NULL DEFAULT 'text',
    reply_to       UUID,
    is_edited      BOOLEAN     NOT NULL DEFAULT FALSE,
    edited_at      TIMESTAMPTZ,
    is_deleted     BOOLEAN     NOT NULL DEFAULT FALSE,
    deleted_at     TIMESTAMPTZ,
    is_pinned      BOOLEAN     NOT NULL DEFAULT FALSE,
    -- media_url/mime/bytes/filename are cached on the row so history rendering
    -- doesn't need to fan out to media.assets. media_id is the authoritative FK;
    -- ON DELETE SET NULL keeps chat history intact if the asset is later removed.
    media_id       UUID        REFERENCES media.assets(id) ON DELETE SET NULL,
    media_url      TEXT,
    media_mime     TEXT,
    media_bytes    BIGINT,
    media_filename TEXT,
    client_msg_id  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_message_room_id    ON chat.message(room_id);
CREATE INDEX IF NOT EXISTS idx_chat_message_created_at ON chat.message(room_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_message_client_msg
    ON chat.message(room_id, sender_id, client_msg_id)
    WHERE client_msg_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_message_reply_to
    ON chat.message(reply_to)
    WHERE reply_to IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_message_media_id
    ON chat.message (media_id)
    WHERE media_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS chat.read_receipts (
    room_id      UUID        NOT NULL,
    user_id      UUID        NOT NULL,
    last_read_id UUID        NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id)
);

CREATE TABLE IF NOT EXISTS chat.reaction (
    message_id UUID        NOT NULL,
    user_id    UUID        NOT NULL,
    emoji      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX IF NOT EXISTS idx_reaction_message_id ON chat.reaction(message_id);

CREATE TABLE IF NOT EXISTS chat.room_members (
    room_id   UUID        NOT NULL REFERENCES chat.room(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Per-member soft-hide: user "deletes" the chat from their list without
    -- affecting other members or the underlying messages. Cleared on re-open
    -- or when a new message lands.
    hidden_at TIMESTAMPTZ,
    PRIMARY KEY (room_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_room_members_user_id ON chat.room_members(user_id);

-- +goose Down
DROP SCHEMA IF EXISTS chat CASCADE;
