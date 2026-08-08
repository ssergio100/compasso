ALTER TABLE device ADD COLUMN applied_policy_revision INTEGER NOT NULL DEFAULT 0
  CHECK (applied_policy_revision >= 0);

CREATE TABLE device_command (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  acknowledged_at TEXT
);

CREATE INDEX device_command_pending_idx
  ON device_command(device_id, acknowledged_at, created_at);

INSERT INTO schema_migrations(version) VALUES (2);
PRAGMA user_version = 2;
