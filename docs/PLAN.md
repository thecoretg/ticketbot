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

Existing switches, keep all three: `master_dry_run` (app config, blocks every CW write),
per-workflow `dry_run`, and `MOCK_WEBEX` (env, stubs Webex posting only; retire it in item 3).

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

- [ ] `RequireConnectwiseSignature` returns 401 on a failed check. Add env `HOOK_SIGNATURE_MODE`
      (`enforce` default, `log` fallback) in `env.Load` and `.env.example`. Remove the flag after
      the first week of the parallel run.
- [ ] Patch batching: the engine collects every `set_status`, `set_priority`, `set_owner`,
      `add_resource` and `patch` operation from one run into a single PATCH. On conflict the last
      node in canvas order wins and a `warning` event names both nodes. Run order: one PATCH,
      then notes, then notifications.
- [ ] Rate cap: 20 mutating writes per ticket per rolling 15 minutes (app config, default 20).
      On hit, block writes to that ticket for one hour, record an `error` event, post to the ops
      room. This is the only loop protection on a fresh instance until ticketbot has posted a
      note, because `loopGuard` learns `cw_api_member_identifier` from that first note.
- [ ] Editors may toggle a workflow's own `dry_run`. `master_dry_run`, rate cap, ops room and
      redirect room are admin only.

CLAUDE.md: document batching and the conflict rule in the Workflows section; document the rate cap
and that the loop guard is inert until the first note.

### 3. Observability and the parallel run

- [ ] App config: `ops_room_id`, `redirect_room_id` (both nullable `webex_recipient` refs).
- [ ] Redirect: when set, every notification (room, person, resources_owner) goes to the redirect
      room, prefixed with the intended target. Then delete `MOCK_WEBEX` and `internal/mock`'s
      Webex stub, plus the `.env.example` line.
- [ ] Ops alerts (intake failure, rate cap, staleness) post to the ops room, falling back to
      `slog.Error` when unset.
- [ ] Staleness: no webhook received for 60 minutes between 07:30 and 19:00 America/Chicago,
      Monday to Friday, posts one alert and one recovery message. Threshold is app config.
- [ ] Dashboard: webhooks received per hour for the last seven days, so the threshold can be tuned
      from data. Danny wires an external uptime monitor to `GET /healthcheck` separately.

CLAUDE.md: list the new config keys; state that `MOCK_WEBEX` is gone.

### 4. CI

The deploy is a GitHub Action that calls an Easypanel build URL; Easypanel then builds the
Dockerfile from the watched branch (`v2-main` now, `main` at cutover).

- [ ] `.github/workflows/ci.yml` on push and pull request: `go vet`, `gofmt -l`, golangci-lint
      (config exists in `.golangci.yml`), `go test ./...`, Postgres 16 service container for
      `TEST_POSTGRES_DSN=... go test ./internal/postgres/` after migrating with goose, and
      `docker build` with no push.
- [ ] Deploy job: on the watched branch only, after the checks pass, call the Easypanel URL from
      a repository secret.

CLAUDE.md: add a line under Commands saying CI runs the same checks as `.claude/skills/verify`.

### 5. Workflow export and import

- [ ] Export one or all workflows as JSON. Recipients are embedded by email or room name plus
      type, lists by name with their items, so nothing references a database ID.
- [ ] Import creates missing recipients and lists, then the workflows, and reports what it
      created. Validate through `Validate` in `internal/service/workflow/service.go` before saving.

CLAUDE.md: one line under Workflows on the export format.

## During the parallel run

None of these change engine behaviour, so they can ship without restarting the run.

### 6. Editor surfaces for non-programmers

- [ ] Condition builder on the `if` node: field picker from `ConditionFields`
      (`internal/service/workflow/fields.go`), operators is, is not, is empty, contains, in list,
      and for fields with a `changed/*` and `old/*` pair, "changed to" and "changed from". The
      builder compiles to `cwquery` text; an Advanced toggle shows and edits the raw DSL. Round
      trip: a DSL the builder cannot represent opens in Advanced.
- [ ] Notify node: placeholder picker from `msgtemplate.Placeholders` with the descriptions as
      labels, inserting at the cursor, and a live preview rendered against a ticket chosen from
      the recent tickets list.
- [ ] Help buttons (not inline text) on each node kind and on the builder, explaining triggers
      fire per event, a walk ends at an unwired port, `skip_notify` is per walk, try dry run first.
- [ ] `docs/USER_GUIDE.md`: one page for the concepts above. Danny trains the team in person.

CLAUDE.md: mention the builder and picker in the Frontend section; keep the canvas as the only
editor.

### 7. Notify as a channel

- [ ] Reshape `NotifyAction`: `Target NotifyTarget` becomes a channel type with kind
      (`webex_room`, `webex_person`, `resources_owner`) and per-kind params. No new channels.
      Upgrade on read like the v1 rule chain (`models.DecodeWorkflowDocument`).
- [ ] Slack, Teams, client-facing notifications, Notion sync and reporting are deliberately
      unplanned. Add a channel kind only when one is scheduled.

CLAUDE.md: update the Workflows section to describe channels.

### 8. Backups

- [ ] Easypanel nightly Postgres dump to object storage. Infrastructure task, not code; tick it
      when it is configured on the new instance.

## Done

Every box above ticked: delete this file, remove its pointer from CLAUDE.md, and confirm every
CLAUDE.md edit listed here has landed.
