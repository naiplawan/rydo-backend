BEGIN;

ALTER TABLE vehicles ADD CONSTRAINT vehicles_finite_cost_settings CHECK (
  km_per_liter <> 'NaN'::numeric
  AND fuel_price_per_liter <> 'NaN'::numeric
  AND maintenance_reserve_per_km <> 'NaN'::numeric
);

ALTER TABLE daily_sessions ADD CONSTRAINT daily_sessions_finite_distance_and_costs CHECK (
  distance_km <> 'NaN'::numeric
  AND snapshot_km_per_liter <> 'NaN'::numeric
  AND snapshot_fuel_price <> 'NaN'::numeric
  AND snapshot_maintenance_per_km <> 'NaN'::numeric
);

ALTER TABLE income_entries ADD CONSTRAINT income_entries_finite_gross_income
  CHECK (gross_income <> 'NaN'::numeric);

ALTER TABLE expense_entries ADD CONSTRAINT expense_entries_finite_amount
  CHECK (amount <> 'NaN'::numeric);

COMMIT;
