ALTER TABLE device_command ADD COLUMN delivery_attempts INTEGER NOT NULL DEFAULT 0
  CHECK (delivery_attempts >= 0);

ALTER TABLE device_command ADD COLUMN first_offered_at TEXT;
ALTER TABLE device_command ADD COLUMN last_offered_at TEXT;

-- Historical acknowledgements prove at least one successful delivery even
-- though older schemas did not count individual offers.
UPDATE device_command SET
  delivery_attempts = 1,
  first_offered_at = created_at,
  last_offered_at = acknowledged_at
WHERE acknowledged_at IS NOT NULL;

INSERT INTO schema_migrations(version) VALUES (7);
PRAGMA user_version = 7;
