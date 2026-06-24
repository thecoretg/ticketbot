-- +goose Up
-- +goose StatementBegin
-- ticketbot 2.0 baseline schema. This is a fresh, consolidated definition of the
-- entire database — it replaces the incremental 0001–0013 migration history. New
-- changes from here on are additive migrations (0002+), bumping gooseMigrationVersion.

-- Single-row application config (id is always 1).
CREATE TABLE IF NOT EXISTS app_config (
    id                         INT     PRIMARY KEY DEFAULT 1,
    attempt_notify             BOOLEAN NOT NULL DEFAULT false,
    max_message_length         INT     NOT NULL DEFAULT 300,
    max_concurrent_syncs       INT     NOT NULL DEFAULT 5,
    require_totp               BOOLEAN NOT NULL DEFAULT false,
    debug_logging              BOOLEAN NOT NULL DEFAULT false,
    log_retention_days         INT     NOT NULL DEFAULT 7,
    log_cleanup_interval_hours INT     NOT NULL DEFAULT 24,
    log_buffer_size            INT     NOT NULL DEFAULT 500,
    attempt_workflow           BOOLEAN NOT NULL DEFAULT false,
    cw_bot_member_identifier   TEXT    NOT NULL DEFAULT '',
    root_url                   TEXT    NOT NULL DEFAULT '',
    cw_company_id              TEXT    NOT NULL DEFAULT '',
    cw_client_id               TEXT    NOT NULL DEFAULT '',
    cw_public_key              TEXT    NOT NULL DEFAULT '',
    cw_private_key             TEXT    NOT NULL DEFAULT '',
    webex_secret               TEXT    NOT NULL DEFAULT ''
);

-- Auth: users, API keys, web sessions, TOTP.
CREATE TABLE IF NOT EXISTS api_user (
    id                      SERIAL PRIMARY KEY,
    email_address           TEXT UNIQUE NOT NULL,
    created_on              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    password_hash           BYTEA,
    password_reset_required BOOLEAN NOT NULL DEFAULT false,
    totp_secret             TEXT,
    totp_enabled            BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS api_key (
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    key_hash   BYTEA NOT NULL,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    key_hint   TEXT,
    UNIQUE(user_id, key_hash)
);

CREATE TABLE IF NOT EXISTS session (
    id         SERIAL PRIMARY KEY,
    user_id    INT         NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    token_hash BYTEA       NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_on TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS totp_pending (
    id         SERIAL PRIMARY KEY,
    user_id    INT         NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    token_hash BYTEA       NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_on TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS totp_recovery_code (
    id         SERIAL PRIMARY KEY,
    user_id    INT       NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    code_hash  BYTEA     NOT NULL,
    used       BOOLEAN   NOT NULL DEFAULT false,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Application logs (buffered + persisted).
CREATE TABLE IF NOT EXISTS app_log (
    id         SERIAL      PRIMARY KEY,
    time       TIMESTAMPTZ NOT NULL,
    level      TEXT        NOT NULL,
    message    TEXT        NOT NULL,
    attrs      JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_app_log_time ON app_log(time);

-- Webex recipients (rooms + people).
CREATE TABLE IF NOT EXISTS webex_recipient (
    id            SERIAL PRIMARY KEY,
    webex_id      TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    email         TEXT,
    type          TEXT NOT NULL,
    last_activity TIMESTAMP NOT NULL,
    created_on    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── ConnectWise reference entities (CW id is the primary key) ──────────────────
CREATE TABLE IF NOT EXISTS cw_board (
    id         INT PRIMARY KEY,
    name       TEXT NOT NULL,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted    BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_company (
    id         INT PRIMARY KEY,
    name       TEXT NOT NULL,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted    BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_contact (
    id         INT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name  TEXT,
    company_id INT REFERENCES cw_company(id),
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted    BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_member (
    id            INT PRIMARY KEY,
    identifier    TEXT NOT NULL,
    first_name    TEXT NOT NULL,
    last_name     TEXT NOT NULL,
    primary_email TEXT NOT NULL,
    updated_on    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted       BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_ticket_status (
    id               INT PRIMARY KEY,
    board_id         INT NOT NULL REFERENCES cw_board(id),
    name             TEXT NOT NULL,
    default_status   BOOLEAN NOT NULL,
    display_on_board BOOLEAN NOT NULL,
    inactive         BOOLEAN NOT NULL,
    closed           BOOLEAN NOT NULL,
    added_on         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted          BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_ticket_type (
    id           INT PRIMARY KEY,
    board_id     INT NOT NULL REFERENCES cw_board(id),
    name         TEXT NOT NULL,
    category     TEXT NOT NULL DEFAULT '',
    default_flag BOOLEAN NOT NULL DEFAULT false,
    inactive     BOOLEAN NOT NULL DEFAULT false,
    added_on     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_ticket_subtype (
    id                   INT PRIMARY KEY,
    board_id             INT NOT NULL REFERENCES cw_board(id),
    name                 TEXT NOT NULL,
    inactive             BOOLEAN NOT NULL DEFAULT false,
    type_association_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    added_on             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted              BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_ticket_item (
    id         INT PRIMARY KEY,
    board_id   INT NOT NULL REFERENCES cw_board(id),
    name       TEXT NOT NULL,
    inactive   BOOLEAN NOT NULL DEFAULT false,
    added_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted    BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_ticket (
    id          INT PRIMARY KEY,
    summary     TEXT NOT NULL,
    board_id    INT REFERENCES cw_board(id) NOT NULL,
    status_id   INT REFERENCES cw_ticket_status(id) NOT NULL,
    owner_id    INT REFERENCES cw_member(id),
    company_id  INT REFERENCES cw_company(id) NOT NULL,
    contact_id  INT REFERENCES cw_contact(id),
    resources   TEXT,
    updated_by  TEXT,
    updated_on  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted     BOOLEAN NOT NULL DEFAULT false,
    type_id     INT REFERENCES cw_ticket_type(id),
    subtype_id  INT REFERENCES cw_ticket_subtype(id),
    item_id     INT REFERENCES cw_ticket_item(id)
);

CREATE TABLE IF NOT EXISTS cw_ticket_note (
    id         INT PRIMARY KEY,
    ticket_id  INT NOT NULL REFERENCES cw_ticket(id) ON DELETE CASCADE,
    member_id  INT REFERENCES cw_member(id),
    contact_id INT REFERENCES cw_contact(id),
    content    TEXT,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted    BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_cw_ticket_type_board    ON cw_ticket_type(board_id);
CREATE INDEX idx_cw_ticket_subtype_board ON cw_ticket_subtype(board_id);
CREATE INDEX idx_cw_ticket_item_board    ON cw_ticket_item(board_id);

-- ── Notifier (board settings + forwards) ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS notifier_rule (
    id                 SERIAL PRIMARY KEY,
    cw_board_id        INT NOT NULL REFERENCES cw_board(id) ON DELETE CASCADE,
    webex_recipient_id INT NOT NULL REFERENCES webex_recipient(id) ON DELETE CASCADE,
    notify_enabled     BOOLEAN NOT NULL DEFAULT true,
    created_on         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    simulation_mode    BOOLEAN NOT NULL DEFAULT false,
    notify_on_update   BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (cw_board_id, webex_recipient_id)
);

CREATE TABLE IF NOT EXISTS notifier_forward (
    id              SERIAL PRIMARY KEY,
    source_id       INT NOT NULL REFERENCES webex_recipient(id),
    destination_id  INT NOT NULL REFERENCES webex_recipient(id),
    start_date      TIMESTAMP,
    end_date        TIMESTAMP,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    user_keeps_copy BOOLEAN NOT NULL DEFAULT true,
    created_on      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    simulation_mode BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (source_id, destination_id, start_date, end_date),
    CHECK (source_id <> destination_id),
    CHECK (start_date < end_date)
);

CREATE TABLE IF NOT EXISTS ticket_notification (
    id                SERIAL PRIMARY KEY,
    ticket_id         INT NOT NULL REFERENCES cw_ticket(id) ON DELETE CASCADE,
    ticket_note_id    INT REFERENCES cw_ticket_note(id) ON DELETE CASCADE,
    recipient_id      INT REFERENCES webex_recipient(id) ON DELETE CASCADE,
    forwarded_from_id INT REFERENCES webex_recipient(id) ON DELETE CASCADE,
    sent              BOOLEAN NOT NULL DEFAULT false,
    skipped           BOOLEAN NOT NULL DEFAULT true,
    created_on        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── Per-ticket lifecycle journal (Tickets tab audit source) ───────────────────
CREATE TABLE IF NOT EXISTS ticket_journal (
    ticket_id    INT PRIMARY KEY,
    summary      TEXT        NOT NULL DEFAULT '',
    board_name   TEXT        NOT NULL DEFAULT '',
    company_name TEXT        NOT NULL DEFAULT '',
    contact_name TEXT        NOT NULL DEFAULT '',
    status_name  TEXT        NOT NULL DEFAULT '',
    owner_name   TEXT        NOT NULL DEFAULT '',
    last_trigger TEXT        NOT NULL DEFAULT '',
    last_outcome TEXT        NOT NULL DEFAULT '',
    had_error    BOOLEAN     NOT NULL DEFAULT false,
    first_seen   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_run     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    runs         JSONB       NOT NULL DEFAULT '[]'::jsonb,
    type_name    TEXT        NOT NULL DEFAULT '',
    subtype_name TEXT        NOT NULL DEFAULT '',
    item_name    TEXT        NOT NULL DEFAULT ''
);
CREATE INDEX idx_ticket_journal_last_run ON ticket_journal(last_run DESC);

-- ── Workflows (board-scoped automations + run-once markers) ────────────────────
CREATE TABLE IF NOT EXISTS workflow (
    id               SERIAL PRIMARY KEY,
    name             TEXT    NOT NULL,
    cw_board_id      INT     NOT NULL REFERENCES cw_board(id) ON DELETE CASCADE,
    on_ticket_action TEXT    NOT NULL DEFAULT 'both',
    conditions       JSONB   NOT NULL DEFAULT '{}'::jsonb,
    actions          JSONB   NOT NULL DEFAULT '[]'::jsonb,
    priority         INT     NOT NULL DEFAULT 100,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    created_on       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    simulation_mode  BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_workflow_board ON workflow(cw_board_id);

CREATE TABLE IF NOT EXISTS workflow_run (
    ticket_id    INT NOT NULL,
    workflow_id  INT NOT NULL REFERENCES workflow(id) ON DELETE CASCADE,
    action_index INT NOT NULL,
    created_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (ticket_id, workflow_id, action_index)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS workflow_run;
DROP TABLE IF EXISTS workflow;
DROP TABLE IF EXISTS ticket_journal;
DROP TABLE IF EXISTS ticket_notification;
DROP TABLE IF EXISTS notifier_forward;
DROP TABLE IF EXISTS notifier_rule;
DROP TABLE IF EXISTS cw_ticket_note;
DROP TABLE IF EXISTS cw_ticket;
DROP TABLE IF EXISTS cw_ticket_item;
DROP TABLE IF EXISTS cw_ticket_subtype;
DROP TABLE IF EXISTS cw_ticket_type;
DROP TABLE IF EXISTS cw_ticket_status;
DROP TABLE IF EXISTS cw_member;
DROP TABLE IF EXISTS cw_contact;
DROP TABLE IF EXISTS cw_company;
DROP TABLE IF EXISTS cw_board;
DROP TABLE IF EXISTS webex_recipient;
DROP TABLE IF EXISTS app_log;
DROP TABLE IF EXISTS totp_recovery_code;
DROP TABLE IF EXISTS totp_pending;
DROP TABLE IF EXISTS session;
DROP TABLE IF EXISTS api_key;
DROP TABLE IF EXISTS api_user;
DROP TABLE IF EXISTS app_config;
-- +goose StatementEnd
