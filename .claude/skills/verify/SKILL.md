---
name: verify
description: Run the full local verification for this repo (build, vet, unit tests, gofmt on changed files, and Postgres integration tests when a test DB is reachable). Use before committing or when asked to check that changes work.
---

Run these in order and stop at the first failure, reporting the output:

1. `gofmt -l $(git diff --name-only --diff-filter=AM HEAD -- '*.go'; git ls-files --others --exclude-standard -- '*.go')` — any listed file must be formatted with `gofmt -w`. Do not reformat files you did not change.
2. `go build ./... && go vet ./...`
3. `go test ./...`
4. If `TEST_POSTGRES_DSN` is set, or the scratch DB answers `pg_isready -h localhost -p 5433`, run:
   `TEST_POSTGRES_DSN="${TEST_POSTGRES_DSN:-postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable}" go test ./internal/postgres/`
   The DB must already be migrated to the current `gooseMigrationVersion`; if the tests fail on a missing column or table, say so rather than migrating it yourself.
5. If `queries/*.sql` or `migrations/` changed, run `sqlc generate` and confirm `git status` shows no unexpected diff under `internal/db/`.
6. If dashboard files under `internal/web/static/` changed, run `node --check` on each changed `.js` file.

Never run `internal/server/e2e_test.go` with `TEST_TICKET_IDS` set; it calls the live ConnectWise API and is the user's to run.

Finish with a one-line pass/fail summary per step.
