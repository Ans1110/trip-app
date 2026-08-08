-- +goose Up
ALTER TABLE auth.mfa_configs DROP COLUMN IF EXISTS totp_secret;
ALTER TABLE auth.mfa_configs ADD COLUMN IF NOT EXISTS method VARCHAR(32) NOT NULL DEFAULT 'email';

-- +goose Down
ALTER TABLE auth.mfa_configs DROP COLUMN IF EXISTS method;
ALTER TABLE auth.mfa_configs ADD COLUMN IF NOT EXISTS totp_secret TEXT NOT NULL DEFAULT '';
