-- +goose Up
CREATE SCHEMA IF NOT EXISTS finance;

-- Trip gets a base_currency so expenses in mixed currencies can be aggregated
-- for settlement and stats. Default 'USD' so existing rows are valid; the FE
-- lets the trip owner change it.
ALTER TABLE trip.trip
    ADD COLUMN IF NOT EXISTS base_currency CHAR(3) NOT NULL DEFAULT 'USD';

-- expense: one line item per purchase. Money is NUMERIC(18,4) to keep enough
-- precision for micro-currencies and downstream percentage math without
-- forcing every consumer to use floats. amount_base + rate_to_base are
-- denormalized at write time so reports don't need a per-row FX lookup;
-- rate_to_base is NULL when currency == trip base_currency.
--   split_strategy — 'equal' | 'custom' | 'percentage'; determines how the
--                    accompanying finance.expense_share rows were derived.
--   occurred_at    — when the expense actually happened (may pre-date created_at
--                    for backdated entries); FX lookup keys off this timestamp.
--   receipt_asset_id — optional pointer into media.asset. media ownership is
--                    checked at attach time and never mirrored here.
CREATE TABLE IF NOT EXISTS finance.expense (
    id                UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id           UUID           NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    paid_by           UUID           NOT NULL,
    amount            NUMERIC(18,4)  NOT NULL,
    currency          CHAR(3)        NOT NULL,
    amount_base       NUMERIC(18,4)  NOT NULL,
    rate_to_base      NUMERIC(20,10),
    description       TEXT           NOT NULL DEFAULT '',
    category          TEXT           NOT NULL DEFAULT 'other',
    split_strategy    TEXT           NOT NULL DEFAULT 'equal',
    occurred_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    receipt_asset_id  UUID,
    created_by        UUID           NOT NULL,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    CHECK (amount > 0),
    CHECK (amount_base > 0),
    CHECK (split_strategy IN ('equal', 'custom', 'percentage'))
);

CREATE INDEX IF NOT EXISTS idx_finance_expense_trip_id
    ON finance.expense(trip_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_finance_expense_paid_by
    ON finance.expense(paid_by) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_finance_expense_occurred_at
    ON finance.expense(trip_id, occurred_at DESC) WHERE deleted_at IS NULL;

-- expense_share: the per-participant slice of an expense. Rows always sum to
-- the parent expense.amount (validated at service layer). share_pct is only
-- populated when split_strategy = 'percentage' — kept so the original ratios
-- survive amount edits.
CREATE TABLE IF NOT EXISTS finance.expense_share (
    expense_id   UUID           NOT NULL REFERENCES finance.expense(id) ON DELETE CASCADE,
    user_id      UUID           NOT NULL,
    amount       NUMERIC(18,4)  NOT NULL,
    amount_base  NUMERIC(18,4)  NOT NULL,
    share_pct    NUMERIC(9,6),
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    PRIMARY KEY (expense_id, user_id),
    CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS idx_finance_expense_share_user_id
    ON finance.expense_share(user_id);

-- fx_snapshot: cached exchange rates keyed on (base, quote, as_of_date). The
-- day-precision as_of lets us reuse one snapshot for every expense on the
-- same calendar day rather than hammering the rate provider. Historical
-- rates are never mutated — a new day = a new row.
CREATE TABLE IF NOT EXISTS finance.fx_snapshot (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    base         CHAR(3)        NOT NULL,
    quote        CHAR(3)        NOT NULL,
    rate         NUMERIC(20,10) NOT NULL,
    as_of        DATE           NOT NULL,
    source       TEXT           NOT NULL DEFAULT 'manual',
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    UNIQUE (base, quote, as_of),
    CHECK (rate > 0)
);

CREATE INDEX IF NOT EXISTS idx_finance_fx_snapshot_lookup
    ON finance.fx_snapshot(base, quote, as_of DESC);

-- budget: one row per (trip, category). Amount is stored in the trip's base
-- currency at set time; if the trip later changes base currency, existing
-- budgets are NOT auto-converted (an explicit edit is required).
CREATE TABLE IF NOT EXISTS finance.budget (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id      UUID           NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    category     TEXT           NOT NULL,
    amount       NUMERIC(18,4)  NOT NULL,
    currency     CHAR(3)        NOT NULL,
    created_by   UUID           NOT NULL,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    UNIQUE (trip_id, category),
    CHECK (amount > 0)
);

-- settlement: a proposed or confirmed transfer between two members. Generated
-- by the min-cashflow reducer, then flipped 'proposed' → 'confirmed' by the
-- payee once they've actually received the money.
--   status — 'proposed' | 'confirmed' | 'cancelled'
--   confirmed_at populated only when status flips to 'confirmed'.
CREATE TABLE IF NOT EXISTS finance.settlement (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id       UUID           NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    payer_id      UUID           NOT NULL,
    payee_id      UUID           NOT NULL,
    amount        NUMERIC(18,4)  NOT NULL,
    currency      CHAR(3)        NOT NULL,
    status        TEXT           NOT NULL DEFAULT 'proposed',
    note          TEXT           NOT NULL DEFAULT '',
    created_by    UUID           NOT NULL,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT now(),
    confirmed_at  TIMESTAMPTZ,
    cancelled_at  TIMESTAMPTZ,
    CHECK (payer_id <> payee_id),
    CHECK (amount > 0),
    CHECK (status IN ('proposed', 'confirmed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_finance_settlement_trip
    ON finance.settlement(trip_id, status);
CREATE INDEX IF NOT EXISTS idx_finance_settlement_payer
    ON finance.settlement(payer_id, status);
CREATE INDEX IF NOT EXISTS idx_finance_settlement_payee
    ON finance.settlement(payee_id, status);

-- +goose Down
DROP TABLE IF EXISTS finance.settlement;
DROP TABLE IF EXISTS finance.budget;
DROP TABLE IF EXISTS finance.fx_snapshot;
DROP TABLE IF EXISTS finance.expense_share;
DROP TABLE IF EXISTS finance.expense;
DROP SCHEMA IF EXISTS finance;
ALTER TABLE trip.trip DROP COLUMN IF EXISTS base_currency;
