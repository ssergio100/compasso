ALTER TABLE device ADD COLUMN graphical_session_active INTEGER NOT NULL DEFAULT 0
  CHECK (graphical_session_active IN (0, 1));

ALTER TABLE device ADD COLUMN graphical_session_id TEXT;

INSERT INTO schema_migrations(version) VALUES (3);
PRAGMA user_version = 3;
