package ratelimiter

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed token_bucket.lua
var tokenBucketScript string

// RedisTokenBucket implements Token Bucket rate limiting using Redis Lua scripts
type RedisTokenBucket struct {
	client     *redis.Client
	script     *redis.Script
	capacity   int
	refillRate int
	ttl        int
}

func NewRedisTokenBucket(client *redis.Client, capacity, refillRate, ttl int) *RedisTokenBucket {
	return &RedisTokenBucket{
		client:     client,
		script:     redis.NewScript(tokenBucketScript),
		capacity:   capacity,
		refillRate: refillRate,
		ttl:        ttl,
	}
}

// Allow checks if a request is allowed for the given identifier
// Returns (allowed bool, retryAfter int seconds, err error)
func (tb *RedisTokenBucket) Allow(ctx context.Context, identifier string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:%s", identifier)
	now := float64(time.Now().UnixMilli()) / 1000.0

	result, err := tb.script.Run(ctx, tb.client, []string{key},
		tb.capacity, tb.refillRate, tb.ttl, now,
	).Int64Slice()

	if err != nil {
		return false, 0, fmt.Errorf("rate limit script error: %w", err)
	}

	if len(result) < 2 {
		return false, 0, fmt.Errorf("unexpected rate limit result")
	}

	allowed := result[0] == 1
	secondVal := int(result[1])

	if allowed {
		log.Printf("[RATE_LIMIT] Allowed %s, remaining tokens: %d", identifier, secondVal)
	} else {
		log.Printf("[RATE_LIMIT] Denied %s, retry after: %ds", identifier, secondVal)
	}

	return allowed, secondVal, nil
}
