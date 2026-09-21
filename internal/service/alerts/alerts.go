// Package alerts is where operator alerts go: intake failures, rate-cap blocks and staleness.
// Log is the default sink; the ops room sink replaces it once one is configured.
package alerts

import (
	"context"
	"log/slog"
)

// Alerter delivers one operator alert. Implementations must never block intake for long and
// must never return an error to the caller: an alert that cannot be delivered is logged.
type Alerter interface {
	Alert(ctx context.Context, subject, body string)
}

// Log writes alerts to the error log.
type Log struct{}

func (Log) Alert(_ context.Context, subject, body string) {
	slog.Error("alert: "+subject, "detail", body)
}
