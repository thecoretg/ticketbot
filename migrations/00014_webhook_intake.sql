-- +goose Up
-- +goose StatementBegin
CREATE TABLE webhook_intake (
    id BIGSERIAL PRIMARY KEY,
    ticket_id INT NOT NULL,
    action TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);
CREATE INDEX idx_webhook_intake_open ON webhook_intake (ticket_id, id) WHERE status IN ('pending', 'processing');
CREATE INDEX idx_webhook_intake_status ON webhook_intake (status, id DESC);
CREATE INDEX idx_webhook_intake_received ON webhook_intake (received_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS webhook_intake;
-- +goose StatementEnd
