package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

// RequireConnectwiseSignature checks the webhook signature and restores the body for the handler.
//
// A failed check is logged but the request still proceeds. That matches the behaviour of the
// original gin middleware (which attached the error and ran the handler anyway); tightening it to
// a 401 is a deliberate policy change, not part of the framework port.
func RequireConnectwiseSignature() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeError(w, http.StatusBadRequest, "reading request body")
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			valid, err := psa.ValidateWebhook(r)
			if err != nil || !valid {
				slog.Warn("connectwise webhook signature check failed", "valid", valid, "error", errString(err))
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
