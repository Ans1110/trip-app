-- +goose Up
-- Per-member hide flag on chat.room_members. When a user "deletes" a chat from
-- their list, we soft-remove *for them* by stamping hidden_at; the messages and
-- the room itself stay untouched. hidden_at is cleared when the user re-opens
-- the DM (EnsureDM) or when a new message lands in the room (writer flush).
ALTER TABLE chat.room_members
    ADD COLUMN IF NOT EXISTS hidden_at TIMESTAMPTZ NULL;

-- +goose Down
ALTER TABLE chat.room_members
    DROP COLUMN IF EXISTS hidden_at;
