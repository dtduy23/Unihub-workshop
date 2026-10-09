package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"unihub-workshop/internal/metrics"
)

func responseStatus(w chimw.WrapResponseWriter) int {
	if status := w.Status(); status != 0 {
		return status
	}
	// net/http sends 200 when a handler returns without an explicit header.
	return http.StatusOK
}

// MetricsMiddleware records request count, latency histogram, and in-flight gauge
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		metrics.RequestsInFlight.Inc()
		defer metrics.RequestsInFlight.Dec()

		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(responseStatus(ww))

		// Use Chi route pattern to normalize paths (e.g. "/api/v1/workshops/{id}")
		path := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			path = rctx.RoutePattern()
		}

		metrics.RequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		metrics.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}
