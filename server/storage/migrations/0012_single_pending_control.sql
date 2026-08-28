-- Remote control expresses the latest desired state; it is not an ordered job
-- queue. Keep only the newest unacknowledged control operation per device.
DELETE FROM activity
WHERE id IN (
  SELECT older.id
  FROM device_command AS older
  WHERE older.acknowledged_at IS NULL
    AND older.kind IN (
      'pause_monitoring', 'resume_monitoring', 'block_now', 'clear_manual_block'
    )
    AND EXISTS (
      SELECT 1
      FROM device_command AS newer
      WHERE newer.device_id = older.device_id
        AND newer.acknowledged_at IS NULL
        AND newer.kind IN (
          'pause_monitoring', 'resume_monitoring', 'block_now', 'clear_manual_block'
        )
        AND (
          newer.created_at > older.created_at OR
          (newer.created_at = older.created_at AND newer.id > older.id)
        )
    )
);

-- Completed control envelopes are transport bookkeeping. The retained
-- activity is the bounded human history; additive bonus commands are kept.
DELETE FROM device_command
WHERE kind IN (
  'pause_monitoring', 'resume_monitoring', 'block_now', 'clear_manual_block'
)
AND (
  acknowledged_at IS NOT NULL OR
  EXISTS (
    SELECT 1
    FROM device_command AS newer
    WHERE newer.device_id = device_command.device_id
      AND newer.acknowledged_at IS NULL
      AND newer.kind IN (
        'pause_monitoring', 'resume_monitoring', 'block_now', 'clear_manual_block'
      )
      AND (
        newer.created_at > device_command.created_at OR
        (newer.created_at = device_command.created_at AND newer.id > device_command.id)
      )
  )
);

-- Dedicated activities already represent control operations. Remove the
-- redundant, unbounded technical audit copies created by older versions.
DELETE FROM audit_event
WHERE origin = 'web'
  AND kind IN (
    'pause_monitoring', 'resume_monitoring', 'block_now', 'clear_manual_block'
  );

-- Pausing means stopping protection and therefore also clears a latent manual
-- block. Increment the revision so enrolled agents observe the normalization.
UPDATE device_control
SET manual_block = 0, revision = revision + 1
WHERE monitoring_paused = 1 AND manual_block = 1;

CREATE UNIQUE INDEX device_command_one_pending_control_idx
ON device_command(device_id)
WHERE acknowledged_at IS NULL
  AND kind IN (
    'pause_monitoring', 'resume_monitoring', 'block_now', 'clear_manual_block'
  );

CREATE TRIGGER device_control_exclusive_insert
BEFORE INSERT ON device_control
WHEN NEW.monitoring_paused = 1 AND NEW.manual_block = 1
BEGIN
  SELECT RAISE(ABORT, 'device control cannot be paused and blocked');
END;

CREATE TRIGGER device_control_exclusive_update
BEFORE UPDATE ON device_control
WHEN NEW.monitoring_paused = 1 AND NEW.manual_block = 1
BEGIN
  SELECT RAISE(ABORT, 'device control cannot be paused and blocked');
END;

INSERT INTO schema_migrations(version) VALUES (12);
PRAGMA user_version = 12;
