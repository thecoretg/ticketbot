# v2 cutover plan

Decisions from the 2026-09-21 design review. This file is the work queue for whoever picks up the
next item, human or agent. Work one item per branch (`v2-<purpose>` off `v2-main`), tick it off
here in the same branch, and apply the CLAUDE.md edits listed under the item. When every box is
ticked, delete this file and the pointer to it in CLAUDE.md.

## Context

- v1 (`main`) runs in production on Lightsail. v2 (`v2-main`) will run on a fresh Easypanel
  instance. Nothing migrates; workflows are rebuilt (or imported, item 5) on the new server.
- Rewst v1 ticket workflows are rebuilt by hand as ticketbot workflows. Ticketbot stays
  ticket-scoped; Rewst keeps everything else.
- Users: Danny builds workflows now; management and automation engineers with no scripting
  background will build and edit them later. Every editor-facing surface must work without
  knowing the `cwquery` DSL or placeholder names.
- Failure ranking, worst first: silent downtime, wrong write to ConnectWise, misdirected or
  duplicate notification.
- Cutover: run v2 against live webhooks for two to four weeks with `master_dry_run` on and every
  notification redirected to one test room (item 3). Cut over when the run log is boring:
  disable v1's callbacks, turn off `master_dry_run` and the redirect.
- The new instance reuses v1's ConnectWise API member and keys, so the loop guard treats v1's
  notes as its own during the parallel run.
- Single app instance. No distributed locking anywhere.

Switches: `master_dry_run` (app config, blocks every CW write), per-workflow `dry_run`, and the
redirect room (app config, sends every notification to one room even under dry run).

## Before the parallel run

### 1. Persisted webhook intake

Done. `internal/service/intake` owns the queue; `webhook_intake` is the table (migration 14).

- [x] Migration, queries, sqlc.
- [x] Handler inserts a row and returns 200 (500 when the insert itself fails, so ConnectWise
      retries the delivery). Deletes go through the same table.
- [x] Four workers claim rows through a query that never hands out a ticket with an older open
      row, so one ticket's webhooks run in arrival order and never concurrently. Backoff 5s, 30s,
      2m, 10m, 30m, 1h, then `failed`. An attempt ignores shutdown cancellation and `Wait` gives
      it the shutdown timeout to finish; rows still in flight are reset to pending on start.
- [x] Intake page under Administration: counts per status, filterable table, retry and discard
      on failed rows. Failure alerts go through `intake.Alerter`, which logs until item 3 wires
      the ops room. Ticket-level `error` events are still recorded by the ticketbot run itself.
- [x] `done` and `discarded` rows purge hourly on `log_retention_days`.
- [x] Worker unit tests with fakes; claim-ordering test in `internal/postgres`.
- [x] Sync (manual, from the Sync page) stays as the backstop. Processing is diff-based, so sync
      and a late queued webhook cannot double-fire.

### 2. Write safety

Done.

- [x] `RequireConnectwiseSignature(enforce)` returns 401 on a failed check. `HOOK_SIGNATURE_MODE`
      (`enforce` default, `log` fallback) in `env.Load` and `.env.example`. Remove the `log`
      option after the first week of the parallel run.
- [x] Patch batching in the engine: `set_*` and `patch` operations queue into one PATCH sent after
      the walks; the last node to set a path wins, the earlier action reports `superseded`, and
      the pair is listed under `conflicts` on the workflow event. Two `add_resource` nodes merge.
      Queued operations are applied to a copy of the ticket so later `if` nodes see the intended
      state; a dry run or a failed PATCH hands the caller's original ticket back. Notes post after
      the PATCH, then one refetch; notifications follow as before.
- [x] Rate cap: `write_cap_per_ticket` (app config, default 20) per rolling 15 minutes via
      `workflow.RateLimiter`; on hit the ticket is blocked for an hour, every queued write reports
      an error, the run records an `error` event with stage `rate_cap`, and `alerts.Alerter` fires
      (log sink until item 3). This is the only loop protection on a fresh instance until ticketbot
      has posted a note, because `loopGuard` learns `cw_api_member_identifier` from that first note.
- [x] Editors may toggle a workflow's own `dry_run` (PUT /workflows is editor); `master_dry_run`,
      the write cap and the coming ops and redirect rooms live on PUT /config, which is admin only.

### 3. Observability and the parallel run

Done.

- [x] App config: `ops_room_id`, `redirect_room_id` (nullable `webex_recipient` refs, 0 clears on
      the wire), `stale_alert_minutes` (default 60). Migration 16. Config page has pickers.
- [x] Redirect: with a room set, every notification (room, person, resources_owner, forwards) is
      re-addressed to it with an "intended for" prefix and is sent even under dry run; the stored
      notification keeps the intended recipient and the history event shows `redirected_to`.
      `MOCK_WEBEX` and `internal/mock` are gone; the message sender is always the real client.
- [x] `alerts.Webex` posts intake failures, write-cap blocks and staleness to the ops room and
      always logs; without a room it only logs.
- [x] Staleness: `intake.staleWatch` checks every minute; no webhook for `stale_alert_minutes`
      between 07:30 and 19:00 America/Chicago on a weekday alerts once, and the next webhook posts a
      recovery. Silence outside those hours never alerts.
- [x] `GET /intake/hourly` and a webhooks-per-hour bar chart on the Intake page for the last seven
      days. Danny wires an external uptime monitor to `GET /healthcheck` separately.

### 4. CI

Done. `.github/workflows/ci.yml`.

- [x] On every push and pull request: gofmt, build, vet, golangci-lint 2.13, `sqlc generate` with
      a clean-diff check on `internal/db`, unit tests, goose-migrated Postgres 16 service container
      for `internal/postgres`, `node --check` on the dashboard scripts, and `docker build`.
- [x] Deploy job runs only on a push to `v2-main` after the checks pass and POSTs the
      `EASYPANEL_DEPLOY_URL` repository secret. At cutover change the branch in the workflow's
      `if` and in Easypanel. Add the secret in the repository settings before the first push.

### 5. Workflow export and import

Done. `internal/service/transfer`, `GET /workflows/export`, `POST /workflows/import`, buttons on
the Workflows page.

- [x] The bundle (`models.WorkflowBundle`, version 1) carries the workflows plus the Webex
      recipients (by Webex id) and lists (by name, with member ids) their nodes reference, keyed
      by the source database id. ConnectWise ids travel unchanged.
- [x] Import matches recipients by Webex id and lists by name, creates what is missing (skipping
      list members the synced data does not know, with a warning), rewrites `recipient_id` and
      `in list N` tokens, then creates each workflow through `Create` or, with replace, `Replace`,
      so validation runs. The report lists created recipients and lists and each workflow's
      created / replaced / skipped / error.

## During the parallel run

None of these change engine behaviour, so they can ship without restarting the run.

### 6. Editor surfaces for non-programmers

Done.

- [x] The condition builder already existed (`condition.js`: field picker, is / is not / contains
      / starts with / one of / empty / in list, match all or any, Advanced raw text with round
      trip and a reason when the builder cannot show a condition). Added **changed to** and
      **changed from**: `ConditionField` now carries `changed_path` and `old_path` companions
      (derived by path in `fields.go`), the builder compiles them to
      `(changed/x = true and x = v)` and `old/x = v`, and parses those shapes back into one row.
- [x] Placeholder picker already existed. Added `POST /workflows/preview-message` and a Preview
      control on the Notify step that renders the message (or the default layout) with a stored
      ticket.
- [x] Help buttons: ⓘ in the editor toolbar, on the Condition label and on the message preview,
      all opening one "How workflows run" modal (`wfShowHelp`) that scrolls to the relevant
      section. No new inline text.
- [x] `docs/USER_GUIDE.md`. Danny trains the team in person.

### 7. Notify as a channel

Done.

- [x] `NotifyAction.Channel` (`webex_room`, `webex_person`, `resources_owner`) replaces `Target`.
      `NotifyAction.Normalize` maps a pre-channel `target` on read (`DecodeWorkflowDocument`, the
      v1 rule upgrade, validation and the engine all call it), so stored rows, old exports and old
      API bodies still load; the field is never written back. `NotifyChannel.RecipientType` says
      which recipient row a channel needs. Action events record `channel`; the history renders
      both spellings.
- [x] Slack, Teams, client-facing notifications, Notion sync and reporting stay unplanned. A new
      transport is a new channel kind with its own params, resolved in
      `notifier.resolveTarget`, not a new action kind.

### 8. Backups

- [ ] Easypanel nightly Postgres dump to object storage. Infrastructure task, not code; tick it
      when it is configured on the new instance.

## Do last

Polish noted during the 2026-09-21 review of items 1 to 7. None blocks the parallel run.

### 9. Tooltip clipped on the ticket list

- [ ] The "Open in ConnectWise" tooltip on a ticket id in the Tickets table is cut off at the
      table's edge (`tickets.js`, the `data-tip` on the id link; `.table-wrap` clips overflow).
      Flip the tooltip side near an edge or let it escape the wrap. Check light and dark.

### 10. Configurable business hours for the stale-webhook alert

- [ ] Replace the constants in `internal/service/intake/stale.go` (07:30 to 19:00, weekdays,
      America/Chicago) with app config: open time, close time, time zone, days. Config page rows
      beside "Stale webhook alert"; the user guide and CLAUDE.md sentence about business hours
      follow.

### 11. Canvas context menu

- [ ] Right-click on a node opens a menu (`openMenu` in `app.js` is the existing popover):
      Duplicate, Copy, Delete, Enable/Disable, Open inspector. Right-click on empty canvas offers
      Paste when the clipboard holds a copied node (keep an in-page clipboard, not the system
      one), placing it at the pointer with a new id and no wires. Keyboard equivalents where
      cheap (⌘C / ⌘V / Delete already handles removal).

## Done## Done

Every box above ticked, including Do last: delete this file, remove its pointer from CLAUDE.md, and confirm every
CLAUDE.md edit listed here has landed.
