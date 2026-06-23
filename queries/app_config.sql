-- name: GetAppConfig :one
SELECT * FROM app_config
WHERE id = 1;

-- name: InsertDefaultAppConfig :one
INSERT INTO app_config (id) VALUES (1)
ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
RETURNING *;

-- name: UpsertAppConfig :one
INSERT INTO app_config(id, attempt_notify, max_message_length, max_concurrent_syncs, require_totp, debug_logging, log_retention_days, log_cleanup_interval_hours, log_buffer_size, attempt_workflow, cw_bot_member_identifier, root_url, cw_company_id, cw_client_id, cw_public_key, cw_private_key, webex_secret)
VALUES(1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (id) DO UPDATE SET
    attempt_notify = EXCLUDED.attempt_notify,
    max_message_length = EXCLUDED.max_message_length,
    max_concurrent_syncs = EXCLUDED.max_concurrent_syncs,
    require_totp = EXCLUDED.require_totp,
    debug_logging = EXCLUDED.debug_logging,
    log_retention_days = EXCLUDED.log_retention_days,
    log_cleanup_interval_hours = EXCLUDED.log_cleanup_interval_hours,
    log_buffer_size = EXCLUDED.log_buffer_size,
    attempt_workflow = EXCLUDED.attempt_workflow,
    cw_bot_member_identifier = EXCLUDED.cw_bot_member_identifier,
    root_url = EXCLUDED.root_url,
    cw_company_id = EXCLUDED.cw_company_id,
    cw_client_id = EXCLUDED.cw_client_id,
    cw_public_key = EXCLUDED.cw_public_key,
    cw_private_key = EXCLUDED.cw_private_key,
    webex_secret = EXCLUDED.webex_secret
RETURNING *;

