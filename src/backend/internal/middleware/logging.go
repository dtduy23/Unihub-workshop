package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// StructuredLogger replaces chi's default text logger with structured JSON output.
// Each request produces one JSON line with method, path, status, latency, IP, and user ID.
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := NewResponseWriterWrapper(w)
		next.ServeHTTP(ww, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.StatusCode,
			"latency_ms", time.Since(start).Milliseconds(),
			"remote_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"user_id", GetUserID(r.Context()),
			"size_bytes", ww.Written,
		)
	})
}
