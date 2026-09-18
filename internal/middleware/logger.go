package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusWriter records the status code written by the handler.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// RequestLog logs one line per request. The message carries the method and path so the log
// buffer can filter health-check and log-poll noise by message text.
func RequestLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			if sw.status == 0 {
				sw.status = http.StatusOK
			}
			logger.Info(fmt.Sprintf("%s %s %d", r.Method, r.URL.Path, sw.status),
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"latency", time.Since(start).String(),
				"remote", r.RemoteAddr,
			)
		})
	}
}

// Recover turns a handler panic into a 500 and logs the stack instead of killing the connection.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic in handler", "method", r.Method, "path", r.URL.Path,
						"panic", fmt.Sprint(rec), "stack", string(debug.Stack()))
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
