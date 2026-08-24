-- Project every existing human-facing administrative mutation that predates
-- the centralized activity projection. Technical polling and command audit
-- records remain outside this list because commands already have their own
-- end-to-end activity identifiers.
INSERT OR IGNORE INTO activity(
  id, device_id, kind, origin, status, details_json,
  occurred_at, observed_at, completed_at, expires_at
)
SELECT uuid, device_id, kind, 'admin', 'completed',
       CASE kind
         WHEN 'device_created' THEN json_object(
           'name', COALESCE(CAST(json_extract(payload_json, '$.name') AS TEXT), '')
         )
         WHEN 'device_renamed' THEN json_object(
           'name', COALESCE(CAST(json_extract(payload_json, '$.name') AS TEXT), '')
         )
         WHEN 'quotas_updated' THEN json_object(
           'warning_minutes', COALESCE(CAST(json_extract(payload_json, '$.warning_minutes') AS TEXT), '')
         )
         WHEN 'routine_saved' THEN json_object(
           'routine_id', COALESCE(CAST(json_extract(payload_json, '$.routine_id') AS TEXT), ''),
           'name', COALESCE(CAST(json_extract(payload_json, '$.name') AS TEXT), ''),
           'action', COALESCE(CAST(json_extract(payload_json, '$.action') AS TEXT), 'saved')
         )
         WHEN 'routine_deleted' THEN json_object(
           'routine_id', COALESCE(CAST(json_extract(payload_json, '$.routine_id') AS TEXT), ''),
           'name', COALESCE(CAST(json_extract(payload_json, '$.name') AS TEXT), '')
         )
         ELSE '{}'
       END,
       created_at, created_at, created_at,
       strftime('%Y-%m-%dT%H:%M:%fZ', created_at, '+30 days')
FROM audit_event
WHERE device_id IS NOT NULL
  AND origin='web'
  AND kind IN (
    'device_created', 'device_renamed', 'quotas_updated', 'routine_saved',
    'routine_deleted', 'local_password_changed', 'device_token_issued',
    'device_token_revoked'
  );

INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT uuid, 'requested', 'admin', created_at, created_at
FROM audit_event
WHERE device_id IS NOT NULL
  AND origin='web'
  AND kind IN (
    'device_created', 'device_renamed', 'quotas_updated', 'routine_saved',
    'routine_deleted', 'local_password_changed', 'device_token_issued',
    'device_token_revoked'
  );

INSERT OR IGNORE INTO activity_step(activity_id, kind, actor, occurred_at, last_occurred_at)
SELECT uuid, 'completed', 'server', created_at, created_at
FROM audit_event
WHERE device_id IS NOT NULL
  AND origin='web'
  AND kind IN (
    'device_created', 'device_renamed', 'quotas_updated', 'routine_saved',
    'routine_deleted', 'local_password_changed', 'device_token_issued',
    'device_token_revoked'
  );

INSERT INTO schema_migrations(version) VALUES (10);
PRAGMA user_version = 10;
