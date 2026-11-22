BEGIN;

ALTER TABLE daily_sessions
  DROP CONSTRAINT daily_sessions_vehicle_owner_fkey,
  ADD CONSTRAINT daily_sessions_vehicle_id_fkey
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id);

ALTER TABLE vehicles DROP CONSTRAINT vehicles_id_user_id_key;

COMMIT;
