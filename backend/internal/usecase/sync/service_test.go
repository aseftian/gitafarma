package sync

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/adapter/storage/sqlite"
	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type stubSheet struct {
	meds []domain.Medicine
}

func (s *stubSheet) FetchAllMedicines(context.Context) ([]domain.Medicine, error) {
	return s.meds, nil
}
func (s *stubSheet) UpsertMedicine(context.Context, domain.Medicine) error { return nil }
func (s *stubSheet) DeleteMedicine(context.Context, string) error          { return nil }

type stubNotifier struct {
	changes [][]domain.PriceChange
}

func (n *stubNotifier) Enabled() bool     { return true }
func (n *stubNotifier) PublicKey() string { return "pub" }
func (n *stubNotifier) NotifyPriceChanges(_ context.Context, changes []domain.PriceChange) error {
	n.changes = append(n.changes, changes)
	return nil
}

func TestSyncService_DiffDetectionAndPush(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	medRepo := sqlite.NewMedicineRepository(db)
	settingRepo := sqlite.NewSettingRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	sheet := &stubSheet{
		meds: []domain.Medicine{
			{ID: "1", Name: "Paracetamol 500mg", Barcode: "8991", SKU: "GF-1", Price: 5000, UOM: "Strip", AltUOM: "Tablet", UpdatedAt: t0},
			{ID: "2", Name: "Amoxicillin 500mg", Barcode: "8992", SKU: "GF-2", Price: 12000, UOM: "Strip", AltUOM: "Kapsul", UpdatedAt: t0},
		},
	}
	notifier := &stubNotifier{}

	svc := NewService(sheet, medRepo, settingRepo, nil, notifier, 5*time.Minute, logger)
	svc.nowFn = func() time.Time { return t0 }

	// 1. Cold start seed -> upserts 2, NO push spam
	res, err := svc.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	if res.Upserted != 2 || res.PriceChanges != 0 || len(notifier.changes) != 0 {
		t.Fatalf("unexpected initial sync result: %+v notifier=%d", res, len(notifier.changes))
	}

	// 2. Price change on ID 1, metadata only on ID 2, ID 3 added
	t1 := t0.Add(24 * time.Hour)
	svc.nowFn = func() time.Time { return t1 }
	sheet.meds = []domain.Medicine{
		{ID: "1", Name: "Paracetamol 500mg", Barcode: "8991", SKU: "GF-1", Price: 5500, UOM: "Strip", AltUOM: "Tablet", UpdatedAt: t0},
		{ID: "2", Name: "Amoxicillin 500mg Generik", Barcode: "8992", SKU: "GF-2", Price: 12000, UOM: "Strip", AltUOM: "Kapsul", UpdatedAt: t0},
		{ID: "3", Name: "Vitamin C 500mg", Barcode: "8993", SKU: "GF-3", Price: 8000, UOM: "Strip", AltUOM: "Tablet", UpdatedAt: t1},
	}

	res, err = svc.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if res.Upserted != 3 || res.PriceChanges != 2 || len(notifier.changes) != 1 {
		t.Fatalf("unexpected second sync: %+v notifier=%+v", res, notifier.changes)
	}

	// ID 1 UpdatedAt updated to t1; ID 2 UpdatedAt preserved as t0 (metadata-only change)
	m1, _ := medRepo.FindByID(ctx, "1")
	m2, _ := medRepo.FindByID(ctx, "2")
	if !m1.UpdatedAt.Equal(t1) {
		t.Fatalf("expected ID 1 UpdatedAt=%v, got %v", t1, m1.UpdatedAt)
	}
	if !m2.UpdatedAt.Equal(t0) {
		t.Fatalf("expected ID 2 UpdatedAt preserved=%v, got %v", t0, m2.UpdatedAt)
	}

	// 3. Delete ID 2 from sheet -> removed locally
	sheet.meds = []domain.Medicine{sheet.meds[0], sheet.meds[2]}
	res, err = svc.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("third sync: %v", err)
	}
	if res.Deleted != 1 || res.Upserted != 0 {
		t.Fatalf("unexpected third sync: %+v", res)
	}
}
