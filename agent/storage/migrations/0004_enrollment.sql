CREATE TABLE enrollment (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  server_url TEXT NOT NULL,
  device_id TEXT NOT NULL
);

INSERT INTO schema_migrations(version) VALUES (4);
PRAGMA user_version = 4;
