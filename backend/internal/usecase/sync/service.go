package sync

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

// Service mengelola sinkronisasi berkala Google Sheet -> SQLite dan deteksi perubahan harga.
type Service struct {
	sheet           port.MedicineSheetGateway
	medRepo         port.MedicineRepository
	settingRepo     port.SettingRepository
	whitelistRepo   port.AdminWhitelistRepository
	notifier        port.PushNotifier
	defaultInterval time.Duration
	logger          *slog.Logger
	nowFn           func() time.Time
	mu              sync.Mutex
}

func NewService(
	sheet port.MedicineSheetGateway,
	medRepo port.MedicineRepository,
	settingRepo port.SettingRepository,
	whitelistRepo port.AdminWhitelistRepository,
	notifier port.PushNotifier,
	defaultInterval time.Duration,
	logger *slog.Logger,
) *Service {
	if defaultInterval < time.Minute {
		defaultInterval = 5 * time.Minute
	}
	return &Service{
		sheet:           sheet,
		medRepo:         medRepo,
		settingRepo:     settingRepo,
		whitelistRepo:   whitelistRepo,
		notifier:        notifier,
		defaultInterval: defaultInterval,
		logger:          logger,
		nowFn:           time.Now,
	}
}

func (s *Service) SyncOnce(ctx context.Context) (port.SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	incoming, err := s.sheet.FetchAllMedicines(ctx)
	if err != nil {
		return port.SyncResult{}, err
	}

	existingList, err := s.medRepo.ListAll(ctx)
	if err != nil {
		return port.SyncResult{}, err
	}

	existingMap := make(map[string]domain.Medicine, len(existingList))
	for _, m := range existingList {
		existingMap[m.ID] = m
	}

	now := s.nowFn().UTC()
	seen := make(map[string]struct{}, len(incoming))
	var toUpsert []domain.Medicine
	var changes []domain.PriceChange

	for _, inc := range incoming {
		seen[inc.ID] = struct{}{}
		old, exists := existingMap[inc.ID]
		if !exists {
			if inc.UpdatedAt.IsZero() {
				inc.UpdatedAt = now
			}
			toUpsert = append(toUpsert, inc)
			changes = append(changes, domain.PriceChange{
				Medicine:   inc,
				OldPrice:   0,
				NewPrice:   inc.Price,
				ChangeType: domain.ChangeAdded,
			})
			continue
		}

		if old.Price != inc.Price {
			inc.UpdatedAt = now
			toUpsert = append(toUpsert, inc)
			changes = append(changes, domain.PriceChange{
				Medicine:   inc,
				OldPrice:   old.Price,
				NewPrice:   inc.Price,
				ChangeType: domain.ChangePriceUpdated,
			})
			continue
		}

		if medicineMetadataChanged(old, inc) {
			if !inc.UpdatedAt.After(old.UpdatedAt) {
				inc.UpdatedAt = old.UpdatedAt
			}
			toUpsert = append(toUpsert, inc)
		}
	}

	var toDelete []string
	for id := range existingMap {
		if _, ok := seen[id]; !ok {
			toDelete = append(toDelete, id)
		}
	}

	if err := s.medRepo.ApplySyncDiff(ctx, toUpsert, toDelete); err != nil {
		return port.SyncResult{}, err
	}

	// Hindari spam notifikasi saat seeding awal ketika database lokal sebelumnya kosong.
	notifyCount := 0
	if len(existingList) > 0 && len(changes) > 0 {
		notifyCount = len(changes)
		if s.notifier != nil {
			if err := s.notifier.NotifyPriceChanges(ctx, changes); err != nil && s.logger != nil {
				s.logger.Warn("push notification failed after sync", "error", err)
			}
		}
	}

	return port.SyncResult{
		TotalFetched: len(incoming),
		Upserted:     len(toUpsert),
		Deleted:      len(toDelete),
		PriceChanges: notifyCount,
	}, nil
}

// SyncAllManual menjalankan sinkronisasi obat dan refresh cache whitelist admin secara bersamaan.
func (s *Service) SyncAllManual(ctx context.Context) (port.SyncResult, error) {
	if s.whitelistRepo != nil {
		if err := s.whitelistRepo.Refresh(ctx); err != nil && !errors.Is(err, domain.ErrSheetNotConfigured) {
			return port.SyncResult{}, err
		}
	}
	return s.SyncOnce(ctx)
}

// StartScheduler menjalankan sinkronisasi berkala hingga ctx dibatalkan.
func (s *Service) StartScheduler(ctx context.Context) {
	go func() {
		s.runSafeSync(ctx)
		for {
			interval := s.currentInterval(ctx)
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				s.runSafeSync(ctx)
			}
		}
	}()
}

func (s *Service) runSafeSync(ctx context.Context) {
	res, err := s.SyncOnce(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrSheetNotConfigured) {
			return
		}
		if s.logger != nil {
			s.logger.Warn("scheduled sync failed", "error", err)
		}
		return
	}
	if s.logger != nil && (res.Upserted > 0 || res.Deleted > 0) {
		s.logger.Info("scheduled sync completed",
			"fetched", res.TotalFetched,
			"upserted", res.Upserted,
			"deleted", res.Deleted,
			"price_changes", res.PriceChanges,
		)
	}
}

func (s *Service) currentInterval(ctx context.Context) time.Duration {
	if s.settingRepo == nil {
		return s.defaultInterval
	}
	setting, err := s.settingRepo.Get(ctx, domain.SettingSyncInterval)
	if err != nil {
		return s.defaultInterval
	}
	d, err := time.ParseDuration(setting.Value)
	if err != nil || d < time.Minute {
		return s.defaultInterval
	}
	return d
}

func medicineMetadataChanged(a, b domain.Medicine) bool {
	return a.Name != b.Name ||
		a.Barcode != b.Barcode ||
		a.SKU != b.SKU ||
		a.UOM != b.UOM ||
		a.AltUOM != b.AltUOM
}
