# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Ticketbot ingests ConnectWise PSA ticket webhooks, runs per-board workflows (conditions + actions), and notifies Webex rooms/people. Go 1.27, net/http (ServeMux), pgx, sqlc, goose; vanilla-JS dashboard embedded via `go:embed`.

## Commands

- `make db-up` / `make run`: Postgres 16 in Docker on `localhost:5433` (testuser/testpass/testdb), app via `go run .` on the host. Env comes from `./.env` (1Password-mounted; see `.env.example`). `make db-nuke` drops the DB volume; `make db-down` keeps it.
- `make run` reads `.env` line by line on purpose: macOS `/bin/sh` (bash 3.2) sources the 1Password FIFO as empty. Don't "simplify" it to `source`.
- `make gensql` (= `sqlc generate`) after editing `queries/*.sql` or adding a migration. Never hand-edit `internal/db/`.
- Tests: `go test ./...` is unit-only. Run `TEST_POSTGRES_DSN=<dsn> go test ./internal/postgres/` (against a DB already migrated to the current version) when touching repos, queries, or migrations. `internal/server/e2e_test.go` hits the live ConnectWise API and is gated on `TEST_TICKET_IDS`; leave it to the user.
- Dashboard files (`internal/web/static/`) are embedded at build time: restart `make run` to see changes, and hard-refresh the browser.
- Dependencies are not vendored; they come from the module proxy (tctg-go is public). Run `go mod tidy` after any `go.mod` change.

## Migrations

- Files: `migrations/000NN_snake_case.sql` with `-- +goose Up/Down` and `StatementBegin/End` markers; every migration needs a working Down.
- Bump `gooseMigrationVersion` in `main.go` to the new number or the migration will not run. Startup migrates to *exactly* that version (down as well as up), so lowering it rolls the schema back.
- sqlc reads the schema from `migrations/`, so add the migration before regenerating.

## Layering

`queries/*.sql` → `internal/db` (generated) → `internal/postgres` (repo impls) → `internal/repos` (interfaces) → `internal/service/*` → `internal/handlers` → `internal/server/routes.go`. New repos are registered in `internal/postgres/all.go` and `internal/repos/all.go`; new services are wired in `internal/server/server.go`.

## Frontend

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
browser console lists contrast, hit-target and accessible-name failures on the page.

## Conventions

- Feature branches off `main`; commit there with one-line Conventional Commits with scope (`feat(workflow): …`), no body. The user merges and deletes branches.
- Tests use embedded-interface partial fakes (see `internal/service/notifier/forwards_test.go`); `*psa.Client` is concrete, so services that need CW define a narrow interface (`workflow.CWClient`).
- `.gitignore` has `*.env` and `/ticketbot`; the latter must stay anchored or `internal/service/ticketbot/` gets ignored again.
- `.env.example` at the repo root is the canonical env var list. Every variable is read once in `internal/env/env.go` (`env.Load`); add new ones there and in `.env.example`, never with a stray `os.Getenv`. `ROOT_URL` is only required when `SKIP_HOOKS` is unset.
