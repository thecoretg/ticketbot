package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

// RequireConnectwiseSignature checks the webhook signature and restores the body for the handler.
// A failed check is a 401, so a forged callback cannot start a workflow run.
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
				slog.Warn("connectwise webhook rejected: signature check failed", "valid", valid, "error", errString(err), "remote", r.RemoteAddr)
				writeError(w, http.StatusUnauthorized, "invalid webhook signature")
				return
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
