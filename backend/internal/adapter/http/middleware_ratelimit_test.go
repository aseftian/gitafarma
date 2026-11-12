package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_EnforcesBurstAndRefills(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	rl := NewRateLimiter(60, 2) // 1 req/sec, burst 2
	rl.nowFn = func() time.Time { return now }

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	doReq := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/medicines", nil)
		req.RemoteAddr = ip + ":12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := doReq("10.0.0.1"); code != http.StatusOK {
		t.Fatalf("req 1 expected 200, got %d", code)
	}
	if code := doReq("10.0.0.1"); code != http.StatusOK {
		t.Fatalf("req 2 expected 200, got %d", code)
	}
	if code := doReq("10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("req 3 expected 429, got %d", code)
	}

	// Different IP unaffected
	if code := doReq("10.0.0.2"); code != http.StatusOK {
		t.Fatalf("other IP expected 200, got %d", code)
	}

	// Refill after 1 second
	now = now.Add(1 * time.Second)
	if code := doReq("10.0.0.1"); code != http.StatusOK {
		t.Fatalf("refilled req expected 200, got %d", code)
	}
}

func TestClientIP_PreventsXFFSpoofing(t *testing.T) {
	// 1. Direct untrusted public peer spoofing X-Forwarded-For -> ignored, uses RemoteAddr
	reqDirect := httptest.NewRequest(http.MethodGet, "/", nil)
	reqDirect.RemoteAddr = "203.0.113.50:54321"
	reqDirect.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := clientIP(reqDirect); got != "203.0.113.50" {
		t.Fatalf("expected direct peer IP 203.0.113.50, got %q", got)
	}

	// 2. Behind trusted Docker proxy (172.18.0.2): client spoofed 1.2.3.4, proxy appended real IP 198.51.100.77
	reqProxied := httptest.NewRequest(http.MethodGet, "/", nil)
	reqProxied.RemoteAddr = "172.18.0.2:41000"
	reqProxied.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.77")
	if got := clientIP(reqProxied); got != "198.51.100.77" {
		t.Fatalf("expected rightmost proxy-appended IP 198.51.100.77, got %q", got)
	}
}
