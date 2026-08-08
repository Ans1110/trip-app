-- +goose Up
-- Admin/moderation: user-filed reports on posts/comments/users, resolved by
-- staff via admin endpoints. Ban / soft-delete actions themselves mutate the
-- source tables (auth.users, post.posts, post.comments) — this schema only
-- carries the report inbox + resolution trail. Actor decisions land in the
-- shared auth.audit_logs table.
CREATE SCHEMA IF NOT EXISTS admin;

CREATE TABLE IF NOT EXISTS admin.reports (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    subject_type    VARCHAR(16) NOT NULL,
    subject_id      UUID        NOT NULL,
    reason          VARCHAR(32) NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    resolution      TEXT        NOT NULL DEFAULT '',
    resolved_by     UUID        REFERENCES auth.users(id) ON DELETE SET NULL,
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reports_subject_check CHECK (subject_type IN ('post', 'comment', 'user')),
    CONSTRAINT reports_status_check  CHECK (status IN ('pending', 'resolved', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_reports_status_created
    ON admin.reports (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_subject
    ON admin.reports (subject_type, subject_id);

CREATE INDEX IF NOT EXISTS idx_reports_reporter
    ON admin.reports (reporter_id, created_at DESC);

-- One pending report per (reporter, subject) so a spam-click doesn't create
-- N duplicate rows in the moderator queue.
CREATE UNIQUE INDEX IF NOT EXISTS uq_reports_pending_per_reporter
    ON admin.reports (reporter_id, subject_type, subject_id)
    WHERE status = 'pending';

INSERT INTO auth.roles (name)
VALUES ('admin')
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DROP SCHEMA IF EXISTS admin CASCADE;
DELETE FROM auth.roles WHERE name = 'admin';
