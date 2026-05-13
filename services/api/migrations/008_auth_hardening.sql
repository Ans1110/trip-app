-- Auth hardening: device fingerprint, RBAC roles, audit log

ALTER TABLE auth.user_sessions
    ADD COLUMN IF NOT EXISTS device_fingerprint VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_user_sessions_fingerprint
    ON auth.user_sessions(device_fingerprint);

CREATE TABLE IF NOT EXISTS auth.roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(64) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth.user_roles (
    user_id    UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES auth.roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user ON auth.user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON auth.user_roles(role_id);

CREATE TABLE IF NOT EXISTS auth.audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    actor_user_id   UUID,
    target_user_id  UUID,

    action          VARCHAR(64) NOT NULL,
    status          VARCHAR(32) NOT NULL,

    resource_type   VARCHAR(64),
    resource_id     TEXT,

    ip_address      INET,
    user_agent      TEXT,

    request_id      UUID,
    trace_id        TEXT,

    detail          JSONB,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_actor    ON auth.audit_logs(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_target   ON auth.audit_logs(target_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action   ON auth.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON auth.audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_created  ON auth.audit_logs(created_at DESC);
