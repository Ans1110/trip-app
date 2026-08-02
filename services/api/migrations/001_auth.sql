-- +goose Up
CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE IF NOT EXISTS auth.users (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 TEXT         UNIQUE NOT NULL,
    name                  TEXT         NOT NULL DEFAULT '',
    avatar_url            TEXT         NOT NULL DEFAULT '',
    password_hash         TEXT,
    is_blocked            BOOLEAN      NOT NULL DEFAULT FALSE,
    is_verified           BOOLEAN      NOT NULL DEFAULT FALSE,
    status                VARCHAR(20)  NOT NULL DEFAULT 'active',
    deactivated_at        TIMESTAMPTZ,
    deletion_scheduled_at TIMESTAMPTZ,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth.providers (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider    TEXT        NOT NULL,
    provider_id TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_providers_user_id ON auth.providers(user_id);

CREATE TABLE IF NOT EXISTS auth.email_verifications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_email_verif_user ON auth.email_verifications(user_id);

CREATE TABLE IF NOT EXISTS auth.password_resets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pwd_reset_user ON auth.password_resets(user_id);

CREATE TABLE IF NOT EXISTS auth.mfa_configs (
    user_id     UUID        PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    totp_secret TEXT        NOT NULL,
    is_enabled  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth.user_sessions (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    device_name        TEXT,
    device_type        VARCHAR(20),
    device_fingerprint VARCHAR(64),
    ip_address         INET,
    user_agent         TEXT,
    refresh_token_hash TEXT        NOT NULL UNIQUE,
    last_active_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user        ON auth.user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_fingerprint ON auth.user_sessions(device_fingerprint);

CREATE TABLE IF NOT EXISTS auth.roles (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth.user_roles (
    user_id    UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role_id    UUID        NOT NULL REFERENCES auth.roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user ON auth.user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON auth.user_roles(role_id);

CREATE TABLE IF NOT EXISTS auth.audit_logs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id  UUID,
    target_user_id UUID,
    action         VARCHAR(64) NOT NULL,
    status         VARCHAR(32) NOT NULL,
    resource_type  VARCHAR(64),
    resource_id    TEXT,
    ip_address     INET,
    user_agent     TEXT,
    request_id     UUID,
    trace_id       TEXT,
    detail         JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_actor    ON auth.audit_logs(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_target   ON auth.audit_logs(target_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action   ON auth.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON auth.audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_created  ON auth.audit_logs(created_at DESC);

-- +goose Down
DROP SCHEMA IF EXISTS auth CASCADE;
