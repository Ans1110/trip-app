-- Backfill the token_hash column on auth.email_verifications and auth.password_resets
-- for databases created before 002_auth_supplement.sql declared it. The earlier
-- migration uses CREATE TABLE IF NOT EXISTS, so an older table will not pick the
-- column up automatically — INSERTs from the auth handler then fail with
-- "column \"token_hash\" of relation ... does not exist". This migration is idempotent.

-- email_verifications
ALTER TABLE auth.email_verifications
    ADD COLUMN IF NOT EXISTS token_hash TEXT;

DELETE FROM auth.email_verifications WHERE token_hash IS NULL;

ALTER TABLE auth.email_verifications
    ALTER COLUMN token_hash SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_email_verifications_token_hash
    ON auth.email_verifications(token_hash);

ALTER TABLE auth.email_verifications
    DROP COLUMN IF EXISTS token;

-- password_resets
ALTER TABLE auth.password_resets
    ADD COLUMN IF NOT EXISTS token_hash TEXT;

DELETE FROM auth.password_resets WHERE token_hash IS NULL;

ALTER TABLE auth.password_resets
    ALTER COLUMN token_hash SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_password_resets_token_hash
    ON auth.password_resets(token_hash);

ALTER TABLE auth.password_resets
    DROP COLUMN IF EXISTS token;

-- user_sessions
ALTER TABLE auth.user_sessions
    ADD COLUMN IF NOT EXISTS refresh_token_hash TEXT;

ALTER TABLE auth.user_sessions
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;

DELETE FROM auth.user_sessions WHERE refresh_token_hash IS NULL;

ALTER TABLE auth.user_sessions
    ALTER COLUMN refresh_token_hash SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_sessions_refresh_token_hash
    ON auth.user_sessions(refresh_token_hash);

ALTER TABLE auth.user_sessions
    DROP CONSTRAINT IF EXISTS user_sessions_refresh_token_key;

ALTER TABLE auth.user_sessions
    DROP COLUMN IF EXISTS refresh_token;
