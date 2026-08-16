package middleware

import (
	"net/http"
	"sync"
	"time"

	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// tokenBucket implements a per-key token-bucket rate limiter using only stdlib.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	lastFill time.Time
	rps      float64
	burst    float64
}

func newBucket(rps, burst int) *tokenBucket {
	return &tokenBucket{
		tokens:   float64(burst),
		lastFill: time.Now(),
		rps:      float64(rps),
		burst:    float64(burst),
	}
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens += now.Sub(b.lastFill).Seconds() * b.rps
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

type limiterStore struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
	rps     int
	burst   int
}

func newLimiterStore(rps, burst int) *limiterStore {
	ls := &limiterStore{
		buckets: make(map[string]*tokenBucket),
		rps:     rps,
		burst:   burst,
	}
	go ls.evict()
	return ls
}

func (ls *limiterStore) allow(key string) bool {
	ls.mu.RLock()
	b, ok := ls.buckets[key]
	ls.mu.RUnlock()
	if !ok {
		ls.mu.Lock()
		b, ok = ls.buckets[key]
		if !ok {
			b = newBucket(ls.rps, ls.burst)
			ls.buckets[key] = b
		}
		ls.mu.Unlock()
	}
	return b.allow()
}

// evict removes idle buckets every 60 s to prevent unbounded map growth.
func (ls *limiterStore) evict() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		threshold := time.Now().Add(-5 * time.Minute)
		ls.mu.Lock()
		for k, b := range ls.buckets {
			b.mu.Lock()
			idle := b.lastFill.Before(threshold)
			b.mu.Unlock()
			if idle {
				delete(ls.buckets, k)
			}
		}
		ls.mu.Unlock()
	}
}

// NewRateLimiter returns a middleware that allows rps requests/second per key
// with a burst of burst. keyFn extracts the limiting key from the request.
// Returns HTTP 429 when the limit is exceeded, with a Retry-After header.
func NewRateLimiter(rps, burst int, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	store := newLimiterStore(rps, burst)
	retryAfterSecs := "1"
	if rps < 1 {
		retryAfterSecs = "5"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !store.allow(keyFn(r)) {
				w.Header().Set("Retry-After", retryAfterSecs)
				WriteError(w, apperrors.New(http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests — please slow down"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPKey extracts the client IP for rate-limiting. Honours X-Forwarded-For and
// X-Real-IP set by reverse proxies upstream of the API.
func IPKey(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return "ip:" + ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return "ip:" + ip
	}
	return "ip:" + r.RemoteAddr
}

// UserKey extracts the authenticated user ID for rate-limiting. Falls back to
// IPKey when the request has not yet been authenticated (e.g. /auth/login).
func UserKey(r *http.Request) string {
	if uid := UserID(r.Context()); uid != "" {
		return "u:" + uid
	}
	return IPKey(r)
}
