package metrics

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartDBCollector launches a background goroutine that periodically polls
// database pool statistics and updates Prometheus gauge metrics.
func StartDBCollector(pool *pgxpool.Pool, interval time.Duration) {
	if pool == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			stat := pool.Stat()
			DBActiveConns.Set(float64(stat.AcquiredConns()))
			DBIdleConns.Set(float64(stat.IdleConns()))
		}
	}()
}
