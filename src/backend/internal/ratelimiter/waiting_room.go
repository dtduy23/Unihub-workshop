package ratelimiter

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed waiting_room.lua
var waitingRoomScript string

//go:embed waiting_room_promote.lua
var waitingRoomPromoteScript string

// QueueStatus represents the result of a waiting room check
type QueueStatus int

const (
	QueueGranted       QueueStatus = 1 // User can proceed immediately
	QueueWaiting       QueueStatus = 2 // User has been placed in queue
	QueueAlreadyActive QueueStatus = 3 // User already has an active token
	QueueAlreadyQueued QueueStatus = 0 // User is already in queue
)

func (s QueueStatus) String() string {
	switch s {
	case QueueGranted:
		return "GRANTED"
	case QueueWaiting:
		return "WAITING"
	case QueueAlreadyActive:
		return "ALREADY_ACTIVE"
	case QueueAlreadyQueued:
		return "ALREADY_QUEUED"
	default:
		return "UNKNOWN"
	}
}

// WaitingRoomResult holds the response from the waiting room
type WaitingRoomResult struct {
	Status       QueueStatus `json:"status"`
	StatusText   string      `json:"status_text"`
	Position     int         `json:"position,omitempty"`      // Queue position (1-indexed), 0 if granted
	TotalInQueue int         `json:"total_in_queue,omitempty"`
	RetryAfter   int         `json:"retry_after,omitempty"`   // Seconds until client should poll again
	AccessTTL    int         `json:"access_ttl,omitempty"`    // Seconds the access token is valid
	EstimatedWait int        `json:"estimated_wait,omitempty"` // Estimated seconds to wait
}

// WaitingRoom implements a virtual queue using Redis Sorted Sets
type WaitingRoom struct {
	client        *redis.Client
	enterScript   *redis.Script
	promoteScript *redis.Script
	maxActive     int           // Max concurrent users allowed through
	tokenTTL      int           // Seconds an access token is valid
	queueTTL      int           // Seconds before queue auto-expires
	throughput    int           // Users processed per promotion cycle
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
		throughput:     maxActive, // promote up to maxActive per cycle
	}
}

// Enter attempts to enter the waiting room for a specific workshop
// Returns the user's queue status and position
func (wr *WaitingRoom) Enter(ctx context.Context, workshopID, userID string) (*WaitingRoomResult, error) {
	queueKey := fmt.Sprintf("waitingroom:%s", workshopID)
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)
	now := float64(time.Now().UnixMilli()) / 1000.0

	result, err := wr.enterScript.Run(ctx, wr.client,
		[]string{queueKey, activeKey},
		userID, now, wr.maxActive, wr.tokenTTL, wr.queueTTL,
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
		res.RetryAfter = 5 // Poll every 5 seconds
		// Estimate wait: position * (tokenTTL / maxActive)
		res.EstimatedWait = secondVal * (wr.tokenTTL / wr.maxActive)
		if res.EstimatedWait < 5 {
			res.EstimatedWait = 5
		}
		log.Printf("[WAITING_ROOM] %s QUEUED at position %d/%d for workshop %s (est. %ds)",
			userID, secondVal, totalInQueue, workshopID, res.EstimatedWait)

	case QueueAlreadyActive:
		res.AccessTTL = secondVal
		res.RetryAfter = 0
		log.Printf("[WAITING_ROOM] %s already has active token for workshop %s", userID, workshopID)

	case QueueAlreadyQueued:
		res.Position = secondVal
		res.TotalInQueue = totalInQueue
		res.RetryAfter = 5
		res.EstimatedWait = secondVal * (wr.tokenTTL / wr.maxActive)
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
	return wr.client.SIsMember(ctx, activeKey, userID).Result()
}

// ReleaseAccess removes a user from the active set (called after successful registration)
func (wr *WaitingRoom) ReleaseAccess(ctx context.Context, workshopID, userID string) error {
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)
	return wr.client.SRem(ctx, activeKey, userID).Err()
}

// PromoteNext moves the next batch of users from the queue to the active set
// Should be called periodically by a background worker
func (wr *WaitingRoom) PromoteNext(ctx context.Context, workshopID string) (int, error) {
	queueKey := fmt.Sprintf("waitingroom:%s", workshopID)
	activeKey := fmt.Sprintf("waitingroom:active:%s", workshopID)

	promoted, err := wr.promoteScript.Run(ctx, wr.client,
		[]string{queueKey, activeKey},
		wr.maxActive, wr.tokenTTL,
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
	return wr.client.SCard(ctx, activeKey).Result()
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
