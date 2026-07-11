-- +goose Up
-- Chat supplement:
--   * media reference columns on chat.message for image/file forwards from Media Service
--   * client_msg_id for sender-side idempotency (retries after WS reconnect must
--     not create duplicate messages)
--   * pair-key on DM rooms so ensure-DM lookups are cheap and race-safe
--   * fill the missing fk / index gaps the earlier migrations left

ALTER TABLE chat.message
    ADD COLUMN IF NOT EXISTS media_url        TEXT,
    ADD COLUMN IF NOT EXISTS media_public_id  TEXT,
    ADD COLUMN IF NOT EXISTS media_mime       TEXT,
    ADD COLUMN IF NOT EXISTS media_bytes      BIGINT,
    ADD COLUMN IF NOT EXISTS client_msg_id    TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_message_client_msg
    ON chat.message(room_id, sender_id, client_msg_id)
    WHERE client_msg_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_message_reply_to
    ON chat.message(reply_to)
    WHERE reply_to IS NOT NULL;

-- Deterministic pair key for DM rooms: sha1(min(user_a)||max(user_b)).
-- Enforced only on `type='dm'` so group rooms are unaffected.
ALTER TABLE chat.room
    ADD COLUMN IF NOT EXISTS dm_pair_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_room_dm_pair
    ON chat.room(dm_pair_key)
    WHERE type = 'dm' AND dm_pair_key IS NOT NULL;

-- Group rooms are 1:1 with trips; a partial unique index enforces that
-- without breaking DMs (which have trip_id IS NULL).
CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_room_trip
    ON chat.room(trip_id)
    WHERE type = 'group' AND trip_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS chat.uq_chat_room_trip;
DROP INDEX IF EXISTS chat.uq_chat_room_dm_pair;
ALTER TABLE chat.room DROP COLUMN IF EXISTS dm_pair_key;
DROP INDEX IF EXISTS chat.idx_chat_message_reply_to;
DROP INDEX IF EXISTS chat.uq_chat_message_client_msg;
ALTER TABLE chat.message
    DROP COLUMN IF EXISTS client_msg_id,
    DROP COLUMN IF EXISTS media_bytes,
    DROP COLUMN IF EXISTS media_mime,
    DROP COLUMN IF EXISTS media_public_id,
    DROP COLUMN IF EXISTS media_url;
