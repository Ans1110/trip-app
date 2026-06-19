-- Supplements 004_friend_schema.sql with: optional invitation message, status check,
-- a block list, and single/multi-use invitation tokens used by add-friend QR codes
-- and share links.

ALTER TABLE friend.invitations
    ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'invitations_status_check'
          AND table_schema = 'friend'
    ) THEN
        ALTER TABLE friend.invitations
            ADD CONSTRAINT invitations_status_check
            CHECK (status IN ('pending', 'accepted', 'declined', 'cancelled'));
    END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_invitations_pending
    ON friend.invitations(receiver_id, status)
    WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS friend.blocks (
    blocker_id UUID NOT NULL,
    blocked_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked_id ON friend.blocks(blocked_id);

CREATE TABLE IF NOT EXISTS friend.invitation_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    max_uses INTEGER NOT NULL DEFAULT 1,
    uses INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_invitation_tokens_user_id ON friend.invitation_tokens(user_id);
