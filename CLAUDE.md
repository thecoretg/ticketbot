# CLAUDE.md

Guidance for agents working in this repo.

> **Keep this file current — required.** If you add or change anything a future Claude session would benefit from
> knowing (a new architectural pattern, a non-obvious gotcha, a convention, a new subsystem/feature, a build/auth
> quirk), update the relevant section of this file in the same change. Treat it as part of "done," not optional.

## What this is

`ticketbot` (`github.com/thecoretg/ticketbot`, Go 1.26) integrates **ConnectWise PSA** with **Cisco Webex**.
A CW webhook fires on ticket changes; the bot syncs ticket data into Postgres and routes notifications to Webex
rooms based on admin-configured rules. It also runs a **workflow engine** that can mutate the CW ticket and act on
it (update fields, add notes, notify, add resources/CC) before syncing, and records a per-ticket **audit journal**
of everything that happened. Credentials and key settings are managed in-app (env-overridable). Admin UI is a
vanilla-JS SPA served at `/panel`; everything else is a JSON REST API.

## Commands

- **Build:** `go build ./...` (binary: `CGO_ENABLED=0 go build -mod=vendor -o server .`)
- **Run locally (full stack):** `make docker-up` (re-vendors, builds image, starts Postgres + app on `:8080`).
  `make docker-down` stops and **wipes the DB volume** (`-v`).
- **Tests:** `go test ./...`. Unit tests live in `internal/service/{workflow,journal,ticketbot}` (pure logic; no DB needed).
- **Regenerate DB code:** `make gensql` (runs `sqlc generate`). Required after any change to `queries/` or `migrations/`.
- **Vendor:** `make vendor` (`go mod tidy && go mod vendor`). Deps are **vendored** — `vendor/` is committed.
- **Bumping `tctg-go`** (a **private** repo; `GOPRIVATE` already covers `github.com/thecoretg/*`): a plain
  `go get`/vendor fails if shell git can't auth to GitHub. If only `gh` is authenticated, point git at its token first:
  `git config --global url."https://x-access-token:$(gh auth token)@github.com/".insteadOf "https://github.com/"`,
  then `GOFLAGS=-mod=mod go get github.com/thecoretg/tctg-go@<ver> && make vendor`. **Unset that url rule afterward** —
  it persists your token in `~/.gitconfig`.

## Architecture

Dependencies are wired explicitly in `internal/server/server.go` `NewApp()` into a big `Services` struct +
`repos.AllRepos`. Layers, top to bottom:

1. **Routes** — `internal/server/routes.go` (Gin groups, `middleware.CombinedAuth` on everything but healthcheck).
2. **Handlers** — `internal/handlers/` (HTTP only; bind JSON, call service, use `outputJSON`/`*Error` helpers from `output.go`).
3. **Services** — `internal/service/<name>/` (business logic). Key ones: `cwsvc` (CW sync), `notifier` (Webex routing),
   `ticketbot` (orchestrates a webhook: lock → **workflow** → sync → notify → **journal**), `workflow`, `journal`,
   `config`, `user`, `authsvc`, `syncsvc`.
4. **Repos** — interfaces in `internal/repos/`, Postgres impls in `internal/postgres/`. Every repo has `WithTx(pgx.Tx)`.
5. **DB** — sqlc-generated in `internal/db/` (`DBTX` works with both `*pgxpool.Pool` and `pgx.Tx`).
6. **Frontend** — `internal/web/static/` (`index.html` nav + `app.js` SPA + `style.css`), embedded via `embed.FS`.
   The SPA is driven by a **hash router** (`router`/`navigate`/`parseRoute` in `app.js`): the URL hash is the source
   of truth so views are linkable and back/forward works. Routes are `#/<tab>[/<sub>…]` — e.g. `#/tickets`,
   `#/tickets/758099` (ticket detail), `#/notifier/forwards`. `tabLoaders[tab](parts)` renders each view; navigation
   goes through `navigate('/path')` (sets the hash → `hashchange` → `router`). Detail views (ticket journal, notifier
   subtab) live in the hash; modal-based edits (workflows/users/keys) are not routed. Hash routing needs **no server
   changes** (the hash isn't sent to the server; `/panel` always serves `index.html`).

CW/Webex client lives in the vendored `github.com/thecoretg/tctg-go` (`connectwise/psa`, `webex`).

## Adding a feature = one vertical slice

Mirror the `notifier_rule`, `workflow`, or `ticket_journal` slice exactly:

1. **Migration** `migrations/000NN_*.sql` (goose `-- +goose Up/Down`, wrap in `StatementBegin/End`). Additive only.
   The **2.0 baseline is a single consolidated `00001_init.sql`** (the whole schema, fresh — no incremental history);
   new migrations start at `00002`.
2. **Bump `gooseMigrationVersion`** in `main.go` to the new number — the app migrates up/down to exactly this on boot
   (baseline is `1`).
3. **Queries** in `queries/*.sql` (sqlc annotations) → run `make gensql`.
4. **Model** in `models/`. **Repo interface** in `internal/repos/` + add to `AllRepos`. **Postgres impl** in
   `internal/postgres/` + add to the `AllRepos` literal in `internal/postgres/all.go`.
5. **Service**, **handler**, **route registration**, then **frontend** tab/modal in `app.js` (+ nav button in `index.html`).

**CW reference entities** (board, status, type, subtype, item, company, contact, member) follow a narrower variant:
they live under `cwsvc` (no handler/route of their own), use the **CW id as the primary key**, and have an
`ensure<X>` method in `internal/service/cwsvc/ticket.go` (get-from-store → TTL check → fetch from CW → upsert) plus a
`Sync<X>` in `internal/service/syncsvc/`. **Board-scoped** entities (`status`, `type`, `subtype`, `item`) carry a
`board_id` FK, expose `ListByBoard`, and are synced per-board inside `SyncBoards`' loop (`syncsvc/boards.go`). Mirror
`cw_ticket_status` as the canonical board-scoped slice. Ticket type/subtype/item ensures run **before** `ensureTicket`
so the ticket's nullable FK columns resolve. Note: ConnectWise exposes subtype→type associations
(`BoardSubType.TypeAssociationIds`, stored as `type_association_ids` JSONB) but **no** item→subtype link, so items are
stored flat.

Config flags **and credentials** live in the single-row `app_config` table (`models/app.go` `Config`/`ConfigUpdateParams`/`DefaultConfig`,
`queries/app_config.sql` `UpsertAppConfig`, `internal/postgres/appconfig.go`, `internal/service/config/service.go`).
`s.Cfg` is a shared `*models.Config` pointer updated live by the config service — flag changes take effect without restart,
but **credential changes need a restart** (the CW/Webex clients are built once at boot from `s.Cfg`). Env vars take
precedence and are written back to the DB on boot (`server.mergeEnvConfig`); env-set fields are reported via
`env_locked` and locked in the Config tab; secrets (`cw_private_key`, `webex_secret`) are write-only over the API.

## Conventions & gotchas

- **sqlc version pin:** committed generated files are from **sqlc v1.31.1**. A different sqlc rewrites the `// versions:`
  comment in *every* `internal/db/*.sql.go` on `make gensql`, creating noisy diffs. Either match that version, or after
  `gensql` revert files whose only change is that comment (keep just the files you meant to touch).
- **Postgres only.** `pgx/v5` + sqlc. JSONB columns map to `[]byte`; nullable ints to `*int` (see `sqlc.yaml` overrides).
- **No server-side HTML templating.** The one use of `text/template` is the workflow engine's dynamic field rendering
  (`internal/service/workflow/template.go`) — not for HTML.
- **Tests need a real Postgres** for anything DB-touching; pure logic (workflow matching/templates, journal record/cap,
  run assembly) is unit-testable without one. Migration changes are verified by spinning a throwaway `postgres:16`
  container and running `goose -dir migrations postgres <dsn> up`/`down`. There's no test DB harness checked in.
- **Env vars** (`internal/server/cfg.go`):
  - **Bootstrap, always env-only** (chicken-and-egg): `INITIAL_ADMIN_EMAIL`, `POSTGRES_DSN`.
  - **Credentials & settings — env OR the Config tab** (env wins and is written back; see `mergeEnvConfig`):
    `ROOT_URL`, `CW_PUB_KEY`, `CW_PRIV_KEY`, `CW_CLIENT_ID`, `CW_COMPANY_ID`, `WEBEX_SECRET`,
    `CW_BOT_MEMBER_IDENTIFIER`, `ATTEMPT_WORKFLOW`, `ATTEMPT_NOTIFY`, `DEBUG_LOGGING`. First boot still needs the
    CW/Webex creds present somewhere (the clients reject empty config) — after that they can be dropped from env.
  - **Test/dev:** `SKIP_AUTH`, `SKIP_HOOKS`, `MOCK_WEBEX`, `STORE_TTL_SECONDS`, `API_KEY`,
    `SKIP_INITIAL_PASSWORD_RESET`, `INITIAL_ADMIN_PASSWORD`, `PORT`.
- **Auth:** bootstrap admin created on boot (`user.BootstrapAdmin`) if absent; forces a password reset unless
  `SKIP_INITIAL_PASSWORD_RESET=true`. Panel uses session JWTs; API uses keys. Optional TOTP.

## Workflows (was "transformer")

`internal/service/workflow/` runs admin-configured workflows that mutate/act on the live CW ticket via the API
**before** `cwsvc.ProcessTicket` syncs it locally (so mutations flow into the local sync with no cwsvc changes).
Invoked from `ticketbot.Service.ProcessTicket`, guarded by the `attempt_workflow` flag; a no-op when off or nothing
matches. Failures are **non-fatal** (logged, never block sync/notify).

- A **workflow** = a required board + `on_ticket_action` (new/updated/both) + an optional **nested boolean condition
  tree** + an **ordered list of actions**. The condition tree (`models.ConditionGroup`/`ConditionNode`/`Condition`)
  and actions (`models.Action`) are stored as JSONB columns on the `workflow` table.
- **Conditions** are evaluated in Go (`conditions.go`) against the fetched `psa.Ticket` + most-recent note. Fields
  include `last_note_text`/`last_note_sender`/`last_note_type`; operators include `is_any_of`/`is_none_of` (comma-token
  sets). Groups nest with AND/OR. Two field kinds: **string** fields (`conditionFields`, string operators) and
  **boolean** fields (`conditionBoolFields`, e.g. `customer_updated_flag`, matched with `is_true`/`is_false` and no
  value — rendered as an on/off toggle in the builder). Add a new bool field = one entry in `conditionBoolFields` +
  one in the UI's `WORKFLOW_FIELDS`/`WORKFLOW_BOOL_FIELDS`.
- **CW-backed condition pickers (UI only):** `status_name`/`type_name`/`subtype_name`/`company_name`/
  `company_identifier` are still plain string fields server-side, but the builder renders a **multi-select chips
  picker** (`CONDITION_PICKERS`/`attachMultiCombobox` in `app.js`) limited to `is_any_of`/`is_none_of` — selecting real
  CW items, storing the comma-joined names (or company identifier). Statuses use the live `/cw/boards/:id/statuses`
  endpoint; types/subtypes use `/cw/boards/:id/types`/`/subtypes` (local synced data, `cwsvc.ListBoardTypes/SubTypes`);
  companies use `/cw/companies`. No matching changes — reuses `tokensIntersect`. (Caveat: comma-delimited, so an item
  name containing a literal comma would tokenize wrong; CW status/type/subtype names don't.)
- **Actions:** implement `ActionHandler` and register in `newRegistry()`. Current: `ticket_update`, `add_note`,
  `send_message` (Webex; has `skip_further_notifications`), `skip_notifications`, `add_resource`, `add_email_cc`.
  Templated string fields carry the `tmpl:` tag, rendered against the ticket and validated at save time.
- `Run` returns `*RunResult{ SkipNotify, BotTriggered, Events }`; `Events` are human-readable journal lines.
- **Loop prevention is the #1 correctness concern** (the bot's own edit fires another webhook):
  - **Author gate uses `ticket.Info.UpdatedBy`, NOT the webhook `MemberID`.** CW callbacks always report the
    *callback-owner* member (the integration's API member), never the actual editor — so the webhook member is
    useless for "did the bot do this?". `updatedBy` reflects the real last editor; bot identity is
    `app_config.cw_bot_member_identifier`. The check runs at the **start** of `Run` (before this run's own actions
    edit the ticket) and is surfaced as `RunResult.BotTriggered`.
  - Non-idempotent actions (`add_note`, `send_message`) are guarded by run-once markers (`workflow_run`, keyed
    ticket+workflow+action_index). `ticket_update` is idempotent (no-ops when the field already matches).

## Simulation mode

Per-entity **dry-run** toggle to confirm workflows/notifications detect correctly without side effects. Each
**workflow**, **notifier_rule**, and **notifier_forward** has a `simulation_mode` BOOLEAN column. Processing
(`ProcessTicket`) runs end-to-end as normal — sync always uses real ticket data. Simulated outcomes are journaled as
**"Would …"** events (`JournalEvent.Simulated`, surfaced as a SIM badge in the Tickets UI).

- **Workflows:** a simulated workflow runs against a per-workflow copy of `Exec` with `Exec.Simulate=true` (set in
  `run.go`'s loop). Each `ActionHandler.Apply` checks `x.Simulate` *immediately before its mutating CW/Webex call* and
  returns the `Change` it would have made (now with `Change.Simulated`) without performing it or mutating the in-memory
  ticket. In sim, `runActions` **skips** writing run-once markers and **does not** propagate `SkipNotify` (sim must not
  change live behavior). `events.go` `actionEvent` renders "Would …" / skip status when `Change.Simulated`.
- **Notifier rules / forwards:** `recipData.simulated` marks recipients sourced from a simulated rule
  (`recipient.go`) or simulated forward (`forwards.go`), with **real-wins precedence** (a recipient reachable via any
  non-simulated path is sent for real; only real forwards suppress the source). Simulated recipients are **recorded as
  skipped `ticket_notification` rows but never sent** — so the existing `ExistsForNote` dedup stops a delayed re-fire
  once simulation is turned off (the explicit requirement). Events come from `requestEvents` ("Would notify …").
  Forward destinations inherit their source's simulated state (forwarding a simulated source is also simulated).
- **Authoritative for a board setting:** because a board only notifies when it has an enabled rule, simulating a rule
  authoritatively suppresses everything that rule governs. `getAllRecipients` decides per-event from `peopleGoverning`
  (the settings governing the ticket's people): the people are simulated unless one of those settings is real. So a
  board whose only/all relevant settings are simulated sends nothing — recipient *and* people.
- **CRUD:** toggling is via inline table switches → `PUT /notifiers/rules/:id` and `PUT /notifiers/forwards/:id`
  (forwards previously had no update path; `UpdateNotifierForward` query + repo `Update` were added). Workflows reuse
  the existing `PUT /workflows/:id`.

## Notifier rule = board setting (new-ticket recipient + notify-on-update)

A `notifier_rule` ("board setting") routes notifications for one board (`internal/service/notifier/recipient.go`
`getAllRecipients`):

- **New ticket** → the setting's configured **recipient** (`webex_recipient_id`, a room **or** person) **plus** the
  ticket's **people** (owner/resources, derived from the ticket — not the rule).
- **Updated ticket** → the ticket's **people only**, and only when at least one enabled setting for the board has
  `notify_on_update = true` (migration `00013`, default true to preserve prior behavior). The configured recipient is
  **never** notified on updates.
- A board with **no enabled rules notifies nobody** (early return in `processNotifications`) — the rule's existence is
  the board's notify on/off switch. The note-sender is always excluded.
- **new-vs-updated is the CW webhook action, not local-DB presence.** `ProcessTicket(…, added bool)` sets
  `isNew = added && !exists` where `added` is `webhook action == "added"`. A ticket the bot never synced can still be
  an *update* (its "added" hook was missed) — keying off DB presence alone would wrongly fire new-ticket routing on it.
  The `&& !exists` guard keeps a re-delivered "added" hook from re-firing new-ticket routing.

## Ticket journal (Tickets tab)

`internal/service/journal/` — one `ticket_journal` row per ticket; `ticketbot.ProcessTicket` appends a human-readable
**run** (timeline of `JournalEvent`s with `ok`/`error`/`skip`/`info` status) on every non-bot processing pass, capped at
50 runs/ticket, cleaned up after `log_retention_days`. It's the **audit source** — per-ticket/notification INFO slog
lines were demoted to DEBUG (`notifier`/`cwsvc` `logRequest`). Denormalized name columns drive the overview table
(`ListSummaries`, runs omitted); the detail view shows the full timeline.

- **Pure no-op runs are NOT journaled.** Connectwise fires many webhooks per change; most do nothing (note already
  notified, nothing matched). `recordJournal` drops any run whose outcome is `OutcomeNothingToDo`, so the tab only shows
  runs that did something real, errored, or simulated. (Snapshot columns therefore reflect the last *meaningful* run.)
- **Outcomes** (`buildRun`): `WithErrors` > `Completed` (any `ok` event) > **`Simulated`** (only simulation "Would …"
  events — kept, NOT a no-op, and not hidden by "Hide no-op runs") > `NothingToDo` (dropped).

- **Bot-triggered runs are NOT journaled.** The decision is `ticketbot.botTriggeredRun`: trust the workflow's
  pre-action `BotTriggered` when workflows ran; otherwise fall back to post-sync `updatedBy == bot`. Do **not** use the
  post-sync editor when workflows ran — this run's own `ticket_update`/`add_note` actions make the bot the last editor,
  which would falsely skip a legitimate run (this was a real bug).
