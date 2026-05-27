package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"unihub-workshop/internal/ratelimiter"
)

// RateLimitMiddleware applies Token Bucket rate limiting with Redis + local fallback
type RateLimitMiddleware struct {
	redisBucket *ratelimiter.RedisTokenBucket
	localBucket *ratelimiter.LocalBucket
	redisTimeout time.Duration
}

func NewRateLimitMiddleware(
	redisBucket *ratelimiter.RedisTokenBucket,
	localBucket *ratelimiter.LocalBucket,
) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redisBucket:  redisBucket,
		localBucket:  localBucket,
		redisTimeout: 50 * time.Millisecond, // Circuit breaker threshold from spec
	}
}

func (rl *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract identifier from JWT context
		identifier := ""
		if uid, ok := r.Context().Value(UserIDKey).(string); ok && uid != "" {
			identifier = uid
		} else {
			// Fallback to IP for unauthenticated requests
			identifier = r.RemoteAddr
		}

		// Try Redis first with timeout
		ctx, cancel := context.WithTimeout(r.Context(), rl.redisTimeout)
		allowed, retryOrRemaining, err := rl.redisBucket.Allow(ctx, identifier)
		cancel()

		if err != nil {
			// Redis failed - fallback to local rate limiter
			log.Printf("[RATE_LIMIT] Redis failed, using local fallback: %v", err)
			allowed, retryOrRemaining = rl.localBucket.Allow(identifier)
		}

		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryOrRemaining))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Rate limit exceeded","message":"Too many requests. Please try again later."}`))
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", retryOrRemaining))

		next.ServeHTTP(w, r)
	})
}
