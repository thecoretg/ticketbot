---
name: migration
description: Create a new goose migration file, bump gooseMigrationVersion in main.go, and remind about sqlc regeneration. Use when a schema change is needed.
disable-model-invocation: true
---

Create a new database migration named `$ARGUMENTS` (snake_case; if empty, ask for a name).

1. Find the highest existing number: `ls migrations/*.sql | sort | tail -1`. New number = that + 1, zero-padded to 5 digits.
2. Create `migrations/000NN_$ARGUMENTS.sql` with this skeleton, then fill in the Up and a working Down:

   ```sql
   -- +goose Up
   -- +goose StatementBegin

   -- +goose StatementEnd

   -- +goose Down
   -- +goose StatementBegin

   -- +goose StatementEnd
   ```

3. Bump `gooseMigrationVersion` in `main.go` to the new number. Without this the migration never runs.
4. If the migration changes tables that `queries/*.sql` touch, update the queries and run `make gensql`.
5. Verify against the local DB (`make db-up` if needed):

   ```bash
   goose -dir migrations postgres "host=localhost port=5433 dbname=testdb user=testuser password=testpass sslmode=disable" up
   goose -dir migrations postgres "host=localhost port=5433 dbname=testdb user=testuser password=testpass sslmode=disable" down
   goose -dir migrations postgres "host=localhost port=5433 dbname=testdb user=testuser password=testpass sslmode=disable" up
   ```

   Then `go build ./...` and, if repos changed, `TEST_POSTGRES_DSN="postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable" go test ./internal/postgres/`.

Report the file created, the new version number, and whether Down was exercised.
