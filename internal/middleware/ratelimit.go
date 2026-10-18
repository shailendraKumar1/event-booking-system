package middleware

import (
	"net/http"
	"sync"
	"time"
)

// TokenBucket implements a per-IP token bucket rate limiter.
type TokenBucket struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    int
	burst   int
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

func NewRateLimiter(rps, burst int) *TokenBucket {
	return &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    rps,
		burst:   burst,
	}
}

func (rl *TokenBucket) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: float64(rl.burst), lastCheck: time.Now()}
		rl.buckets[ip] = b
	}
	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.lastCheck = now
	b.tokens += elapsed * float64(rl.rate)
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimit middleware wraps the token bucket limiter.
func RateLimit(limiter *TokenBucket) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = fwd
			}
			if !limiter.Allow(ip) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
