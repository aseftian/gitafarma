package medicine

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/adapter/storage/sqlite"
	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type stubSheet struct {
	upserted []domain.Medicine
	deleted  []string
	failErr  error
}

func (s *stubSheet) FetchAllMedicines(context.Context) ([]domain.Medicine, error) { return nil, nil }
func (s *stubSheet) UpsertMedicine(_ context.Context, med domain.Medicine) error {
	if s.failErr != nil {
		return s.failErr
	}
	s.upserted = append(s.upserted, med)
	return nil
}
func (s *stubSheet) DeleteMedicine(_ context.Context, id string) error {
	if s.failErr != nil {
		return s.failErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubNotifier struct {
	changes []domain.PriceChange
}

func (n *stubNotifier) Enabled() bool     { return true }
func (n *stubNotifier) PublicKey() string { return "pub" }
func (n *stubNotifier) NotifyPriceChanges(_ context.Context, c []domain.PriceChange) error {
	n.changes = append(n.changes, c...)
	return nil
}

func TestMedicineService_CRUDWriteThroughAndRecentSetting(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	medRepo := sqlite.NewMedicineRepository(db)
	setRepo := sqlite.NewSettingRepository(db)
	auditRepo := sqlite.NewAuditLogRepository(db)
	sheet := &stubSheet{}
	notifier := &stubNotifier{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t0 := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	_ = setRepo.EnsureDefaults(ctx, domain.DefaultSettings(t0))

	svc := NewService(medRepo, setRepo, sheet, auditRepo, notifier, logger)
	svc.nowFn = func() time.Time { return t0 }

	// Create
	created, err := svc.Create(ctx, "admin@gitafarma.id", domain.Medicine{
		Name: "Paracetamol 500mg", Barcode: "8991", SKU: "GF-01", Price: 5000, UOM: "Strip", AltUOM: "Tablet",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || len(sheet.upserted) != 1 || len(notifier.changes) != 1 {
		t.Fatalf("unexpected create state: %+v", created)
	}

	// Update price -> triggers push
	created.Price = 5500
	updated, err := svc.Update(ctx, "admin@gitafarma.id", created)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Price != 5500 || len(notifier.changes) != 2 {
		t.Fatalf("expected price push notification on update")
	}

	// ListRecent uses recent_days=7
	recent, err := svc.ListRecent(ctx, 20, 0)
	if err != nil || recent.RecentDays != 7 || recent.Total != 1 {
		t.Fatalf("unexpected recent result: %+v err=%v", recent, err)
	}

	// Sheet failure prevents local mutation
	sheet.failErr = errors.New("sheets api error")
	if err := svc.Delete(ctx, "admin@gitafarma.id", created.ID); err == nil {
		t.Fatalf("expected delete error when sheet fails")
	}
	if _, err := medRepo.FindByID(ctx, created.ID); err != nil {
		t.Fatalf("local medicine must remain intact when sheet delete fails: %v", err)
	}

	// Delete succeeds once sheet recovers
	sheet.failErr = nil
	if err := svc.Delete(ctx, "admin@gitafarma.id", created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := medRepo.FindByID(ctx, created.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	logs, _ := auditRepo.ListRecent(ctx, 10)
	if len(logs) != 3 {
		t.Fatalf("expected 3 audit logs (create, update, delete), got %d", len(logs))
	}
}
