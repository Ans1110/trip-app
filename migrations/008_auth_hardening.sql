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
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID,
    action      VARCHAR(64) NOT NULL,
    status      VARCHAR(16) NOT NULL,
    ip_address  TEXT,
    user_agent  TEXT,
    detail      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_user   ON auth.audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON auth.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created ON auth.audit_logs(created_at DESC);
