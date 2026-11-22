CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  line_user_id VARCHAR(128) NOT NULL UNIQUE,
  display_name VARCHAR(120),
  timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Bangkok',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE platforms (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code VARCHAR(32) NOT NULL UNIQUE,
  name VARCHAR(80) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE auth_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  refresh_token_hash BYTEA NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ
);

CREATE TABLE user_platforms (
  user_id UUID NOT NULL REFERENCES users(id),
  platform_id UUID NOT NULL REFERENCES platforms(id),
  PRIMARY KEY (user_id, platform_id)
);

INSERT INTO platforms (code, name) VALUES
  ('grab', 'Grab'),
  ('lineman', 'LINE MAN'),
  ('shopeefood', 'ShopeeFood'),
  ('other', 'Other')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE vehicles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  name VARCHAR(100) NOT NULL,
  km_per_liter NUMERIC(8,3) NOT NULL CHECK (km_per_liter > 0),
  fuel_price_per_liter NUMERIC(8,2) NOT NULL CHECK (fuel_price_per_liter >= 0),
  maintenance_reserve_per_km NUMERIC(8,3) NOT NULL DEFAULT 0
    CHECK (maintenance_reserve_per_km >= 0),
  is_default BOOLEAN NOT NULL DEFAULT true CHECK (is_default),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX vehicles_one_active_per_user
ON vehicles(user_id) WHERE deleted_at IS NULL;

CREATE TABLE daily_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  vehicle_id UUID NOT NULL REFERENCES vehicles(id),
  work_date DATE NOT NULL,
  distance_km NUMERIC(10,2) NOT NULL CHECK (distance_km >= 0),
  working_hours NUMERIC(8,2) NOT NULL CHECK (working_hours BETWEEN 0 AND 24),
  snapshot_km_per_liter NUMERIC(8,3) NOT NULL CHECK (snapshot_km_per_liter > 0),
  snapshot_fuel_price NUMERIC(8,2) NOT NULL CHECK (snapshot_fuel_price >= 0),
  snapshot_maintenance_per_km NUMERIC(8,3) NOT NULL
    CHECK (snapshot_maintenance_per_km >= 0),
  version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX daily_sessions_one_active_per_date
ON daily_sessions(user_id, work_date DESC)
WHERE deleted_at IS NULL;

CREATE TABLE session_requests (
  user_id UUID NOT NULL REFERENCES users(id),
  request_key UUID NOT NULL,
  request_fingerprint BYTEA NOT NULL,
  response_status INTEGER NOT NULL CHECK (response_status IN (200, 201)),
  response_body JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (user_id, request_key)
);

CREATE INDEX session_requests_expiry_idx ON session_requests(expires_at);

CREATE TABLE income_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES daily_sessions(id),
  platform_id UUID NOT NULL REFERENCES platforms(id),
  gross_income NUMERIC(12,2) NOT NULL CHECK (gross_income >= 0),
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX income_entries_one_active_per_platform
ON income_entries(session_id, platform_id) WHERE deleted_at IS NULL;

CREATE TABLE expense_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES daily_sessions(id),
  category VARCHAR(32) NOT NULL,
  amount NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CHECK (category IN ('parking', 'toll', 'phone', 'other'))
);

CREATE INDEX expense_entries_session_idx
ON expense_entries(session_id) WHERE deleted_at IS NULL;
