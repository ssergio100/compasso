CREATE TABLE pending_control_effect (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  command_id TEXT NOT NULL UNIQUE,
  revision INTEGER NOT NULL CHECK (revision > 0),
  kind TEXT NOT NULL CHECK (kind IN (
    'pause_monitoring', 'resume_monitoring', 'block_now', 'clear_manual_block'
  )),
  received_at TEXT NOT NULL
);

INSERT INTO schema_migrations(version) VALUES (5);
PRAGMA user_version = 5;
