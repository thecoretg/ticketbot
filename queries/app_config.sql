-- name: GetAppConfig :one
SELECT * FROM app_config
WHERE id = 1;

-- name: InsertDefaultAppConfig :one
INSERT INTO app_config (id) VALUES (1)
ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
RETURNING *;

-- name: UpsertAppConfig :one
INSERT INTO app_config(id, master_dry_run, cw_api_member_identifier, max_message_length, max_concurrent_syncs, require_totp, debug_logging, log_retention_days, log_cleanup_interval_hours, log_buffer_size, sso_enabled, password_login_enabled, note_preview_length, write_cap_per_ticket, ops_room_id, redirect_room_id, stale_alert_minutes, history_retention_days, business_open, business_close, business_days, business_zone)
VALUES(1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
ON CONFLICT (id) DO UPDATE SET
    master_dry_run = EXCLUDED.master_dry_run,
    cw_api_member_identifier = EXCLUDED.cw_api_member_identifier,
    max_message_length = EXCLUDED.max_message_length,
    max_concurrent_syncs = EXCLUDED.max_concurrent_syncs,
    require_totp = EXCLUDED.require_totp,
    debug_logging = EXCLUDED.debug_logging,
    log_retention_days = EXCLUDED.log_retention_days,
    log_cleanup_interval_hours = EXCLUDED.log_cleanup_interval_hours,
    log_buffer_size = EXCLUDED.log_buffer_size,
    sso_enabled = EXCLUDED.sso_enabled,
    password_login_enabled = EXCLUDED.password_login_enabled,
    note_preview_length = EXCLUDED.note_preview_length,
    write_cap_per_ticket = EXCLUDED.write_cap_per_ticket,
    ops_room_id = EXCLUDED.ops_room_id,
    redirect_room_id = EXCLUDED.redirect_room_id,
    stale_alert_minutes = EXCLUDED.stale_alert_minutes,
    history_retention_days = EXCLUDED.history_retention_days,
    business_open = EXCLUDED.business_open,
    business_close = EXCLUDED.business_close,
    business_days = EXCLUDED.business_days,
    business_zone = EXCLUDED.business_zone
RETURNING *;
