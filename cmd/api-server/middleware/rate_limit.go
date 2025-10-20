package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements per-user rate limiting
type RateLimiter struct {
	limiters          map[string]*rate.Limiter
	lastSeen          map[string]time.Time
	mu                sync.RWMutex
	requestsPerSecond rate.Limit
	burstSize         int
	cleanupInterval   time.Duration
}

// NewRateLimiter creates a new rate limiter with specified limits
func NewRateLimiter(requestsPerSecond int, burstSize int) *RateLimiter {
	rl := &RateLimiter{
		limiters:          make(map[string]*rate.Limiter),
		lastSeen:          make(map[string]time.Time),
		requestsPerSecond: rate.Limit(requestsPerSecond),
		burstSize:         burstSize,
		cleanupInterval:   5 * time.Minute,
	}
	go rl.cleanupStale()
	return rl
}

// getLimiter returns the rate limiter for a specific user
func (rl *RateLimiter) getLimiter(userID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rl.requestsPerSecond, rl.burstSize)
		rl.limiters[userID] = limiter
	}
	rl.lastSeen[userID] = time.Now()
	return limiter
}

// cleanupStale removes stale rate limiters to prevent memory leaks
func (rl *RateLimiter) cleanupStale() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for userID, lastSeen := range rl.lastSeen {
			if time.Since(lastSeen) > 10*time.Minute {
				delete(rl.limiters, userID)
				delete(rl.lastSeen, userID)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns the HTTP middleware handler
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"Unauthorized","code":"NO_USER_CONTEXT"}`, http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				return
			}

			limiter := rl.getLimiter(claims.UserID)
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Rate limit exceeded","code":"RATE_LIMIT_EXCEEDED"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
