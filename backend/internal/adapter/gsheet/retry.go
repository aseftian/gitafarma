package gsheet

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"time"

	"google.golang.org/api/googleapi"
)

type retryConfig struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

var defaultRetryConfig = retryConfig{
	maxAttempts: 4,
	baseDelay:   400 * time.Millisecond,
	maxDelay:    5 * time.Second,
}

func retryWithBackoff[T any](ctx context.Context, cfg retryConfig, fn func() (T, error)) (T, error) {
	var zero T
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < cfg.maxAttempts; attempt++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}
		lastErr = err
		if !isRetryableError(err) || attempt == cfg.maxAttempts-1 {
			return zero, err
		}

		delay := cfg.baseDelay << attempt
		if delay > cfg.maxDelay {
			delay = cfg.maxDelay
		}
		// Tambah jitter 0..25% agar tidak terjadi thundering herd
		jitter := time.Duration(rand.Int64N(int64(delay/4) + 1))
		timer := time.NewTimer(delay + jitter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, lastErr
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		return gErr.Code == http.StatusTooManyRequests ||
			gErr.Code == http.StatusInternalServerError ||
			gErr.Code == http.StatusBadGateway ||
			gErr.Code == http.StatusServiceUnavailable ||
			gErr.Code == http.StatusGatewayTimeout
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
