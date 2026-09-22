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

- [x] `RequireConnectwiseSignature` returns 401 on a failed check. The temporary
      `HOOK_SIGNATURE_MODE=log` fallback was removed once live traffic proved the check.
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

- [x] Easypanel nightly Postgres dump to object storage. Danny owns this on the Easypanel side;
      marked done 2026-09-21 on his word, nothing in the repo depends on it.

## Do last

Polish noted during the 2026-09-21 review of items 1 to 7. None blocks the parallel run.

Decisions from the 2026-09-21 grilling of items 9 to 12. Order of work: 12, 11, 10, 9.

### 9. Tooltip clipped on the ticket list

Done. The ticket id link in the Tickets table lost its `data-tip` and gained a trailing
external-link icon (`icons.js` `external`, `.link.ext` in `app.css`) with an accessible label.

### 10. Configurable business hours for the stale-webhook alert

Done. Migration 18 adds `business_open`, `business_close`, `business_days`, `business_zone` to
app config (defaults 07:30, 19:00, mon..fri, America/Chicago). `models.ParseBusinessWindow`
validates them on save; `Config.BusinessWindow` parses them for `intake.staleWatch`, falling back
to the defaults if a stored row is somehow bad. Config page: two time inputs, a zone select (US
zones plus UTC) and weekday checkboxes beside "Stale webhook alert".

### 11. Canvas context menu and multi-select

Done.

- [x] Right-click on a step opens the app popover at the pointer (`openMenuAt` in `app.js`):
      Open inspector, Duplicate, Copy, Enable or disable, Delete. Right-click on empty canvas:
      Paste (disabled until something is copied), Add a step (opens the step list), Fit to view.
      Shift plus right-click keeps the browser menu. Viewers get Open inspector only.
- [x] Shift-drag on empty canvas draws a `.marquee` and selects the steps it touches; Shift-click
      adds or removes one. Dragging any selected card moves the group. A "N steps selected"
      panel replaces the inspector with Duplicate, Copy, Enable or disable, Delete. ⌘C, ⌘V, ⌘D
      and Delete act on the selection.
- [x] `cvClipboard` lives for the browser session and crosses workflows. Paste keeps the wires
      between copied steps, drops the rest, keeps the layout, gives new ids, lands at the pointer
      from the menu or offset by 32px from the originals from the keyboard or Duplicate. Deleting a
      selection keeps the last trigger.

### 12. Workflow run results page

Done.

- [x] `workflow_run` (migration 17) written by `run.flush` for every run where an enabled
      workflow was found, including runs where no trigger listened. Counts, duration and the
      outcome (`clean`, `errors`, `nobody_notified`, `no_trigger`, derived by `Verdict`). The
      migration backfilled 109 runs from the existing history on the local database; the live
      instance backfills on deploy.
- [x] Results tab beside the workflow list at `#workflows/results`, filters (board, outcome, date
      range, ticket) in the hash, 5-second polling of the first page, "Load older runs" paging.
      `GET /workflows/runs` and `GET /workflows/runs/{run_id}`, viewer-readable.
- [x] Run detail (`results.js`): the workflow's canvas in replay (`cv.replay`: pan and zoom only,
      no rail, no edits) with the recorded path lit and a "Recorded run" side panel, steps whose
      node is gone listed under the canvas, then the run's events rendered by the ticket history
      renderer.
- [x] **Results** button in the editor toolbar opens the tab filtered to the board.
- [x] `history_retention_days` (default 90, 0 forever) purges `workflow_run` and `ticket_event`
      together on the intake service's hourly tick (`ticketbot.HistoryPurger`).

### 13. Conditional triggers

Done.

- [x] A trigger node may carry a `condition` (the field an If already has). Empty fires on every
      event it listens for, as today. A failing condition, or one that errors at run time, does
      not start the walk; the run still records the trigger step as "did not fire" (with the error
      when there is one) so Results shows the lane dark but present. A run where no trigger fired
      keeps outcome `no_trigger`.
- [x] Inspector section "Only when" with the same builder as an If. The card keeps the short
      height and shows the condition in its subtitle. Validation applies the If's condition rules
      to triggers. Simulate honours it. Export, import and the results page need nothing.

## Read-only MCP server

Decisions from the 2026-09-22 design review. Ships to the v2 instance during the parallel run
behind `mcp_enabled` (app config, default off); does not gate cutover. Write tools are a later
project, after cutover is stable.

Settled: remote MCP only (ticketbot never calls Anthropic); OAuth 2.1 hand-rolled, PKCE, public
clients, Dynamic Client Registration with spec-permissive redirect URIs; opaque tokens stored as
SHA-256, access 1h, refresh 30d rotated with replay revocation; scopes `read` and `write` where
the effective permission is the lower of the user's role and the granted scope, only `read`
offered now; existing API-key bearer also accepted at `/mcp`; tools the caller's role cannot use
are hidden from `tools/list`; tool names carry a `ticketbot_` prefix so they never collide with
the `cw_*` ConnectWise PSA connector.

### 14. OAuth authorization server

Done (migration 19).

- [x] Migration: `oauth_client` (DCR rows, expire after ten minutes without a grant),
      `oauth_code`, `oauth_grant` (user, client, scopes, created, last used) and `oauth_token`
      (access and refresh hashes, expiry, grant). `mcp_enabled` column on `app_config`.
- [x] `internal/service/oauth`: `/.well-known/oauth-protected-resource`,
      `/.well-known/oauth-authorization-server`, `POST /oauth/register`, `GET /oauth/authorize`
      (validates, then forwards its query string to `/oauth/consent`, which serves the dashboard;
      the SPA signs the user in first when there is no session),
      `POST /oauth/token` (code with PKCE, refresh with rotation), `POST /oauth/revoke`. Every
      endpoint 404s while `mcp_enabled` is off.
- [x] Consent JSON endpoints: `GET /oauth/authorize/info` and `POST /oauth/authorize/decide`
      (approve or deny). Consent is always shown; it names the client and its redirect host.
- [x] `GET /users/me/grants`, `DELETE /users/me/grants/{grant_id}` and, for admins,
      `GET /users/grants/{id}` and `DELETE /users/grants/{id}/{grant_id}` (the `/users/keys`
      shape, since `/users/{id}/grants` conflicts with `/users/keys/{id}`).
- [x] Hourly sweeper purges expired codes, tokens, abandoned clients and expired `session` rows
      (nothing purges sessions today).
- [x] Entra `GET /auth/sso/start` already accepts `?next=` (tctg-go/entra `safeNext`); the password
      login is a same-page SPA form, so only the dashboard needs to carry the path (item 16).
- [x] CLAUDE.md: "MCP and OAuth" section describing the two switches (`mcp_enabled`, scopes) and the
  layering of `internal/service/oauth` and `internal/service/mcp`.

### 15. MCP endpoint and read tools

- [ ] `POST /mcp` via `github.com/modelcontextprotocol/go-sdk`, Streamable HTTP, stateless.
      Bearer resolves to a user through an OAuth access token or an API key; the 401 carries
      `WWW-Authenticate` with `resource_metadata`.
- [ ] `internal/service/mcp` tool registry: each tool declares its minimum role; `tools/list`
      filters by the caller's effective permission. One structured log line per call (user,
      client, tool, argument summary).
- [ ] Tools, all read-only, `limit` default 20 ceiling 100 with a continuation cursor:
      `ticketbot_list_workflows`, `ticketbot_get_workflow` (readable lane walk, `raw` for the
      document), `ticketbot_list_runs`, `ticketbot_get_run`, `ticketbot_ticket_history`,
      `ticketbot_list_lists`, `ticketbot_list_forwards`, `ticketbot_get_config`,
      `ticketbot_simulate`, `ticketbot_evaluate_condition`, `ticketbot_lookup_ids` (cached
      boards, statuses, members, priorities, companies, contacts; description points at the
      ConnectWise connector for live ticket data), `ticketbot_list_webex_rooms`; admin only:
      `ticketbot_intake_status` (stats and failed rows), `ticketbot_tail_logs` (default 200).

### 16. Dashboard and docs

- [ ] Login view honours `next`. Consent view. "Connected apps" section on the profile page
      listing grants (client, scopes, created, last used) with revoke; Users page row action
      revokes another user's grants. Settings gets the `mcp_enabled` switch.
- [ ] `docs/USER_GUIDE.md` "Connect Claude" section and the matching ⓘ on Connected apps, with
      the connector URL and steps.

## Cutover

The steps that end the parallel run, as boxes so the delete rule below holds.

- [x] Removed the `HOOK_SIGNATURE_MODE=log` fallback (2026-09-21): signatures are always enforced.
- [ ] Change the deploy branch to `main` in `.github/workflows/ci.yml` and in Easypanel, in one
      change. Push `v2-main` to `main`.
- [ ] On the instance: disable v1's ConnectWise callbacks, clear the redirect room, turn master
      dry run off. Retire the Lightsail v1 instance.
- [ ] Remove `MOCK_WEBEX` from every `.env` that still has it.

## Done## Done

Every box above ticked, including Do last: delete this file, remove its pointer from CLAUDE.md, and confirm every
CLAUDE.md edit listed here has landed.
