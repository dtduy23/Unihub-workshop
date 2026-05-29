package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"unihub-workshop/internal/metrics"
)

// ResponseWriterWrapper intercepts ResponseWriter to capture status code and bytes written
type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
	Written    int64
}

// NewResponseWriterWrapper creates a wrapper with default 200 status
func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}
}

// WriteHeader captures status code
func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures bytes written
func (rw *ResponseWriterWrapper) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.Written += int64(n)
	return n, err
}

// MetricsMiddleware records request count, latency histogram, and in-flight gauge
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		metrics.RequestsInFlight.Inc()
		defer metrics.RequestsInFlight.Dec()

		ww := NewResponseWriterWrapper(w)
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(ww.StatusCode)

		// Use Chi route pattern to normalize paths (e.g. "/api/v1/workshops/{id}")
		path := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			path = rctx.RoutePattern()
		}

		metrics.RequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		metrics.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}
