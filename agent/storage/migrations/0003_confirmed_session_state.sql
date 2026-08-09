CREATE TABLE confirmed_session_state (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  revision INTEGER NOT NULL CHECK (revision >= 0),
  session_id TEXT NOT NULL,
  local_date TEXT NOT NULL,
  remaining_seconds INTEGER NOT NULL CHECK (remaining_seconds >= 0),
  usage_seconds INTEGER NOT NULL CHECK (usage_seconds >= 0),
  confirmed_at TEXT NOT NULL
);

INSERT INTO schema_migrations(version) VALUES (3);
PRAGMA user_version = 3;
