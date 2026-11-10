package http

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter membatasi jumlah request per IP dengan algoritma token bucket dan pembersihan entri kedaluwarsa.
type RateLimiter struct {
	ratePerSec float64
	burst      float64
	ttl        time.Duration
	nowFn      func() time.Time

	mu      sync.Mutex
	buckets map[string]*rateBucket
	calls   int
}

func NewRateLimiter(perMinute int, burst int) *RateLimiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	if burst <= 0 {
		burst = 10
	}
	return &RateLimiter{
		ratePerSec: float64(perMinute) / 60.0,
		burst:      float64(burst),
		ttl:        15 * time.Minute,
		nowFn:      time.Now,
		buckets:    make(map[string]*rateBucket),
	}
}

func (rl *RateLimiter) Allow(key string) (allowed bool, retryAfterSec int) {
	now := rl.nowFn()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.calls++
	if rl.calls%500 == 0 {
		for k, b := range rl.buckets {
			if now.Sub(b.lastRefill) > rl.ttl {
				delete(rl.buckets, k)
			}
		}
	}

	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &rateBucket{
			tokens:     rl.burst - 1,
			lastRefill: now,
		}
		return true, 0
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * rl.ratePerSec
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastRefill = now
	}

	if b.tokens >= 1 {
		b.tokens -= 1
		return true, 0
	}

	waitSec := int((1.0-b.tokens)/rl.ratePerSec) + 1
	if waitSec < 1 {
		waitSec = 1
	}
	return false, waitSec
}

func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			allowed, retryAfter := rl.Allow(ip)
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				writeJSON(w, http.StatusTooManyRequests, errorEnvelope{
					Error: "terlalu banyak permintaan, silakan coba lagi sebentar",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			if ip := strings.TrimSpace(first); ip != "" {
				return ip
			}
		}
		return xff
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}
