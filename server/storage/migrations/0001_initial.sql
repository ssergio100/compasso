PRAGMA foreign_keys = ON;

CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE admin_user (
  id TEXT PRIMARY KEY,
  login TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE device (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  device_token_hash TEXT NOT NULL,
  last_seen_at TEXT,
  policy_revision INTEGER NOT NULL DEFAULT 0 CHECK (policy_revision >= 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE policy (
  device_id TEXT PRIMARY KEY REFERENCES device(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL CHECK (revision >= 0),
  monitoring_paused INTEGER NOT NULL DEFAULT 0 CHECK (monitoring_paused IN (0, 1)),
  manual_block INTEGER NOT NULL DEFAULT 0 CHECK (manual_block IN (0, 1)),
  warning_minutes INTEGER NOT NULL DEFAULT 10 CHECK (warning_minutes >= 0),
  local_password_verifier TEXT,
  updated_at TEXT NOT NULL
);

CREATE TABLE weekly_quota (
  device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
  weekday INTEGER NOT NULL CHECK (weekday BETWEEN 0 AND 6),
  seconds_allowed INTEGER NOT NULL CHECK (seconds_allowed >= 0),
  PRIMARY KEY (device_id, weekday)
);

CREATE TABLE routine (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  start_second INTEGER NOT NULL CHECK (start_second BETWEEN 0 AND 86399),
  end_second INTEGER NOT NULL CHECK (end_second BETWEEN 0 AND 86399),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1))
);

CREATE TABLE routine_day (
  routine_id TEXT NOT NULL REFERENCES routine(id) ON DELETE CASCADE,
  weekday INTEGER NOT NULL CHECK (weekday BETWEEN 0 AND 6),
  PRIMARY KEY (routine_id, weekday)
);

CREATE TABLE daily_usage (
  device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
  local_date TEXT NOT NULL,
  seconds_used INTEGER NOT NULL DEFAULT 0 CHECK (seconds_used >= 0),
  last_sync_at TEXT,
  PRIMARY KEY (device_id, local_date)
);

CREATE TABLE bonus (
  uuid TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
  local_date TEXT NOT NULL,
  seconds INTEGER NOT NULL CHECK (seconds > 0),
  origin TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE audit_event (
  uuid TEXT PRIMARY KEY,
  device_id TEXT REFERENCES device(id) ON DELETE SET NULL,
  kind TEXT NOT NULL,
  origin TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX audit_event_device_created_idx
  ON audit_event(device_id, created_at);

INSERT INTO schema_migrations(version) VALUES (1);
PRAGMA user_version = 1;
