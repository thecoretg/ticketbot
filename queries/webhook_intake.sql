-- name: InsertWebhookIntake :one
INSERT INTO webhook_intake (ticket_id, action, payload)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ClaimWebhookIntake :one
-- Claims the oldest ready row whose ticket has no older open row, so one ticket's webhooks are
-- processed strictly in arrival order and never two at once.
UPDATE webhook_intake SET status = 'processing', attempts = attempts + 1
WHERE id = (
    SELECT w.id FROM webhook_intake w
    WHERE w.status = 'pending'
      AND w.next_attempt_at <= NOW()
      AND NOT EXISTS (
          SELECT 1 FROM webhook_intake o
          WHERE o.ticket_id = w.ticket_id
            AND o.status IN ('pending', 'processing')
            AND o.id < w.id
      )
    ORDER BY w.id
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: FinishWebhookIntake :exec
UPDATE webhook_intake SET status = 'done', last_error = NULL, finished_at = NOW()
WHERE id = $1;

-- name: RescheduleWebhookIntake :exec
UPDATE webhook_intake SET status = 'pending', next_attempt_at = $2, last_error = $3
WHERE id = $1;

-- name: FailWebhookIntake :exec
UPDATE webhook_intake SET status = 'failed', last_error = $2, finished_at = NOW()
WHERE id = $1;

-- name: RetryWebhookIntake :one
UPDATE webhook_intake
SET status = 'pending', attempts = 0, next_attempt_at = NOW(), last_error = NULL, finished_at = NULL
WHERE id = $1 AND status = 'failed'
RETURNING *;

-- name: DiscardWebhookIntake :one
UPDATE webhook_intake SET status = 'discarded', finished_at = NOW()
WHERE id = $1 AND status = 'failed'
RETURNING *;

-- name: ResetProcessingWebhookIntake :execrows
-- Rows left in processing by a crash or restart go back to pending.
UPDATE webhook_intake SET status = 'pending', next_attempt_at = NOW()
WHERE status = 'processing';

-- name: ListWebhookIntake :many
SELECT * FROM webhook_intake
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY id DESC
LIMIT sqlc.arg('lim');

-- name: CountWebhookIntakeByStatus :many
SELECT status, COUNT(*)::bigint AS n FROM webhook_intake GROUP BY status;

-- name: CountWebhookIntakeReceivedSince :one
SELECT COUNT(*)::bigint FROM webhook_intake WHERE received_at >= $1;

-- name: LastWebhookIntakeReceivedAt :one
SELECT received_at FROM webhook_intake ORDER BY received_at DESC LIMIT 1;

-- name: DeleteFinishedWebhookIntakeBefore :execrows
DELETE FROM webhook_intake
WHERE status IN ('done', 'discarded') AND finished_at < $1;

-- name: CountWebhookIntakeByHour :many
SELECT date_trunc('hour', received_at)::timestamptz AS hour, COUNT(*)::bigint AS n
FROM webhook_intake
WHERE received_at >= $1
GROUP BY 1
ORDER BY 1;
