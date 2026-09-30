package presence

import (
	"context"
	"fmt"
	"time"

	"unihub-workshop/internal/metrics"

	"github.com/redis/go-redis/v9"
)

// PresenceTracker tracks unique visitors and active users using Redis HyperLogLog.
// HyperLogLog provides O(1) time complexity and fixed memory (~12KB) per key,
// allowing ultra-fast estimation of millions of concurrent unique visitors.
type PresenceTracker struct {
	redis         *redis.Client
	windowMinutes int
}

// NewPresenceTracker creates a new PresenceTracker instance.
// windowMinutes defines the sliding time window (e.g., 3 minutes) used to count active visitors.
func NewPresenceTracker(redisClient *redis.Client, windowMinutes int) *PresenceTracker {
	if windowMinutes <= 0 {
		windowMinutes = 3
	}
	return &PresenceTracker{
		redis:         redisClient,
		windowMinutes: windowMinutes,
	}
}

// RecordPresence records a client's presence into the sliding-window HyperLogLog keys.
// clientID can be a UserID (if authenticated) or a Remote IP address.
// workshopID is optional (can be empty string if viewing general platform pages).
func (pt *PresenceTracker) RecordPresence(ctx context.Context, clientID string, workshopID string) error {
	if pt.redis == nil || clientID == "" {
		return nil
	}

	epochMinute := time.Now().Unix() / 60
	ttl := time.Duration(pt.windowMinutes+2) * time.Minute

	pipe := pt.redis.Pipeline()

	// 1. Global presence across the whole platform
	globalKey := fmt.Sprintf("presence:global:%d", epochMinute)
	pipe.PFAdd(ctx, globalKey, clientID)
	pipe.Expire(ctx, globalKey, ttl)

	// 2. Specific workshop presence (if viewing workshop details, countdown, or waiting room)
	if workshopID != "" {
		wsKey := fmt.Sprintf("presence:ws:%s:%d", workshopID, epochMinute)
		pipe.PFAdd(ctx, wsKey, clientID)
		pipe.Expire(ctx, wsKey, ttl)

		// Record workshopID in the active workshops set
		pipe.SAdd(ctx, "presence:active_workshops", workshopID)
		pipe.Expire(ctx, "presence:active_workshops", ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// GetGlobalActiveUsers returns the estimated count of unique active users/IPs across
// all minute buckets within the sliding window using Redis PFCOUNT union.
func (pt *PresenceTracker) GetGlobalActiveUsers(ctx context.Context) (int64, error) {
	if pt.redis == nil {
		return 0, nil
	}

	nowMinute := time.Now().Unix() / 60
	keys := make([]string, 0, pt.windowMinutes)
	for i := 0; i < pt.windowMinutes; i++ {
		keys = append(keys, fmt.Sprintf("presence:global:%d", nowMinute-int64(i)))
	}

	return pt.redis.PFCount(ctx, keys...).Result()
}

// GetWorkshopActiveUsers returns the estimated count of unique active visitors/IPs
// for a specific workshop within the sliding window.
func (pt *PresenceTracker) GetWorkshopActiveUsers(ctx context.Context, workshopID string) (int64, error) {
	if pt.redis == nil || workshopID == "" {
		return 0, nil
	}

	nowMinute := time.Now().Unix() / 60
	keys := make([]string, 0, pt.windowMinutes)
	for i := 0; i < pt.windowMinutes; i++ {
		keys = append(keys, fmt.Sprintf("presence:ws:%s:%d", workshopID, nowMinute-int64(i)))
	}

	return pt.redis.PFCount(ctx, keys...).Result()
}

// GetActiveWorkshops returns list of workshop IDs that recently had visitor presence.
func (pt *PresenceTracker) GetActiveWorkshops(ctx context.Context) ([]string, error) {
	if pt.redis == nil {
		return nil, nil
	}
	return pt.redis.SMembers(ctx, "presence:active_workshops").Result()
}

// StartCollector starts a background goroutine that polls active users from Redis
// and exports them as Prometheus metrics (unihub_active_online_users).
func (pt *PresenceTracker) StartCollector(ctx context.Context, interval time.Duration) {
	if pt.redis == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Update global active users gauge
				if count, err := pt.GetGlobalActiveUsers(ctx); err == nil {
					metrics.ActiveOnlineUsers.Set(float64(count))
				}

				// Update per-workshop active users gauge
				if workshops, err := pt.GetActiveWorkshops(ctx); err == nil {
					for _, wID := range workshops {
						if wCount, err := pt.GetWorkshopActiveUsers(ctx, wID); err == nil {
							metrics.WorkshopActiveUsers.WithLabelValues(wID).Set(float64(wCount))
						}
					}
				}
			}
		}
	}()
}
