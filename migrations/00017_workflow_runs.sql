-- +goose Up
-- +goose StatementBegin
CREATE TABLE workflow_run (
    run_id TEXT PRIMARY KEY,
    ticket_id INT NOT NULL REFERENCES cw_ticket(id) ON DELETE CASCADE,
    board_id INT NOT NULL,
    workflow_id INT,
    workflow_name TEXT NOT NULL DEFAULT '',
    event TEXT NOT NULL,
    source TEXT NOT NULL,
    dry_run BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ NOT NULL,
    duration_ms INT NOT NULL DEFAULT 0,
    steps INT NOT NULL DEFAULT 0,
    actions INT NOT NULL DEFAULT 0,
    writes INT NOT NULL DEFAULT 0,
    notif_sent INT NOT NULL DEFAULT 0,
    notif_would_send INT NOT NULL DEFAULT 0,
    notif_none INT NOT NULL DEFAULT 0,
    errors INT NOT NULL DEFAULT 0,
    outcome TEXT NOT NULL
);
CREATE INDEX idx_workflow_run_started ON workflow_run (started_at DESC);
CREATE INDEX idx_workflow_run_board ON workflow_run (board_id, started_at DESC);
CREATE INDEX idx_workflow_run_ticket ON workflow_run (ticket_id, started_at DESC);
CREATE INDEX idx_ticket_event_occurred ON ticket_event (occurred_at);
ALTER TABLE app_config ADD COLUMN history_retention_days INT NOT NULL DEFAULT 90;

-- Backfill: one summary per run that found an enabled workflow, from the history rows that run
-- already wrote. Legacy (pre-graph) runs count their rules as steps.
INSERT INTO workflow_run (run_id, ticket_id, board_id, workflow_id, workflow_name, event, source, dry_run,
    started_at, duration_ms, steps, actions, writes, notif_sent, notif_would_send, notif_none, errors, outcome)
SELECT s.run_id, s.ticket_id, s.board_id, s.workflow_id, s.workflow_name, s.event, s.source, s.dry_run,
    s.started_at, s.duration_ms, s.steps, s.actions, s.writes, s.notif_sent, s.notif_would_send, s.notif_none, s.errors,
    CASE
        WHEN s.errors > 0 THEN 'errors'
        WHEN s.steps = 0 THEN 'no_trigger'
        WHEN s.notif_none > 0 AND s.notif_sent + s.notif_would_send = 0 THEN 'nobody_notified'
        ELSE 'clean'
    END
FROM (
    SELECT w.run_id, w.ticket_id, t.board_id,
        (w.payload->>'workflow_id')::int AS workflow_id,
        coalesce(w.payload->>'workflow_name', '') AS workflow_name,
        coalesce(w.payload->>'event', CASE WHEN EXISTS (SELECT 1 FROM ticket_event c WHERE c.run_id = w.run_id AND c.kind = 'created') THEN 'created' ELSE 'updated' END) AS event,
        w.source, w.dry_run,
        agg.started_at, agg.duration_ms,
        coalesce(jsonb_array_length(w.payload->'steps'), jsonb_array_length(w.payload->'rules'), 0) AS steps,
        a.actions, a.writes, n.notif_sent, n.notif_would_send, n.notif_none,
        (e.errors + n.notif_err)::int AS errors
    FROM ticket_event w
    JOIN cw_ticket t ON t.id = w.ticket_id
    CROSS JOIN LATERAL (
        SELECT min(x.occurred_at) AS started_at,
               (extract(epoch FROM max(x.occurred_at) - min(x.occurred_at)) * 1000)::int AS duration_ms
        FROM ticket_event x WHERE x.run_id = w.run_id) agg
    CROSS JOIN LATERAL (
        SELECT count(*)::int AS actions,
               count(*) FILTER (WHERE x.payload->>'result' = 'ok' AND x.payload->>'kind' IN ('add_note','set_status','set_priority','set_owner','add_resource','patch'))::int AS writes
        FROM ticket_event x WHERE x.run_id = w.run_id AND x.kind = 'action') a
    CROSS JOIN LATERAL (
        SELECT count(*) FILTER (WHERE x.payload->>'result' = 'sent')::int AS notif_sent,
               count(*) FILTER (WHERE x.payload->>'result' = 'would_send')::int AS notif_would_send,
               count(*) FILTER (WHERE x.payload->>'result' = 'no_recipients')::int AS notif_none,
               count(*) FILTER (WHERE x.payload->>'result' = 'error')::int AS notif_err
        FROM ticket_event x WHERE x.run_id = w.run_id AND x.kind = 'notification') n
    CROSS JOIN LATERAL (
        SELECT count(*)::int AS errors
        FROM ticket_event x WHERE x.run_id = w.run_id AND (x.kind = 'error' OR (x.kind = 'action' AND x.payload->>'result' = 'error'))) e
    WHERE w.kind = 'workflow'
      AND coalesce((w.payload->>'found')::boolean, false)
      AND coalesce((w.payload->>'enabled')::boolean, false)
) s
ON CONFLICT (run_id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config DROP COLUMN IF EXISTS history_retention_days;
DROP INDEX IF EXISTS idx_ticket_event_occurred;
DROP TABLE IF EXISTS workflow_run;
-- +goose StatementEnd
