-- +goose Up
CREATE SCHEMA IF NOT EXISTS vote;

-- poll: the survey artifact itself. Lives inside a trip room; only room
-- members can see or vote.
--   type              — 'location' / 'time' / 'custom'; affects only the
--                        default option meta shape rendered by the FE.
--   is_anonymous      — if true, voter identities are stripped from any
--                        response beyond aggregate counts.
--   max_choices       — how many options a single voter may pick. 1 = single
--                        choice. Cast requests over this count are rejected.
--   allow_option_add  — if true, any room member can append options; else
--                        only the creator may.
--   result_visibility — 'always' (live counts), 'after_vote' (counts hidden
--                        until the caller has cast), 'after_deadline' (only
--                        after status=closed).
--   deadline_at       — optional auto-close time. A background sweeper flips
--                        status to 'closed' once now() >= deadline_at.
--   status            — 'open' | 'closed'. Closed polls reject new votes /
--                        new options.
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

-- Sweeper filter: only open polls with a deadline get scanned by the
-- auto-close job. Partial index keeps the scan cheap once most polls close.
CREATE INDEX IF NOT EXISTS idx_vote_poll_pending_deadline
    ON vote.poll(deadline_at)
    WHERE status = 'open' AND deadline_at IS NOT NULL;

-- option: one row per selectable choice. `meta` carries the type-specific
-- payload so we don't have to migrate the schema every time the FE adds a
-- new poll type (e.g. location coords, time-range endpoints).
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

-- Case-insensitive dedupe of option text within a single poll. CHECK cannot
-- express uniqueness in Postgres, so this must be a functional unique index
-- rather than a table-level constraint.
CREATE UNIQUE INDEX IF NOT EXISTS idx_vote_option_poll_text_unique
    ON vote.option (poll_id, LOWER(text));

-- ballot: one row per (poll, user, option) tuple. Composite PK enforces
-- idempotency — the same user picking the same option twice is a no-op.
-- To change a user's choices under max_choices > 1 semantics, the service
-- deletes their existing ballots and re-inserts the new set in one tx.
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
DROP TABLE IF EXISTS vote.ballot;
DROP TABLE IF EXISTS vote.option;
DROP TABLE IF EXISTS vote.poll;
DROP SCHEMA IF EXISTS vote;
