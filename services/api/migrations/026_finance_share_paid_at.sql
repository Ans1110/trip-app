-- +goose Up
-- Track per-share settlement on the server so every trip member sees the same
-- "paid" ticks. NULL = not yet paid, non-NULL = paid at that instant. Kept as
-- a nullable timestamp (rather than a boolean) so the row also carries when
-- the toggle happened, which we may surface in the UI later.
ALTER TABLE finance.expense_share
    ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE finance.expense_share
    DROP COLUMN IF EXISTS paid_at;
