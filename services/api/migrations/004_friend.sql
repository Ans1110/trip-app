-- +goose Up
CREATE SCHEMA IF NOT EXISTS friend;

CREATE TABLE IF NOT EXISTS friend.friends (
    user_id    UUID        NOT NULL,
    friend_id  UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, friend_id),
    CHECK (user_id <> friend_id)
);

CREATE TABLE IF NOT EXISTS friend.invitations (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id   UUID        NOT NULL,
    receiver_id UUID        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending',
    message     TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (sender_id <> receiver_id),
    CONSTRAINT invitations_status_check
        CHECK (status IN ('pending', 'accepted', 'declined', 'cancelled')),
    UNIQUE (sender_id, receiver_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_friend_pair
    ON friend.invitations (
        LEAST(sender_id, receiver_id),
        GREATEST(sender_id, receiver_id)
    );

CREATE INDEX IF NOT EXISTS idx_invitations_pending
    ON friend.invitations(receiver_id, status)
    WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS friend.blocks (
    blocker_id UUID        NOT NULL,
    blocked_id UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked_id ON friend.blocks(blocked_id);

-- +goose Down
DROP SCHEMA IF EXISTS friend CASCADE;
