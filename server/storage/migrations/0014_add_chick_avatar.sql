-- Migration 13 reached production before chick.webp was added to the second
-- collection. Extend the persisted constraint in a new migration so databases
-- that already recorded version 13 receive the missing key.
CREATE TEMP TABLE avatar_key_backup (
  device_id TEXT PRIMARY KEY,
  avatar_key TEXT NOT NULL
);

INSERT INTO avatar_key_backup(device_id, avatar_key)
SELECT id, avatar_key FROM device;

ALTER TABLE device DROP COLUMN avatar_key;

ALTER TABLE device ADD COLUMN avatar_key TEXT NOT NULL DEFAULT 'cat'
  CHECK (avatar_key IN (
    'cat', 'dog', 'fox', 'rabbit', 'panda', 'owl', 'penguin', 'capybara',
    'chick', 'lion', 'sheep', 'tiger',
    'cat_bow', 'rabbit_flower', 'panda_flower', 'fox_bow'
  ));

UPDATE device
SET avatar_key = (
  SELECT avatar_key_backup.avatar_key
  FROM avatar_key_backup
  WHERE avatar_key_backup.device_id = device.id
);

DROP TABLE avatar_key_backup;

INSERT INTO schema_migrations(version) VALUES (14);
PRAGMA user_version = 14;
