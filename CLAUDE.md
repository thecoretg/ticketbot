# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Ticketbot ingests ConnectWise PSA ticket webhooks, runs per-board workflows (conditions + actions), and notifies Webex rooms/people. Go 1.27, net/http (ServeMux), pgx, sqlc, goose; vanilla-JS dashboard embedded via `go:embed`.

`docs/PLAN.md` is the v2 cutover work queue: pick the next unticked item there, and apply the
CLAUDE.md edits it lists. Delete the file and this paragraph when everything is ticked.

## Commands

- `make db-up` / `make run`: Postgres 16 in Docker on `localhost:5433` (testuser/testpass/testdb), app via `go run .` on the host. Env comes from `./.env` (1Password-mounted; see `.env.example`). `make db-nuke` drops the DB volume; `make db-down` keeps it.
- `make run` reads `.env` line by line on purpose: macOS `/bin/sh` (bash 3.2) sources the 1Password FIFO as empty. Don't "simplify" it to `source`.
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
`stale_alert_minutes` during 07:30 to 19:00 Central on weekdays) go to `ops_room_id` and the log.

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
canvas (`internal/web/static/workflows.js`, `cv*` functions) is the only editor; node heights are
fixed per kind and shared between `ui.css` and the script, so ports line up. Run history stores
`steps` per run; rows from before the graph carry `rules`, and `tickets.js` renders both. Every run
that found an enabled workflow also leaves a `workflow_run` summary row (written in `run.flush`)
that the Results tab lists; its detail replays the recorded path on the editor canvas with
`cv.replay` set, so canvas edits must check `cvEditable()` rather than `canEdit()`. Summaries and
history events age out together on `history_retention_days`.

## Frontend

`docs/USER_GUIDE.md` is the editor-facing explanation of how workflows run; the ⓘ buttons in the
workflow editor (`wfShowHelp` in `workflows.js`) carry the same text, so change both together.
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
