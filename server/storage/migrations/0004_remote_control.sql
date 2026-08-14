CREATE TABLE device_control (
  device_id TEXT PRIMARY KEY REFERENCES device(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1),
  monitoring_paused INTEGER NOT NULL DEFAULT 0 CHECK (monitoring_paused IN (0, 1)),
  manual_block INTEGER NOT NULL DEFAULT 0 CHECK (manual_block IN (0, 1)),
  updated_at TEXT NOT NULL
);

INSERT INTO device_control(device_id, monitoring_paused, manual_block, updated_at)
SELECT device_id, monitoring_paused, manual_block, updated_at FROM policy;

UPDATE policy SET monitoring_paused=0, manual_block=0;

INSERT INTO schema_migrations(version) VALUES (4);
PRAGMA user_version = 4;
