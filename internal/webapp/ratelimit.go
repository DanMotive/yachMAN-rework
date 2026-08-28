package webapp

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter provides a simple in-memory rate limiter per key.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*tokenBucket
	limit    int
	window   time.Duration
}

type tokenBucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a rate limiter with `limit` requests per `window`.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*tokenBucket),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.limiters[key]
	if !exists || now.Sub(bucket.lastReset) > rl.window {
		rl.limiters[key] = &tokenBucket{tokens: rl.limit - 1, lastReset: now}
		return true
	}
	if bucket.tokens <= 0 {
		return false
	}
	bucket.tokens--
	return true
}

// Middleware returns a chi-compatible middleware that rate limits by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			key = forwarded
		}
		if !rl.Allow(key) {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Cleanup runs periodically to remove expired buckets.
func (rl *RateLimiter) Cleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for k, b := range rl.limiters {
				if now.Sub(b.lastReset) > rl.window*2 {
					delete(rl.limiters, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
}
