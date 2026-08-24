-- Some pilot installations recorded migration 8 before the activity-history
-- schema was present. Keep this migration idempotent so it repairs those
-- databases and remains harmless where migration 8 completed normally.
CREATE TABLE IF NOT EXISTS activity (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES device(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  origin TEXT NOT NULL CHECK (origin IN ('admin', 'device', 'server')),
  status TEXT NOT NULL CHECK (status IN ('waiting_device', 'offered', 'completed', 'attention', 'failed')),
  details_json TEXT NOT NULL DEFAULT '{}',
  occurred_at TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  completed_at TEXT,
  expires_at TEXT,
  hidden_at TEXT
);

CREATE INDEX IF NOT EXISTS activity_device_occurred_idx
  ON activity(device_id, occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS activity_expiration_idx
  ON activity(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS activity_step (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  activity_id TEXT NOT NULL REFERENCES activity(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  actor TEXT NOT NULL CHECK (actor IN ('admin', 'device', 'server')),
  occurred_at TEXT NOT NULL,
  last_occurred_at TEXT NOT NULL,
  occurrences INTEGER NOT NULL DEFAULT 1 CHECK (occurrences > 0),
  details_json TEXT NOT NULL DEFAULT '{}',
  UNIQUE(activity_id, kind)
);

CREATE INDEX IF NOT EXISTS activity_step_activity_idx
  ON activity_step(activity_id, occurred_at, id);

CREATE TABLE IF NOT EXISTS activity_history_maintenance (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  last_cleanup_at TEXT
);

INSERT OR IGNORE INTO activity_history_maintenance(singleton_id) VALUES (1);

-- Reconstruct the human-facing history from the durable business records.
INSERT OR IGNORE INTO activity(
  id, device_id, kind, origin, status, details_json,
  occurred_at, observed_at, completed_at, expires_at
)
SELECT id, device_id, kind, 'admin',
       CASE
         WHEN acknowledged_at IS NOT NULL THEN 'completed'
         WHEN delivery_attempts > 0 THEN 'offered'
         ELSE 'waiting_device'
       END,
       CASE WHEN kind='add_bonus'
            THEN json_object('minutes', CAST(json_extract(payload_json, '$.seconds') / 60 AS TEXT))
            ELSE json_object('command', kind) END,
       created_at, created_at, acknowledged_at,
       CASE WHEN acknowledged_at IS NOT NULL
            THEN strftime('%Y-%m-%dT%H:%M:%fZ', acknowledged_at, '+30 days') END
FROM device_command;

INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT id, 'requested', 'admin', created_at, created_at FROM device_command;
INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT id, 'stored', 'server', created_at, created_at FROM device_command;
INSERT OR IGNORE INTO activity_step(
  activity_id, kind, actor, occurred_at, last_occurred_at, occurrences
)
SELECT id, 'offered', 'server', first_offered_at, last_offered_at, delivery_attempts
FROM device_command
WHERE delivery_attempts > 0 AND first_offered_at IS NOT NULL;
INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT id, 'completed', 'device', acknowledged_at, acknowledged_at
FROM device_command WHERE acknowledged_at IS NOT NULL;

INSERT OR IGNORE INTO activity(
  id, device_id, kind, origin, status, details_json,
  occurred_at, observed_at, completed_at, expires_at
)
SELECT uuid, device_id, 'add_bonus', 'device', 'completed',
       json_object(
         'minutes', CAST(json_extract(payload_json, '$.seconds') / 60 AS TEXT),
         'local_date', json_extract(payload_json, '$.local_date')
       ),
       created_at, created_at, created_at,
       strftime('%Y-%m-%dT%H:%M:%fZ', created_at, '+30 days')
FROM audit_event
WHERE device_id IS NOT NULL AND kind='bonus_added' AND origin='local';

INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT uuid, 'local_created', 'device', created_at, created_at
FROM audit_event
WHERE device_id IS NOT NULL AND kind='bonus_added' AND origin='local';
INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT uuid, 'synchronized', 'server', created_at, created_at
FROM audit_event
WHERE device_id IS NOT NULL AND kind='bonus_added' AND origin='local';
INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT uuid, 'confirmed', 'server', created_at, created_at
FROM audit_event
WHERE device_id IS NOT NULL AND kind='bonus_added' AND origin='local';

INSERT INTO schema_migrations(version) VALUES (9);
PRAGMA user_version = 9;
