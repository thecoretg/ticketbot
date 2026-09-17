# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Ticketbot ingests ConnectWise PSA ticket webhooks, runs per-board workflows (conditions + actions), and notifies Webex rooms/people. Go 1.26, gin, pgx, sqlc, goose; vanilla-JS dashboard embedded via `go:embed`.

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

## Conventions

- Feature branches off `main`; commit there with one-line Conventional Commits with scope (`feat(workflow): …`), no body. The user merges and deletes branches.
- Tests use embedded-interface partial fakes (see `internal/service/notifier/forwards_test.go`); `*psa.Client` is concrete, so services that need CW define a narrow interface (`workflow.CWClient`).
- `.gitignore` has `*.env` and `/ticketbot`; the latter must stay anchored or `internal/service/ticketbot/` gets ignored again.
- `.env.example` at the repo root is the canonical env var list. `ROOT_URL` is only required when `SKIP_HOOKS` is unset.
