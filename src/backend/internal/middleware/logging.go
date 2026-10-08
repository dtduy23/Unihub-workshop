package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type requestLogKey struct{}
type requestLogFields struct {
	correlationID string
	userID        string
	studentID     string
}

// SetLogCorrelationID connects the enqueue request log to worker message logs.
func SetLogCorrelationID(r *http.Request, id string) {
	if fields, ok := r.Context().Value(requestLogKey{}).(*requestLogFields); ok {
		fields.correlationID = id
	}
}

func setLogIdentity(r *http.Request, userID, studentID string) {
	if fields, ok := r.Context().Value(requestLogKey{}).(*requestLogFields); ok {
		fields.userID, fields.studentID = userID, studentID
	}
}

// StructuredLogger replaces chi's default text logger with structured JSON output.
// Each request produces one JSON line with method, path, status, latency, IP, and user ID.
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fields := &requestLogFields{correlationID: middleware.GetReqID(r.Context())}
		r = r.WithContext(context.WithValue(r.Context(), requestLogKey{}, fields))
		w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))

		ww := NewResponseWriterWrapper(w)
		next.ServeHTTP(ww, r)

		slog.Info("http request",
			"request_id", middleware.GetReqID(r.Context()),
			"correlation_id", fields.correlationID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.StatusCode,
			"latency_ms", time.Since(start).Milliseconds(),
			"duration", time.Since(start).Seconds(),
			"remote_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"user_id", fields.userID,
			"student_id", fields.studentID,
			"size_bytes", ww.Written,
		)
	})
}
