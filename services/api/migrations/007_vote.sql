-- +goose Up
CREATE SCHEMA IF NOT EXISTS vote;

CREATE TABLE IF NOT EXISTS vote.poll (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id           UUID        NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    created_by        UUID        NOT NULL,
    type              TEXT        NOT NULL DEFAULT 'custom',
    title             TEXT        NOT NULL,
    description       TEXT        NOT NULL DEFAULT '',
    is_anonymous      BOOLEAN     NOT NULL DEFAULT FALSE,
    max_choices       INTEGER     NOT NULL DEFAULT 1,
    allow_option_add  BOOLEAN     NOT NULL DEFAULT FALSE,
    result_visibility TEXT        NOT NULL DEFAULT 'always',
    deadline_at       TIMESTAMPTZ,
    status            TEXT        NOT NULL DEFAULT 'open',
    closed_at         TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (type IN ('location', 'time', 'custom')),
    CHECK (result_visibility IN ('always', 'after_vote', 'after_deadline')),
    CHECK (status IN ('open', 'closed')),
    CHECK (max_choices >= 1)
);

CREATE INDEX IF NOT EXISTS idx_vote_poll_trip_id ON vote.poll(trip_id);

-- Partial index keeps the deadline-sweeper scan cheap once polls close.
CREATE INDEX IF NOT EXISTS idx_vote_poll_pending_deadline
    ON vote.poll(deadline_at)
    WHERE status = 'open' AND deadline_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS vote.option (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id    UUID        NOT NULL REFERENCES vote.poll(id) ON DELETE CASCADE,
    text       TEXT        NOT NULL,
    meta       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    added_by   UUID        NOT NULL,
    sort_order INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vote_option_poll_id ON vote.option(poll_id);

-- Case-insensitive dedupe of option text per poll; must be functional index.
CREATE UNIQUE INDEX IF NOT EXISTS idx_vote_option_poll_text_unique
    ON vote.option (poll_id, LOWER(text));

CREATE TABLE IF NOT EXISTS vote.ballot (
    poll_id    UUID        NOT NULL REFERENCES vote.poll(id) ON DELETE CASCADE,
    option_id  UUID        NOT NULL REFERENCES vote.option(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (poll_id, user_id, option_id)
);

CREATE INDEX IF NOT EXISTS idx_vote_ballot_option_id ON vote.ballot(option_id);
CREATE INDEX IF NOT EXISTS idx_vote_ballot_poll_user ON vote.ballot(poll_id, user_id);

-- +goose Down
DROP SCHEMA IF EXISTS vote CASCADE;
