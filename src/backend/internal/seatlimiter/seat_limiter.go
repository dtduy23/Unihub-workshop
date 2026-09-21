package seatlimiter

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

//go:embed seat_cache.lua
var seatCacheScript string

// SeatLimiter manages Redis-based seat reservation to shield database from overselling
type SeatLimiter struct {
	client *redis.Client
	script *redis.Script
}

func NewSeatLimiter(client *redis.Client) *SeatLimiter {
	return &SeatLimiter{
		client: client,
		script: redis.NewScript(seatCacheScript),
	}
}

// PrepareCache ensures the seat count is in Redis. If not, it sets it.
func (sl *SeatLimiter) PrepareCache(ctx context.Context, workshopID string, initialSeats int) error {
	key := fmt.Sprintf("workshop:seats:%s", workshopID)
	// We use SETNX to avoid overwriting if another process already set it
	return sl.client.SetNX(ctx, key, initialSeats, 0).Err()
}

// TryAcquireSeat attempts to decrement the seat count in Redis
func (sl *SeatLimiter) TryAcquireSeat(ctx context.Context, workshopID string) (bool, error) {
	key := fmt.Sprintf("workshop:seats:%s", workshopID)

	result, err := sl.script.Run(ctx, sl.client, []string{key}, 1).Int64()
	if err != nil {
		return false, fmt.Errorf("redis seat script error: %w", err)
	}

	// -1: Key not found, -2: Out of seats
	if result == -1 {
		return false, fmt.Errorf("seat cache not initialized for workshop %s", workshopID)
	}

	if result == -2 {
		return false, nil // Valid result: no seats left
	}

	return true, nil // Success: seat acquired in Redis
}

// ReleaseSeat increments the seat count back (e.g., if registration failed later)
func (sl *SeatLimiter) ReleaseSeat(ctx context.Context, workshopID string) error {
	key := fmt.Sprintf("workshop:seats:%s", workshopID)
	return sl.client.Incr(ctx, key).Err()
}

// GetAvailableSeats returns the current count from Redis
func (sl *SeatLimiter) GetAvailableSeats(ctx context.Context, workshopID string) (int, error) {
	key := fmt.Sprintf("workshop:seats:%s", workshopID)
	val, err := sl.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}
