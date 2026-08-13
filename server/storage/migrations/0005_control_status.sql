ALTER TABLE device ADD COLUMN applied_control_revision INTEGER NOT NULL DEFAULT 0
  CHECK (applied_control_revision >= 0);

ALTER TABLE device ADD COLUMN graphical_session_locked INTEGER NOT NULL DEFAULT 0
  CHECK (graphical_session_locked IN (0, 1));

INSERT INTO schema_migrations(version) VALUES (5);
PRAGMA user_version = 5;
