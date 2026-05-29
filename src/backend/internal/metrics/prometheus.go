package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// --- HTTP API Metrics ---

	// RequestsTotal counts total HTTP requests processed
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by status code, method and path.",
		},
		[]string{"method", "path", "status"},
	)

	// RequestDuration measures latency of HTTP requests (for P50/P95/P99 calculation)
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// RequestsInFlight tracks active concurrent HTTP requests
	RequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being actively processed.",
		},
	)

	// --- Database Metrics ---

	// DBActiveConns tracks active connections in pgx pool
	DBActiveConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_active_connections",
			Help: "Current number of active connections acquired in pgx database pool.",
		},
	)

	// DBIdleConns tracks idle connections in pgx pool
	DBIdleConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Current number of idle connections in pgx database pool.",
		},
	)

	// --- Redis Cache Metrics ---

	// CacheHits counts successful Redis lookups
	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_hits_total",
			Help: "Total number of successful cache lookups.",
		},
	)

	// CacheMisses counts failed Redis lookups
	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_misses_total",
			Help: "Total number of cache misses.",
		},
	)

	// --- RabbitMQ Message Metrics ---

	// MessagesPublished counts messages published to RabbitMQ
	MessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_messages_published_total",
			Help: "Total number of messages published to RabbitMQ queues.",
		},
		[]string{"queue"},
	)

	// MessagesConsumed counts messages processed by background workers
	MessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_messages_consumed_total",
			Help: "Total number of RabbitMQ messages consumed by workers, partitioned by status.",
		},
		[]string{"queue", "status"},
	)
)
