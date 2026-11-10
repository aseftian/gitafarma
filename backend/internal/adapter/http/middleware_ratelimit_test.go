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
