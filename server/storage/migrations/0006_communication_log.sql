CREATE TABLE communication_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
  source TEXT NOT NULL CHECK (source IN ('agent', 'api', 'interface')),
  target TEXT NOT NULL CHECK (target IN ('agent', 'api', 'interface')),
  operation TEXT NOT NULL,
  result TEXT NOT NULL CHECK (result IN ('success', 'warning', 'error')),
  http_status INTEGER CHECK (http_status BETWEEN 100 AND 599),
  duration_ms INTEGER CHECK (duration_ms >= 0),
  summary TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL
);

CREATE INDEX communication_log_device_id_idx
  ON communication_log(device_id, id DESC);

CREATE INDEX communication_log_created_idx
  ON communication_log(created_at);

CREATE TABLE communication_log_setting (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  retention_days INTEGER NOT NULL DEFAULT 30 CHECK (retention_days BETWEEN 1 AND 365),
  last_cleanup_at TEXT,
  updated_at TEXT NOT NULL
);

INSERT INTO communication_log_setting(singleton_id, retention_days, updated_at)
VALUES (1, 30, CURRENT_TIMESTAMP);

INSERT INTO schema_migrations(version) VALUES (6);
PRAGMA user_version = 6;
