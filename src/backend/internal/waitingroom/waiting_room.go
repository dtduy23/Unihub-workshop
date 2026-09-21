package waitingroom

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed waiting_room.lua
var waitingRoomScript string

//go:embed waiting_room_promote.lua
var waitingRoomPromoteScript string

// WaitingRoom implements a virtual queue using Redis Sorted Sets
type WaitingRoom struct {
	client        *redis.Client
	enterScript   *redis.Script
	promoteScript *redis.Script
	maxActive     int // Max concurrent users allowed through
	tokenTTL      int // Seconds an access token is valid
	queueTTL      int // Seconds before queue auto-expires
	heartbeatTTL  int // Seconds before a waiting user is considered abandoned (60s)
	weight        int // Time penalty multiplier (e.g. 100)
	maxRandom     int // Max random score offset (e.g. 3000 -> 30s fair lottery)
	throughput    int // Users processed per promotion cycle
}

// NewWaitingRoom creates a new virtual waiting room
// maxActive: max concurrent users that can register at once (e.g., 100)
// tokenTTL: seconds a user has to complete their registration (e.g., 300 = 5 min)
// queueTTL: seconds before the entire queue expires (e.g., 3600 = 1 hour)
func NewWaitingRoom(client *redis.Client, maxActive, tokenTTL, queueTTL int) *WaitingRoom {
	return &WaitingRoom{
		client:        client,
		enterScript:   redis.NewScript(waitingRoomScript),
		promoteScript: redis.NewScript(waitingRoomPromoteScript),
		maxActive:     maxActive,
		tokenTTL:      tokenTTL,
		queueTTL:      queueTTL,
		heartbeatTTL:  60,   // 60 seconds expire as requested by user
		weight:        100,  // penalty multiplier per second
		maxRandom:     3000, // random offset up to 30s
		throughput:    maxActive,
	}
}

// SetAlgorithmParams allows customizing heartbeat TTL, weight and max random window
func (wr *WaitingRoom) SetAlgorithmParams(heartbeatTTL, weight, maxRandom int) {
	if heartbeatTTL > 0 {
		wr.heartbeatTTL = heartbeatTTL
	}
	if weight > 0 {
		wr.weight = weight
	}
	if maxRandom > 0 {
		wr.maxRandom = maxRandom
	}
}

// Enter attempts to enter the waiting room for a specific workshop
// Priority score formula: (now - openTime) * weight + randomVal
// Returns the user's queue status and position
func (wr *WaitingRoom) Enter(ctx context.Context, workshopID, userID string, openTime time.Time) (*WaitingRoomResult, error) {
	queueKey := fmt.Sprintf("waitingroom:%s", workshopID)
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)
	heartbeatKey := fmt.Sprintf("waitingroom:heartbeat:%s", workshopID)

	now := float64(time.Now().UnixMilli()) / 1000.0

	var openTimeUnix float64
	if !openTime.IsZero() {
		openTimeUnix = float64(openTime.Unix())
	} else {
		openTimeUnix = now
	}

	randomVal := 0
	if wr.maxRandom > 0 {
		randomVal = rand.IntN(wr.maxRandom)
	}

	result, err := wr.enterScript.Run(ctx, wr.client,
		[]string{queueKey, activeKey, heartbeatKey},
		userID, now, openTimeUnix, wr.maxActive, wr.tokenTTL, wr.heartbeatTTL, wr.weight, randomVal,
	).Int64Slice()

	if err != nil {
		return nil, fmt.Errorf("waiting room script error: %w", err)
	}

	if len(result) < 3 {
		return nil, fmt.Errorf("unexpected waiting room result length")
	}

	status := QueueStatus(result[0])
	secondVal := int(result[1])
	totalInQueue := int(result[2])

	res := &WaitingRoomResult{
		Status:       status,
		StatusText:   status.String(),
		TotalInQueue: totalInQueue,
	}

	switch status {
	case QueueGranted:
		res.AccessTTL = secondVal
		res.RetryAfter = 0
		log.Printf("[WAITING_ROOM] %s GRANTED access to workshop %s (TTL: %ds)", userID, workshopID, secondVal)

	case QueueWaiting:
		res.Position = secondVal
		res.RetryAfter = 5 // Poll every 5 seconds (also acts as heartbeat)
		if wr.maxActive > 0 {
			res.EstimatedWait = secondVal * (wr.tokenTTL / wr.maxActive)
		}
		if res.EstimatedWait < 5 {
			res.EstimatedWait = 5
		}
		log.Printf("[WAITING_ROOM] %s QUEUED at position %d/%d for workshop %s (est. %ds)",
			userID, secondVal, totalInQueue, workshopID, res.EstimatedWait)

	case QueueAlreadyActive:
		res.AccessTTL = secondVal
		res.RetryAfter = 0
		log.Printf("[WAITING_ROOM] %s already has active token for workshop %s (remaining: %ds)", userID, workshopID, secondVal)

	case QueueAlreadyQueued:
		res.Position = secondVal
		res.TotalInQueue = totalInQueue
		res.RetryAfter = 5
		if wr.maxActive > 0 {
			res.EstimatedWait = secondVal * (wr.tokenTTL / wr.maxActive)
		}
		if res.EstimatedWait < 5 {
			res.EstimatedWait = 5
		}
		log.Printf("[WAITING_ROOM] %s already in queue at position %d/%d for workshop %s",
			userID, secondVal, totalInQueue, workshopID)
	}

	return res, nil
}

// HasAccess checks if a user currently has an active access token for a workshop
func (wr *WaitingRoom) HasAccess(ctx context.Context, workshopID, userID string) (bool, error) {
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)
	now := float64(time.Now().UnixMilli()) / 1000.0

	score, err := wr.client.ZScore(ctx, activeKey, userID).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return score > now, nil
}

// ReleaseAccess removes a user from the active set (called after successful registration)
func (wr *WaitingRoom) ReleaseAccess(ctx context.Context, workshopID, userID string) error {
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)
	return wr.client.ZRem(ctx, activeKey, userID).Err()
}

// PromoteNext moves the next batch of users from the queue to the active set
// Should be called periodically by a background worker or lazily on poll
func (wr *WaitingRoom) PromoteNext(ctx context.Context, workshopID string) (int, error) {
	queueKey := fmt.Sprintf("waitingroom:%s", workshopID)
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)
	heartbeatKey := fmt.Sprintf("waitingroom:heartbeat:%s", workshopID)

	now := float64(time.Now().UnixMilli()) / 1000.0

	promoted, err := wr.promoteScript.Run(ctx, wr.client,
		[]string{queueKey, activeKey, heartbeatKey},
		wr.maxActive, wr.tokenTTL, now,
	).Int64()

	if err != nil {
		return 0, fmt.Errorf("promote script error: %w", err)
	}

	if promoted > 0 {
		log.Printf("[WAITING_ROOM] Promoted %d users for workshop %s", promoted, workshopID)
	}

	return int(promoted), nil
}

// GetQueueLength returns the current number of users waiting in queue
func (wr *WaitingRoom) GetQueueLength(ctx context.Context, workshopID string) (int64, error) {
	queueKey := fmt.Sprintf("waitingroom:%s", workshopID)
	return wr.client.ZCard(ctx, queueKey).Result()
}

// GetActiveCount returns the number of users currently with active access
func (wr *WaitingRoom) GetActiveCount(ctx context.Context, workshopID string) (int64, error) {
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)
	now := float64(time.Now().UnixMilli()) / 1000.0
	return wr.client.ZCount(ctx, activeKey, fmt.Sprintf("%f", now), "+inf").Result()
}

// StartPromotionWorker runs a background goroutine that periodically promotes
// users from the queue. Call this for each workshop that has an active queue.
func (wr *WaitingRoom) StartPromotionWorker(ctx context.Context, workshopID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[WAITING_ROOM] Promotion worker started for workshop %s (interval: %v)", workshopID, interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[WAITING_ROOM] Promotion worker stopped for workshop %s", workshopID)
			return
		case <-ticker.C:
			promoted, err := wr.PromoteNext(ctx, workshopID)
			if err != nil {
				log.Printf("[WAITING_ROOM] Promotion error for %s: %v", workshopID, err)
			}

			// If queue is empty, stop the worker
			queueLen, _ := wr.GetQueueLength(ctx, workshopID)
			if queueLen == 0 && promoted == 0 {
				log.Printf("[WAITING_ROOM] Queue empty for workshop %s, stopping promotion worker", workshopID)
				return
			}
		}
	}
}
