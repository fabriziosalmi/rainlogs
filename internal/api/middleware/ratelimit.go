package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

const (
	ipLimiterTTL     = 10 * time.Minute
	ipLimiterCleanup = 5 * time.Minute
)

// ipEntry pairs a token-bucket limiter with the last-seen timestamp for eviction.
type ipEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// ipLimiter holds a token-bucket rate limiter per client IP with TTL-based eviction
// to prevent unbounded memory growth under high cardinality of client IPs.
type ipLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
	r       rate.Limit
	b       int
}

func newIPLimiter(r rate.Limit, b int) *ipLimiter {
	il := &ipLimiter{
		entries: make(map[string]*ipEntry),
		r:       r,
		b:       b,
	}
	go il.cleanupLoop()
	return il
}

func (i *ipLimiter) cleanupLoop() {
	ticker := time.NewTicker(ipLimiterCleanup)
	defer ticker.Stop()
	for range ticker.C {
		i.evict()
	}
}

func (i *ipLimiter) evict() {
	cutoff := time.Now().Add(-ipLimiterTTL)
	i.mu.Lock()
	defer i.mu.Unlock()
	for ip, e := range i.entries {
		if e.lastSeen.Before(cutoff) {
			delete(i.entries, ip)
		}
	}
}

func (i *ipLimiter) get(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	e, ok := i.entries[ip]
	if !ok {
		e = &ipEntry{lim: rate.NewLimiter(i.r, i.b), lastSeen: time.Now()}
		i.entries[ip] = e
	} else {
		e.lastSeen = time.Now()
	}
	return e.lim
}

// RateLimit returns a middleware that limits requests per IP.
// rps = requests per second, burst = burst capacity.
func RateLimit(rps float64, burst int) echo.MiddlewareFunc {
	limiter := newIPLimiter(rate.Limit(rps), burst)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !limiter.get(ip).Allow() {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			return next(c)
		}
	}
}
