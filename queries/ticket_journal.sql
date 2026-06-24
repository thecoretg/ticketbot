-- name: GetTicketJournal :one
SELECT * FROM ticket_journal
WHERE ticket_id = $1 LIMIT 1;

-- name: ListTicketJournalSummaries :many
SELECT ticket_id, summary, board_name, company_name, contact_name, status_name,
       owner_name, type_name, subtype_name, item_name, last_trigger, last_outcome,
       had_error, first_seen, last_run
FROM ticket_journal
ORDER BY last_run DESC
LIMIT $1;

-- name: UpsertTicketJournal :one
INSERT INTO ticket_journal(
    ticket_id, summary, board_name, company_name, contact_name, status_name,
    owner_name, type_name, subtype_name, item_name, last_trigger, last_outcome,
    had_error, first_seen, last_run, runs
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (ticket_id) DO UPDATE SET
    summary      = EXCLUDED.summary,
    board_name   = EXCLUDED.board_name,
    company_name = EXCLUDED.company_name,
    contact_name = EXCLUDED.contact_name,
    status_name  = EXCLUDED.status_name,
    owner_name   = EXCLUDED.owner_name,
    type_name    = EXCLUDED.type_name,
    subtype_name = EXCLUDED.subtype_name,
    item_name    = EXCLUDED.item_name,
    last_trigger = EXCLUDED.last_trigger,
    last_outcome = EXCLUDED.last_outcome,
    had_error    = EXCLUDED.had_error,
    last_run     = EXCLUDED.last_run,
    runs         = EXCLUDED.runs
RETURNING *;

-- name: DeleteTicketJournalsOlderThan :execrows
DELETE FROM ticket_journal
WHERE last_run < $1;
