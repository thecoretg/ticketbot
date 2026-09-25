# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Ticketbot ingests ConnectWise PSA ticket webhooks, runs per-board workflows (conditions + actions), and notifies Webex rooms/people. Go 1.27, net/http (ServeMux), pgx, sqlc, goose; vanilla-JS dashboard embedded via `go:embed`.

## Work tracking

Work is tracked in Notion, not in the repo. The [Ticketbot Tracker](https://app.notion.com/p/3e63ef0e1b0680ba9e99d3f7e0be4d70)
page holds the rules and two databases: **Work** (`collection://90891b8e-32af-43f9-9b74-0fce57fd603d`)
and **Handoffs** (`collection://33a3c724-da94-44c4-87a1-34b9bffe6028`). Read the page's rules before
the first change in a session. In short:

- Pick up Work rows with Status `Next`, or whatever Danny asks for. `Hold` rows wait on data or a
  decision; don't start them. Type `Cutover` rows start only when Danny says "start cutover".
- A new request becomes a row first (Status `Idea`, or `Next` if Danny says to start).
- Starting: Status `In progress` and fill Branch. Merged: Status `Done`, fill Commit, and add a
  closing note to the row's body.
- Never suggest a `Declined` row again unless Danny raises it.
- A session that leaves work open ends with a Handoffs page (date, branch and pushed state, linked
  Work rows, what to read first). Start from the newest one when a row doesn't say where things stand.
- Property names and select options are what these rules match on: don't rename them. No secrets
  in Notion.

Without the Notion connector, ask Danny rather than guessing what is next. The v2 plan that
preceded this (design decisions for plan items 1 to 16) is `docs/PLAN.md` in git history; its
last version is at `42af684`.

## Commands

- `make db-up` / `make run`: Postgres 16 in Docker on `localhost:5433` (testuser/testpass/testdb), app via `go run .` on the host. Env comes from `./.env` (1Password-mounted; see `.env.example`). `make db-nuke` drops the DB volume; `make db-down` keeps it.
- `make run` loads env through `op run --environment $(OP_ENVIRONMENT)` (set in the Makefile), because the 1Password `.env` FIFO serves one read and then empty ones; don't read `.env` yourself before starting it. `make run OP_ENVIRONMENT=` falls back to reading `ENV_FILE` line by line, on purpose: macOS `/bin/sh` (bash 3.2) sources the FIFO as empty. Don't "simplify" it to `source`.
- `make gensql` (= `sqlc generate`) after editing `queries/*.sql` or adding a migration. Never hand-edit `internal/db/`.
- Tests: `go test ./...` is unit-only. Run `TEST_POSTGRES_DSN=<dsn> go test ./internal/postgres/` (against a DB already migrated to the current version) when touching repos, queries, or migrations. `internal/server/e2e_test.go` hits the live ConnectWise API and is gated on `TEST_TICKET_IDS`; leave it to the user.
- Dashboard files (`internal/web/static/`) are embedded at build time: restart `make run` to see changes, and hard-refresh the browser.
- CI (`.github/workflows/ci.yml`) runs the same checks as the `verify` skill plus golangci-lint, a
  clean `sqlc generate` diff and a `docker build`, then calls the Easypanel deploy URL on the
  watched branch. A red check blocks the deploy.
- Dependencies are not vendored; they come from the module proxy (tctg-go is public). Run `go mod tidy` after any `go.mod` change.

## Migrations

- Files: `migrations/000NN_snake_case.sql` with `-- +goose Up/Down` and `StatementBegin/End` markers; every migration needs a working Down.
- Bump `gooseMigrationVersion` in `main.go` to the new number or the migration will not run. Startup migrates to *exactly* that version (down as well as up), so lowering it rolls the schema back.
- sqlc reads the schema from `migrations/`, so add the migration before regenerating.

## Switches and alerts

There is no Webex mock. `master_dry_run` (app config) blocks every ConnectWise write and turns
notifications into `would_send`; a workflow's own `dry_run` does the same for that workflow. The
redirect room (`redirect_room_id`) sends every notification to one room with an "intended for"
prefix and overrides dry run for sending, which is how the parallel run sees real messages.
Operator alerts (`internal/service/alerts`: failed intake rows, write-cap blocks, no webhooks for
`stale_alert_minutes` during the configured business hours) go to `ops_room_id` and the log.

## MCP and OAuth

`internal/service/oauth` is the authorization server in front of the MCP endpoint (`/mcp`):
OAuth 2.1 code flow with PKCE, public clients only, dynamic client registration, refresh
rotation with replay revocation, and the RFC 8414 and RFC 9728 metadata documents. Every URL is
built from `ROOT_URL`. `mcp_enabled` (app config) is the switch: off, all of it answers 404,
while grants and tokens already issued are kept. `GET /oauth/authorize` only validates and
forwards its query string to the dashboard at `/oauth/consent`; the SPA signs the user in if
needed, then calls `GET /oauth/authorize/info` and `POST /oauth/authorize/decide`. Scopes are
`read` and `write`; only `read` is offered (`oauth.OfferedScopes`), and a caller's effective
permission is the lower of their role and their granted scopes. Tokens and codes are stored as
SHA-256 like sessions. The service and `authsvc.SessionPurger` run on the intake purge tick.
`internal/service/mcp` is the endpoint: `auth.RequireBearerToken` from the go-sdk resolves an
OAuth access token or an API key to a `Principal`, then a stateless Streamable HTTP server is
built per request holding only the tools that principal's role and scopes allow, so
`tools/list` is already filtered. Tools live in `tools.go` as `def(...)` entries with a role
floor; every dependency is a narrow interface in `Deps`. Tool names carry the `ticketbot_`
prefix so they never collide with the ConnectWise PSA connector's `cw_*` tools, and
`ticketbot_get_workflow` renders the graph as lanes (`describe.go`) rather than dumping nodes
and edges. Simulate and evaluate-condition logic is in `internal/service/simulate`, shared by
the dashboard handlers and the MCP tools. One `mcp: tool call` log line per call is the audit trail.

## Intake

`POST /hooks/cw/tickets` only inserts a `webhook_intake` row and returns 200. Workers in
`internal/service/intake` claim rows with a query that never hands out a ticket that has an older
open row, so one ticket's webhooks run in arrival order and never concurrently; the worker count
is otherwise free. Failed attempts back off (5s to 1h, six tries) and then park as `failed` for the
Intake page's retry or discard. The queue assumes a single app instance: rows left `processing`
are reset to `pending` on start. Ticket intake logic itself stays in `internal/service/ticketbot`.

## Layering

`queries/*.sql` → `internal/db` (generated) → `internal/postgres` (repo impls) → `internal/repos` (interfaces) → `internal/service/*` → `internal/handlers` → `internal/server/routes.go`. New repos are registered in `internal/postgres/all.go` and `internal/repos/all.go`; new services are wired in `internal/server/server.go`.

## Workflows

A workflow is a graph, not a rule list: `models.Workflow` holds `Nodes` (trigger / if / one node per
`ActionKind`) and `Edges` (from node + port → to node). The `workflow.rules` JSONB column stores a
`WorkflowDocument` (`{"version":2,"nodes":[],"edges":[]}`); a bare array is the v1 rule chain and
`models.DecodeWorkflowDocument` upgrades it on read (`UpgradeRules`), so no SQL migration was needed
and v1 rows persist until their next save. Engine semantics (`internal/service/workflow/engine.go`):
every enabled trigger listening for the event fires, in canvas order; a walk follows the port an if
node picks and ends at an unwired port; a node reached by a second walk is recorded as `joined` and
not re-run; `skip_notify` silences only the notifies after it on its own walk; disabled nodes pass
through (an if takes its `no` port). Ticket writes are batched: `set_*` and `patch` operations
queue into one PATCH sent after every walk, the last node to set a path wins (the earlier action
reports `superseded` and the workflow event lists the conflict), two `add_resource` nodes merge,
and notes post after the PATCH. Queued operations are applied to a copy of the ticket so later if
nodes see the intended state. `workflow.RateLimiter` caps writes per ticket (`write_cap_per_ticket`
in app config, 15-minute window, one-hour block) and is the only loop protection until ticketbot
has posted a note, because `loopGuard` learns its own member identifier from that first note.
A notify node delivers on a channel (`webex_room`, `webex_person`, `resources_owner`); a new
transport is a new channel kind resolved in `notifier.resolveTarget`, not a new action kind.
Documents that still say `target` are upgraded on read by `NotifyAction.Normalize`.
`internal/service/transfer` exports workflows as a bundle that carries their Webex recipients (by
Webex id) and lists (by name) and rewrites those references on import; ConnectWise ids are the
same on every instance and travel as they are. `Validate` in `service.go` owns the structural rules (one
trigger minimum, one wire per port, nothing into a trigger, no cycles, everything reachable). The
canvas (`internal/web/static/workflows.js`, `cv*` functions) is the only editor; every card is 88px
tall (`CV_H`, shared between `ui.css` and the script) so ports line up, and a trigger or if card
shows its condition as a count badge (`cvCondBadgeHTML`) with a hover card, never as text. Run history stores
`steps` per run; rows from before the graph carry `rules`, and `tickets.js` renders both. Every run
that found an enabled workflow also leaves a `workflow_run` summary row (written in `run.flush`)
that the Results tab lists; its detail replays the recorded path on the editor canvas with
`cv.replay` set, so canvas edits must check `cvEditable()` rather than `canEdit()`. Summaries and
history events age out together on `history_retention_days`.

## Frontend

`docs/USER_GUIDE.md` is the editor-facing explanation of how workflows run; the ⓘ buttons in the
workflow editor (`wfShowHelp` in `workflows.js`) and on Connected apps (`connectedShowHelp` in
`app.js`) carry the same text, so change both together. `index.html` is also served at
`/oauth/consent`, so its asset paths are absolute; `enterApp()` decides between the dashboard and
the consent card after any sign-in, and `#connected` is a page without a sidebar entry
(`EXTRA_TABS` names it in the trail).
The condition builder (`condition.js`) is a view over the stored condition text, with "changed
to" and "changed from" compiled from each field's `changed_path` and `old_path` companions.

`internal/web/static/ui.css` is the design system: tokens, component classes and six palettes
(`data-palette` on `<html>`, pinned to `harbor`). `app.css` holds app-specific rules only. Both
are ours. Edit either, but keep the split: a reusable component belongs in `ui.css`, a one-off
in `app.css`, each with a comment saying why. `docs/ui/COMPONENTS.md` documents the component
markup; compose from it before writing anything new. `icons.js` is the icon set, loaded as a
module in `index.html` and exposed as `window.icon` for the classic scripts.

Never write a literal colour, spacing or radius. Everything comes from the `var(--*)` tokens at
the top of `ui.css`. Spacing is the 4px scale (`--s1`…`--s12`). Text clears 4.5:1 against its
actual background; never dim text with `opacity`. No other CSS frameworks or component libraries.

Every change is checked in light **and** dark. `scripts/contrast-audit.js` pasted into the
browser console (then `uiAudit()`) lists contrast, hit-target and accessible-name failures on the page.
Static files are embedded: `.claude/launch.json` starts `make run` for the in-app browser preview.

## Conventions

- Feature branches off `main`; commit there with one-line Conventional Commits with scope (`feat(workflow): …`), no body. The user merges and deletes branches.
- Tests use embedded-interface partial fakes (see `internal/service/notifier/forwards_test.go`); `*psa.Client` is concrete, so services that need CW define a narrow interface (`workflow.CWClient`).
- `.gitignore` has `*.env` and `/ticketbot`; the latter must stay anchored or `internal/service/ticketbot/` gets ignored again.
- `.env.example` at the repo root is the canonical env var list. Every variable is read once in `internal/env/env.go` (`env.Load`); add new ones there and in `.env.example`, never with a stray `os.Getenv`. `ROOT_URL` is only required when `SKIP_HOOKS` is unset.
