CREATE TABLE applied_command (
  id TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);

INSERT INTO schema_migrations(version) VALUES (2);
PRAGMA user_version = 2;
