CREATE SCHEMA IF NOT EXISTS friend;

CREATE TABLE IF NOT EXISTS friend.friend (
    user_id UUID NOT NULL,
    friend_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, friend_id)
);

CREATE INDEX IF NOT EXISTS idx_friend_user_id ON friend.friend(user_id);

CREATE TABLE IF NOT EXISTS friend.invitation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL,
    receiver_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(sender_id, receiver_id)
);

CREATE INDEX IF NOT EXISTS idx_invitation_sender_id ON friend.invitation(sender_id);
CREATE INDEX IF NOT EXISTS idx_invitation_receiver_id ON friend.invitation(receiver_id);