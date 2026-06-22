# CLAUDE.md

Guidance for agents working in this repo. Keep this file updated when conventions or architecture change.

## What this is

`ticketbot` (`github.com/thecoretg/ticketbot`, Go 1.26) integrates **ConnectWise PSA** with **Cisco Webex**.
A CW webhook fires on ticket changes; the bot syncs ticket data into Postgres and routes notifications to Webex
rooms based on admin-configured rules. It also runs a **transformer pipeline** that can mutate the CW ticket
(summary, notes) before syncing. Admin UI is a vanilla-JS SPA served at `/panel`; everything else is a JSON REST API.

## Commands

- **Build:** `go build ./...` (binary: `CGO_ENABLED=0 go build -mod=vendor -o server .`)
- **Run locally (full stack):** `make docker-up` (re-vendors, builds image, starts Postgres + app on `:8080`).
  `make docker-down` stops and **wipes the DB volume** (`-v`).
- **Tests:** `go test ./...`. Only `internal/service/transformer` has unit tests today.
- **Regenerate DB code:** `make gensql` (runs `sqlc generate`). Required after any change to `queries/` or `migrations/`.
- **Vendor:** `make vendor` (`go mod tidy && go mod vendor`). Deps are **vendored** — `vendor/` is committed.

## Architecture

Dependencies are wired explicitly in `internal/server/server.go` `NewApp()` into a big `Services` struct +
`repos.AllRepos`. Layers, top to bottom:

1. **Routes** — `internal/server/routes.go` (Gin groups, `middleware.CombinedAuth` on everything but healthcheck).
2. **Handlers** — `internal/handlers/` (HTTP only; bind JSON, call service, use `outputJSON`/`*Error` helpers from `output.go`).
3. **Services** — `internal/service/<name>/` (business logic). Key ones: `cwsvc` (CW sync), `notifier` (Webex routing),
   `ticketbot` (orchestrates a webhook: lock → transform → sync → notify), `transformer`, `config`, `user`, `authsvc`, `syncsvc`.
4. **Repos** — interfaces in `internal/repos/`, Postgres impls in `internal/postgres/`. Every repo has `WithTx(pgx.Tx)`.
5. **DB** — sqlc-generated in `internal/db/` (`DBTX` works with both `*pgxpool.Pool` and `pgx.Tx`).
6. **Frontend** — `internal/web/static/` (`index.html` nav + `app.js` SPA + `style.css`), embedded via `embed.FS`.

CW/Webex client lives in the vendored `github.com/thecoretg/tctg-go` (`connectwise/psa`, `webex`).

## Adding a feature = one vertical slice

Mirror the `notifier_rule` or `transformer_rule` slice exactly:

1. **Migration** `migrations/000NN_*.sql` (goose `-- +goose Up/Down`, wrap in `StatementBegin/End`). Additive only.
2. **Bump `gooseMigrationVersion`** in `main.go` to the new number — the app migrates up/down to exactly this on boot.
3. **Queries** in `queries/*.sql` (sqlc annotations) → run `make gensql`.
4. **Model** in `models/`. **Repo interface** in `internal/repos/` + add to `AllRepos`. **Postgres impl** in
   `internal/postgres/` + add to the `AllRepos` literal in `internal/postgres/all.go`.
5. **Service**, **handler**, **route registration**, then **frontend** tab/modal in `app.js` (+ nav button in `index.html`).

Config flags live in the single-row `app_config` table (`models/app.go` `Config`/`ConfigUpdateParams`/`DefaultConfig`,
`queries/app_config.sql` `UpsertAppConfig`, `internal/postgres/appconfig.go`, `internal/service/config/service.go`).
`s.Cfg` is a shared `*models.Config` pointer updated live by the config service — flag changes take effect without restart.

## Conventions & gotchas

- **sqlc version pin:** committed generated files are from **sqlc v1.30.0**. A newer sqlc rewrites the `// versions:`
  comment in *every* `internal/db/*.sql.go` on `make gensql`, creating noisy diffs. Either install v1.30.0, or after
  `gensql` revert files whose only change is that comment (keep just the files you meant to touch).
- **Postgres only.** `pgx/v5` + sqlc. JSONB columns map to `[]byte`; nullable ints to `*int` (see `sqlc.yaml` overrides).
- **No server-side HTML templating.** The one use of `text/template` is the transformer's dynamic field rendering
  (`internal/service/transformer/template.go`) — not for HTML.
- **Tests need a real Postgres** for anything DB-touching; pure logic (e.g. transformer matching/templates) is unit-testable
  without one. There's no test DB harness checked in.
- **Required env vars** (validated in `internal/server/cfg.go`): `INITIAL_ADMIN_EMAIL`, `POSTGRES_DSN`, `WEBEX_SECRET`,
  `CW_PUB_KEY`, `CW_PRIV_KEY`, `CW_CLIENT_ID`, `CW_COMPANY_ID`, and `ROOT_URL` (unless `SKIP_HOOKS=true`).
  **Test/optional flags:** `SKIP_AUTH`, `SKIP_HOOKS`, `MOCK_WEBEX`, `STORE_TTL_SECONDS`, `API_KEY`,
  `SKIP_INITIAL_PASSWORD_RESET` (bootstrap admin with no forced first-login password change — fresh DB only),
  `INITIAL_ADMIN_PASSWORD` (defaults to `password`), `PORT`, `DEBUG`.
- **Auth:** bootstrap admin created on boot (`user.BootstrapAdmin`) if absent; forces a password reset unless
  `SKIP_INITIAL_PASSWORD_RESET=true`. Panel uses session JWTs; API uses keys. Optional TOTP.

## Transformer pipeline (recent feature)

`internal/service/transformer/` runs admin-configured rules that mutate the live CW ticket via the API **before**
`cwsvc.ProcessTicket` syncs it locally (so mutations flow into the local sync with no cwsvc changes). Invoked from
`ticketbot.Service.ProcessTicket`, guarded by the `attempt_transform` flag; a no-op when off or no rules match.
Failures are **non-fatal** (logged, never block sync/notify).

- **Extending actions:** implement the `Transformer` interface (`transformer.go`) and add it to `newRegistry()`.
  Current actions: `update_summary`, `add_note`. Field values are Go templates rendered against the `psa.Ticket`,
  validated at rule-save time.
- **Rule matching** (`run.go` `ruleApplies`): board + `apply_on` (new/updated/both) + a list of field `conditions`
  (allowlisted field/operator in `conditions.go`, all AND, case-insensitive).
- **Loop prevention is the #1 correctness concern** (mutating a ticket fires another webhook):
  - **Author gate uses `ticket.Info.UpdatedBy`, NOT the webhook `MemberID`.** CW callbacks always report the
    *callback-owner* member (the integration's API member), never the actual editor — so the webhook member is
    useless for "did the bot do this?". `updatedBy` reflects the real last editor. The bot identity is
    `app_config.cw_bot_member_identifier`.
  - `add_note` is non-idempotent → guarded by run-once markers (`transformer_run` table).
  - `update_summary` is idempotent only for **fixed-point** templates. A prefix template like
    `[{{.Company.Identifier}}] {{.Summary}}` re-prepends on every genuine human edit; if you need prefix behavior,
    add prefix-aware idempotency (skip if already prefixed).
