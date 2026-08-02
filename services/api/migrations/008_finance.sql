-- +goose Up
CREATE SCHEMA IF NOT EXISTS finance;

-- amount_base + rate_to_base are denormalized at write time so reports don't
-- need per-row FX lookups; rate_to_base is NULL when currency == base_currency.
CREATE TABLE IF NOT EXISTS finance.expense (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id          UUID           NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    paid_by          UUID           NOT NULL,
    amount           NUMERIC(18,4)  NOT NULL,
    currency         CHAR(3)        NOT NULL,
    amount_base      NUMERIC(18,4)  NOT NULL,
    rate_to_base     NUMERIC(20,10),
    description      TEXT           NOT NULL DEFAULT '',
    category         TEXT           NOT NULL DEFAULT 'other',
    split_strategy   TEXT           NOT NULL DEFAULT 'equal',
    occurred_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    receipt_asset_id UUID,
    created_by       UUID           NOT NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ,
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

-- share_pct is only populated for 'percentage' splits so original ratios
-- survive amount edits. paid_at NULL = not paid, timestamp = when it was.
CREATE TABLE IF NOT EXISTS finance.expense_share (
    expense_id  UUID           NOT NULL REFERENCES finance.expense(id) ON DELETE CASCADE,
    user_id     UUID           NOT NULL,
    amount      NUMERIC(18,4)  NOT NULL,
    amount_base NUMERIC(18,4)  NOT NULL,
    share_pct   NUMERIC(9,6),
    paid_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    PRIMARY KEY (expense_id, user_id),
    CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS idx_finance_expense_share_user_id
    ON finance.expense_share(user_id);

-- Day-precision as_of lets one snapshot serve every expense on that calendar
-- day; historical rates are never mutated (new day = new row).
CREATE TABLE IF NOT EXISTS finance.fx_snapshot (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    base       CHAR(3)        NOT NULL,
    quote      CHAR(3)        NOT NULL,
    rate       NUMERIC(20,10) NOT NULL,
    as_of      DATE           NOT NULL,
    source     TEXT           NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT now(),
    UNIQUE (base, quote, as_of),
    CHECK (rate > 0)
);

CREATE INDEX IF NOT EXISTS idx_finance_fx_snapshot_lookup
    ON finance.fx_snapshot(base, quote, as_of DESC);

CREATE TABLE IF NOT EXISTS finance.budget (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id    UUID           NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    category   TEXT           NOT NULL,
    amount     NUMERIC(18,4)  NOT NULL,
    currency   CHAR(3)        NOT NULL,
    created_by UUID           NOT NULL,
    created_at TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT now(),
    UNIQUE (trip_id, category),
    CHECK (amount > 0)
);

CREATE TABLE IF NOT EXISTS finance.settlement (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id      UUID           NOT NULL REFERENCES trip.trip(id) ON DELETE CASCADE,
    payer_id     UUID           NOT NULL,
    payee_id     UUID           NOT NULL,
    amount       NUMERIC(18,4)  NOT NULL,
    currency     CHAR(3)        NOT NULL,
    status       TEXT           NOT NULL DEFAULT 'proposed',
    note         TEXT           NOT NULL DEFAULT '',
    created_by   UUID           NOT NULL,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    confirmed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
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
DROP SCHEMA IF EXISTS finance CASCADE;
