#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
pg_bin="$(pg_config --bindir)"
workdir="$(mktemp -d)"
api_pid=
cleanup() {
  if [ -n "$api_pid" ]; then
    kill -TERM "$api_pid" 2>/dev/null || true
    wait "$api_pid" || true
  fi
  "$pg_bin/pg_ctl" -D "$workdir/db" -m fast stop >/dev/null 2>&1 || true
  rm -rf -- "$workdir"
}
trap cleanup EXIT
trap 'exit 1' INT TERM

go build -buildvcs=false -o "$workdir/api" ./cmd/rydo-api
"$pg_bin/initdb" -D "$workdir/db" -A trust --no-locale >/dev/null
"$pg_bin/pg_ctl" -D "$workdir/db" -l "$workdir/postgres.log" \
  -o "-c listen_addresses= -k $workdir" -w start >/dev/null
db_url="postgres:///postgres?host=$workdir&sslmode=disable"
api_port="${RYDO_TEST_PORT:-18421}"
JWT_SIGNING_KEY="foundation-test-access-key-01234567890123456789" \
PREVIEW_SIGNING_KEY="foundation-test-preview-key-01234567890123456789" \
LINE_LOGIN_CHANNEL_ID="foundation-test-channel" WEB_APP_URL="http://127.0.0.1:$api_port" \
DATABASE_URL="$db_url" PORT="$api_port" APP_PORT=0 "$workdir/api" &
api_pid=$!
curl --fail --silent --show-error --retry 5 --retry-connrefused --retry-delay 1 \
  "http://127.0.0.1:$api_port/health/live" >/dev/null

assert_equal() {
  if [ "$1" != "$2" ]; then
    printf '%s: expected %s, got %s\n' "$3" "$2" "$1" >&2
    failures=1
  fi
}
readiness() {
  curl --silent --show-error -o /dev/null -w '%{http_code}' "http://127.0.0.1:$api_port/health/ready"
}
query() {
  psql -XAt -v ON_ERROR_STOP=1 "$db_url" -c "$1"
}
migration() {
  migrate -path db/migrations -database "$db_url" "$@"
}

failures=0
assert_equal "$(curl --silent --show-error -o /dev/null -w '%{http_code}' "http://127.0.0.1:$api_port/api/v1/me")" 401 'Pilot routes mounted in executable'
assert_equal "$(readiness)" 503 'Empty database readiness'
migration up 1
assert_equal "$(readiness)" 503 'Outdated database readiness'
migration up 1
assert_equal "$(readiness)" 503 'Readiness before numeric validation migration'
query "WITH rider AS (
  INSERT INTO users (line_user_id) VALUES ('integration-test') RETURNING id
), vehicle AS (
  INSERT INTO vehicles (user_id, name, km_per_liter, fuel_price_per_liter)
  SELECT id, 'Test vehicle', 40, 35 FROM rider RETURNING id, user_id
)
INSERT INTO daily_sessions (
  user_id, vehicle_id, work_date, distance_km, working_hours,
  snapshot_km_per_liter, snapshot_fuel_price, snapshot_maintenance_per_km
)
SELECT user_id, id, '2026-01-01', 100, 8, 40, 35, 0 FROM vehicle;
INSERT INTO income_entries (session_id, platform_id, gross_income)
SELECT daily_sessions.id, platforms.id, 850 FROM daily_sessions
CROSS JOIN platforms WHERE platforms.code = 'grab';
INSERT INTO expense_entries (session_id, category, amount)
SELECT id, 'parking', 30 FROM daily_sessions" >/dev/null
migration up
assert_equal "$(readiness)" 200 'Migrated database readiness'
assert_equal "$(query 'SELECT distance_km FROM daily_sessions')" 100.00 'Existing workday survives migration'
for field in vehicles.km_per_liter vehicles.fuel_price_per_liter vehicles.maintenance_reserve_per_km \
  daily_sessions.distance_km daily_sessions.snapshot_km_per_liter daily_sessions.snapshot_fuel_price \
  daily_sessions.snapshot_maintenance_per_km income_entries.gross_income expense_entries.amount; do
  table=${field%.*}
  column=${field#*.}
  if ! query "DO \$\$
    BEGIN
      UPDATE $table SET $column = 'NaN';
      RAISE EXCEPTION 'Expected a numeric check violation for $field';
    EXCEPTION WHEN check_violation THEN
      NULL;
    END \$\$;" >/dev/null; then
    failures=1
  fi
done
assert_equal "$(query "SELECT EXISTS (
  SELECT FROM pg_constraint
  WHERE conrelid = 'daily_sessions'::regclass AND confrelid = 'vehicles'::regclass
    AND contype = 'f' AND pg_get_constraintdef(oid) = 'FOREIGN KEY (vehicle_id, user_id) REFERENCES vehicles(id, user_id)'
)")" t 'Database ownership constraint'
query 'UPDATE schema_migrations SET dirty = true' >/dev/null
assert_equal "$(readiness)" 503 'Dirty database readiness'
query 'UPDATE schema_migrations SET dirty = false' >/dev/null
migration down 1
assert_equal "$(readiness)" 503 'Readiness after numeric validation migration rollback'
migration up 1
assert_equal "$(readiness)" 200 'Readiness after numeric validation migration reapplies'
migration down -all
assert_equal "$(readiness)" 503 'Rolled-back database readiness'
assert_equal "$(query "SELECT EXISTS (SELECT FROM pg_extension WHERE extname = 'pgcrypto')")" f 'Unused extension after rollback'
test "$failures" -eq 0
