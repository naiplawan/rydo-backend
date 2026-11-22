BEGIN;

ALTER TABLE expense_entries DROP CONSTRAINT expense_entries_finite_amount;
ALTER TABLE income_entries DROP CONSTRAINT income_entries_finite_gross_income;
ALTER TABLE daily_sessions DROP CONSTRAINT daily_sessions_finite_distance_and_costs;
ALTER TABLE vehicles DROP CONSTRAINT vehicles_finite_cost_settings;

COMMIT;
