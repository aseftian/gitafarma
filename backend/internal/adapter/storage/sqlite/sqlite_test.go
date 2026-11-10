package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

func openTestDB(t *testing.T) *MedicineRepository {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewMedicineRepository(db)
}

func TestMedicineRepository_SearchAndDiff(t *testing.T) {
	ctx := context.Background()
	repo := openTestDB(t)
	now := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)

	initial := []domain.Medicine{
		{ID: "1", Name: "Paracetamol 500mg Tablet", Barcode: "8991111000001", SKU: "GF-001", Price: 5000, UOM: "Strip", AltUOM: "Tablet", UpdatedAt: now.Add(-10 * 24 * time.Hour)},
		{ID: "2", Name: "Amoxicillin 500mg Kapsul", Barcode: "8991111000002", SKU: "GF-002", Price: 12000, UOM: "Strip", AltUOM: "Kapsul", UpdatedAt: now.Add(-2 * 24 * time.Hour)},
		{ID: "3", Name: "OBH Combi Batuk 100ml", Barcode: "8991111000003", SKU: "GF-003", Price: 18500, UOM: "Botol", AltUOM: "ml", UpdatedAt: now.Add(-20 * 24 * time.Hour)},
	}
	if err := repo.ApplySyncDiff(ctx, initial, nil); err != nil {
		t.Fatalf("apply sync diff: %v", err)
	}

	// FTS prefix match
	res, err := repo.Search(ctx, port.MedicineSearchQuery{Query: "parace 500"})
	if err != nil {
		t.Fatalf("search fts: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != "1" {
		t.Fatalf("expected Paracetamol, got %+v", res)
	}

	// Mid-word LIKE fallback (kasus OCR)
	res, err = repo.Search(ctx, port.MedicineSearchQuery{Query: "cetamol"})
	if err != nil {
		t.Fatalf("search mid-word: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != "1" {
		t.Fatalf("expected Paracetamol from mid-word search, got %+v", res)
	}

	// Barcode search
	res, err = repo.Search(ctx, port.MedicineSearchQuery{Barcode: "8991111000002"})
	if err != nil {
		t.Fatalf("search barcode: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != "2" {
		t.Fatalf("expected Amoxicillin, got %+v", res)
	}

	// Recent updates (last 7 days -> only ID 2)
	recent, err := repo.ListRecent(ctx, now.Add(-7*24*time.Hour), 20, 0)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if recent.Total != 1 || recent.Items[0].ID != "2" {
		t.Fatalf("expected 1 recent item, got %+v", recent)
	}

	// Sync diff update + delete
	updated := domain.Medicine{ID: "1", Name: "Paracetamol 500mg Tablet", Barcode: "8991111000001", SKU: "GF-001", Price: 5500, UOM: "Strip", AltUOM: "Tablet", UpdatedAt: now}
	if err := repo.ApplySyncDiff(ctx, []domain.Medicine{updated}, []string{"3"}); err != nil {
		t.Fatalf("apply diff 2: %v", err)
	}

	if _, err := repo.FindByID(ctx, "3"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for deleted id 3, got %v", err)
	}
	m1, err := repo.FindByID(ctx, "1")
	if err != nil || m1.Price != 5500 {
		t.Fatalf("expected updated price 5500, got %+v err=%v", m1, err)
	}
}

func TestSettingsPushAndAuditRepos(t *testing.T) {
	ctx := context.Background()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	settingRepo := NewSettingRepository(db)
	if err := settingRepo.EnsureDefaults(ctx, domain.DefaultSettings(now)); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	s, err := settingRepo.Get(ctx, domain.SettingRecentDays)
	if err != nil || s.Value != "7" {
		t.Fatalf("expected default recent_days=7, got %+v err=%v", s, err)
	}
	if err := settingRepo.Upsert(ctx, domain.Setting{Key: domain.SettingRecentDays, Value: "14", UpdatedAt: now}); err != nil {
		t.Fatalf("upsert setting: %v", err)
	}
	s, _ = settingRepo.Get(ctx, domain.SettingRecentDays)
	if s.Value != "14" {
		t.Fatalf("expected updated value 14, got %s", s.Value)
	}

	pushRepo := NewPushSubscriptionRepository(db)
	sub := domain.PushSubscription{Endpoint: "https://push.example/1", P256dh: "k1", Auth: "a1", UserAgent: "Chrome", CreatedAt: now}
	if err := pushRepo.Save(ctx, sub); err != nil {
		t.Fatalf("save sub: %v", err)
	}
	subs, err := pushRepo.ListAll(ctx)
	if err != nil || len(subs) != 1 {
		t.Fatalf("expected 1 sub, got %d err=%v", len(subs), err)
	}
	if err := pushRepo.DeleteByEndpoint(ctx, sub.Endpoint); err != nil {
		t.Fatalf("delete sub: %v", err)
	}
	subs, _ = pushRepo.ListAll(ctx)
	if len(subs) != 0 {
		t.Fatalf("expected 0 subs after delete, got %d", len(subs))
	}

	auditRepo := NewAuditLogRepository(db)
	if err := auditRepo.Record(ctx, domain.AuditLog{
		AdminEmail: "admin@gitafarma.id",
		Action:     "update_medicine",
		EntityType: "medicine",
		EntityID:   "1",
		Detail:     "price 5000 -> 5500",
		CreatedAt:  now,
	}); err != nil {
		t.Fatalf("record audit: %v", err)
	}
	logs, err := auditRepo.ListRecent(ctx, 10)
	if err != nil || len(logs) != 1 || logs[0].AdminEmail != "admin@gitafarma.id" {
		t.Fatalf("unexpected audit logs: %+v err=%v", logs, err)
	}
}
