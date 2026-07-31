-- +goose Up
-- Trips created before the TWD default flip (migration 024) still carry
-- base_currency='USD' from the original default. Since no trip explicitly
-- opts into USD via the FE (there's no currency picker yet), it's safe to
-- assume USD == "the old default" and rewrite it to TWD in one shot.
UPDATE trip.trip
   SET base_currency = 'TWD'
 WHERE base_currency = 'USD';

-- +goose Down
-- Not reversible: we can't tell which trips were "USD by default" vs
-- "USD on purpose" after the fact. Leave the down migration as a no-op.
SELECT 1;
