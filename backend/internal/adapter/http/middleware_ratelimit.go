package http

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultMaxRateBuckets = 10000

type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter membatasi jumlah request per IP dengan algoritma token bucket dan batas memori maksimum.
type RateLimiter struct {
	ratePerSec float64
	burst      float64
	ttl        time.Duration
	maxBuckets int
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
		maxBuckets: defaultMaxRateBuckets,
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
		rl.evictExpiredLocked(now)
	}

	b, ok := rl.buckets[key]
	if !ok {
		if rl.maxBuckets > 0 && len(rl.buckets) >= rl.maxBuckets {
			rl.evictExpiredLocked(now)
			if len(rl.buckets) >= rl.maxBuckets {
				rl.evictOldestLocked()
			}
		}
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

func (rl *RateLimiter) evictExpiredLocked(now time.Time) {
	for k, b := range rl.buckets {
		if now.Sub(b.lastRefill) > rl.ttl {
			delete(rl.buckets, k)
		}
	}
}

func (rl *RateLimiter) evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time
	for k, b := range rl.buckets {
		if oldestKey == "" || b.lastRefill.Before(oldestTime) {
			oldestKey = k
			oldestTime = b.lastRefill
		}
	}
	if oldestKey != "" {
		delete(rl.buckets, oldestKey)
	}
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

// clientIP mengekstrak IP klien dengan perlindungan terhadap spoofing X-Forwarded-For:
// Header proxy (X-Real-IP / elemen terakhir X-Forwarded-For) hanya dipercaya bila koneksi TCP
// langsung (RemoteAddr) berasal dari jaringan loopback atau private (reverse proxy internal Docker/VPS).
func clientIP(r *http.Request) string {
	remoteIP := extractHost(r.RemoteAddr)
	if isTrustedProxyIP(remoteIP) {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				candidate := strings.TrimSpace(parts[i])
				if candidate != "" {
					return candidate
				}
			}
		}
		if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
			return xrip
		}
	}
	return remoteIP
}

func extractHost(addr string) string {
	addr = strings.TrimSpace(addr)
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func isTrustedProxyIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}
