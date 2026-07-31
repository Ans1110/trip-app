-- +goose Up
-- Switch the trip base currency default from USD to TWD. Existing rows are
-- left as-is so live trips keep whatever they were created with; only new
-- inserts that omit base_currency will now land on TWD.
ALTER TABLE trip.trip
    ALTER COLUMN base_currency SET DEFAULT 'TWD';

-- +goose Down
ALTER TABLE trip.trip
    ALTER COLUMN base_currency SET DEFAULT 'USD';
