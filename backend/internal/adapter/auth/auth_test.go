package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

func TestJWTSessionManager_IssueVerifyAndExpiry(t *testing.T) {
	mgr, err := NewJWTSessionManager("super-secret-key-for-testing-123", 8*time.Hour)
	if err != nil {
		t.Fatalf("new jwt session manager: %v", err)
	}

	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	mgr.nowFn = func() time.Time { return now }

	tok, exp, err := mgr.Issue(domain.Admin{
		Email:  "admin@gitafarma.id",
		Name:   "Pengelola",
		Role:   "admin",
		Active: true,
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !exp.Equal(now.Add(8 * time.Hour)) {
		t.Fatalf("unexpected expiry: %v", exp)
	}

	claims, err := mgr.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Email != "admin@gitafarma.id" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Wrong secret rejected
	otherMgr, _ := NewJWTSessionManager("another-secret-key-for-testing-999", 8*time.Hour)
	if _, err := otherMgr.Verify(tok); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for wrong secret, got %v", err)
	}

	// Expired after 8h
	now = now.Add(9 * time.Hour)
	if _, err := mgr.Verify(tok); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized after expiry, got %v", err)
	}
}
