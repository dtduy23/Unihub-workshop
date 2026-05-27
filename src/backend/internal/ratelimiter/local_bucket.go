package ratelimiter

import (
	"sync"
	"time"
)

// LocalBucket is an in-memory fallback rate limiter when Redis is unavailable
type LocalBucket struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	capacity   int
	refillRate float64
	maxKeys    int
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

func NewLocalBucket(capacity, refillRate, maxKeys int) *LocalBucket {
	lb := &LocalBucket{
		buckets:    make(map[string]*bucket),
		capacity:   capacity,
		refillRate: float64(refillRate),
		maxKeys:    maxKeys,
	}

	// Background cleanup goroutine (size-based eviction)
	go lb.cleanup()

	return lb
}

func (lb *LocalBucket) Allow(identifier string) (bool, int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()

	b, exists := lb.buckets[identifier]
	if !exists {
		// Evict if at capacity
		if len(lb.buckets) >= lb.maxKeys {
			lb.evictOldest()
		}
		b = &bucket{
			tokens:     float64(lb.capacity),
			lastRefill: now,
		}
		lb.buckets[identifier] = b
	}

	// Refill tokens
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * lb.refillRate
	if b.tokens > float64(lb.capacity) {
		b.tokens = float64(lb.capacity)
	}
	b.lastRefill = now

	// Try to consume
	if b.tokens >= 1 {
		b.tokens--
		return true, int(b.tokens)
	}

	retryAfter := int((1 - b.tokens) / lb.refillRate)
	if retryAfter < 1 {
		retryAfter = 1
	}
	return false, retryAfter
}

func (lb *LocalBucket) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range lb.buckets {
		if oldestKey == "" || v.lastRefill.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.lastRefill
		}
	}

	if oldestKey != "" {
		delete(lb.buckets, oldestKey)
	}
}

func (lb *LocalBucket) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		lb.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for k, v := range lb.buckets {
			if v.lastRefill.Before(cutoff) {
				delete(lb.buckets, k)
			}
		}
		lb.mu.Unlock()
	}
}
