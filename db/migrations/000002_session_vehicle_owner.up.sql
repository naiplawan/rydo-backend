BEGIN;

ALTER TABLE vehicles
  ADD CONSTRAINT vehicles_id_user_id_key UNIQUE (id, user_id);

ALTER TABLE daily_sessions
  DROP CONSTRAINT daily_sessions_vehicle_id_fkey,
  ADD CONSTRAINT daily_sessions_vehicle_owner_fkey
    FOREIGN KEY (vehicle_id, user_id) REFERENCES vehicles(id, user_id);

COMMIT;
