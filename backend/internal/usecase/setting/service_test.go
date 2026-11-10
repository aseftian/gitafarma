package setting

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/adapter/storage/sqlite"
	"github.com/ardisan/gitafarma/backend/internal/domain"
)

func TestSettingService_ValidationAndUpdate(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	setRepo := sqlite.NewSettingRepository(db)
	auditRepo := sqlite.NewAuditLogRepository(db)
	_ = setRepo.EnsureDefaults(ctx, domain.DefaultSettings(time.Now().UTC()))

	svc := NewService(setRepo, auditRepo)

	if _, err := svc.Update(ctx, "admin@gitafarma.id", domain.SettingRecentDays, "0"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for recent_days=0, got %v", err)
	}
	if _, err := svc.Update(ctx, "admin@gitafarma.id", domain.SettingSyncInterval, "10s"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for sync_interval=10s, got %v", err)
	}
	if _, err := svc.Update(ctx, "admin@gitafarma.id", "unknown_key", "1"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unknown key, got %v", err)
	}

	updated, err := svc.Update(ctx, "admin@gitafarma.id", domain.SettingRecentDays, "14")
	if err != nil || updated.Value != "14" {
		t.Fatalf("expected recent_days=14, got %+v err=%v", updated, err)
	}
}
