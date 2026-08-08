PRAGMA foreign_keys = ON;

CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE policy_state (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  revision INTEGER NOT NULL CHECK (revision >= 0),
  monitoring_paused INTEGER NOT NULL DEFAULT 0 CHECK (monitoring_paused IN (0, 1)),
  manual_block INTEGER NOT NULL DEFAULT 0 CHECK (manual_block IN (0, 1)),
  warning_minutes INTEGER NOT NULL DEFAULT 10 CHECK (warning_minutes >= 0),
  local_password_verifier TEXT,
  updated_at TEXT NOT NULL
);

CREATE TABLE weekly_quota (
  weekday INTEGER PRIMARY KEY CHECK (weekday BETWEEN 0 AND 6),
  seconds_allowed INTEGER NOT NULL CHECK (seconds_allowed >= 0)
);

CREATE TABLE routine (
  id TEXT PRIMARY KEY,
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
  local_date TEXT PRIMARY KEY,
  seconds_used INTEGER NOT NULL DEFAULT 0 CHECK (seconds_used >= 0),
  checkpoint_at TEXT NOT NULL
);

CREATE TABLE bonus (
  uuid TEXT PRIMARY KEY,
  local_date TEXT NOT NULL,
  seconds INTEGER NOT NULL CHECK (seconds > 0),
  origin TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE pending_event (
  uuid TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0)
);

INSERT INTO schema_migrations(version) VALUES (1);
PRAGMA user_version = 1;
