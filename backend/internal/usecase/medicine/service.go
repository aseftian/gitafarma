package medicine

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	"github.com/google/uuid"
)

type RecentResult struct {
	Items      []domain.Medicine `json:"items"`
	Total      int               `json:"total"`
	RecentDays int               `json:"recent_days"`
}

// Service menangani pencarian obat karyawan serta CRUD write-through ke Google Sheet oleh admin.
type Service struct {
	medRepo  port.MedicineRepository
	setRepo  port.SettingRepository
	sheet    port.MedicineSheetGateway
	audits   port.AuditLogRepository
	notifier port.PushNotifier
	logger   *slog.Logger
	nowFn    func() time.Time
}

func NewService(
	medRepo port.MedicineRepository,
	setRepo port.SettingRepository,
	sheet port.MedicineSheetGateway,
	audits port.AuditLogRepository,
	notifier port.PushNotifier,
	logger *slog.Logger,
) *Service {
	return &Service{
		medRepo:  medRepo,
		setRepo:  setRepo,
		sheet:    sheet,
		audits:   audits,
		notifier: notifier,
		logger:   logger,
		nowFn:    time.Now,
	}
}

func (s *Service) Search(ctx context.Context, q port.MedicineSearchQuery) (port.MedicinePage, error) {
	return s.medRepo.Search(ctx, q)
}

func (s *Service) FindByID(ctx context.Context, id string) (domain.Medicine, error) {
	return s.medRepo.FindByID(ctx, id)
}

func (s *Service) ListRecent(ctx context.Context, limit, offset int) (RecentResult, error) {
	days := 7
	if s.setRepo != nil {
		if setting, err := s.setRepo.Get(ctx, domain.SettingRecentDays); err == nil {
			if parsed, convErr := strconv.Atoi(strings.TrimSpace(setting.Value)); convErr == nil && parsed > 0 {
				days = parsed
			}
		}
	}

	since := s.nowFn().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	page, err := s.medRepo.ListRecent(ctx, since, limit, offset)
	if err != nil {
		return RecentResult{}, err
	}
	return RecentResult{
		Items:      page.Items,
		Total:      page.Total,
		RecentDays: days,
	}, nil
}

func (s *Service) Create(ctx context.Context, adminEmail string, med domain.Medicine) (domain.Medicine, error) {
	med = normalizeMedicine(med)
	if med.ID == "" {
		med.ID = uuid.NewString()
	}
	if err := med.Validate(); err != nil {
		return domain.Medicine{}, err
	}
	med.UpdatedAt = s.nowFn().UTC()

	if err := s.sheet.UpsertMedicine(ctx, med); err != nil {
		return domain.Medicine{}, fmt.Errorf("write-through gsheet create: %w", err)
	}
	if err := s.medRepo.Upsert(ctx, med); err != nil {
		return domain.Medicine{}, fmt.Errorf("save local medicine: %w", err)
	}

	s.recordAudit(ctx, adminEmail, "create", med.ID, fmt.Sprintf("%s (%s) Rp %d", med.Name, med.UOM, med.Price))
	s.notifySafe(ctx, []domain.PriceChange{
		{Medicine: med, OldPrice: 0, NewPrice: med.Price, ChangeType: domain.ChangeAdded},
	})
	return med, nil
}

func (s *Service) Update(ctx context.Context, adminEmail string, med domain.Medicine) (domain.Medicine, error) {
	if strings.TrimSpace(med.ID) == "" {
		return domain.Medicine{}, fmt.Errorf("%w: id wajib diisi", domain.ErrInvalidInput)
	}
	med = normalizeMedicine(med)
	if err := med.Validate(); err != nil {
		return domain.Medicine{}, err
	}

	existing, err := s.medRepo.FindByID(ctx, med.ID)
	if err != nil {
		return domain.Medicine{}, err
	}
	med.UpdatedAt = s.nowFn().UTC()

	if err := s.sheet.UpsertMedicine(ctx, med); err != nil {
		return domain.Medicine{}, fmt.Errorf("write-through gsheet update: %w", err)
	}
	if err := s.medRepo.Upsert(ctx, med); err != nil {
		return domain.Medicine{}, fmt.Errorf("update local medicine: %w", err)
	}

	detail := fmt.Sprintf("%s: Rp %d -> Rp %d", med.Name, existing.Price, med.Price)
	s.recordAudit(ctx, adminEmail, "update", med.ID, detail)

	if existing.Price != med.Price {
		s.notifySafe(ctx, []domain.PriceChange{
			{Medicine: med, OldPrice: existing.Price, NewPrice: med.Price, ChangeType: domain.ChangePriceUpdated},
		})
	}
	return med, nil
}

func (s *Service) Delete(ctx context.Context, adminEmail, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: id wajib diisi", domain.ErrInvalidInput)
	}
	existing, err := s.medRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.sheet.DeleteMedicine(ctx, id); err != nil {
		return fmt.Errorf("write-through gsheet delete: %w", err)
	}
	if err := s.medRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete local medicine: %w", err)
	}

	s.recordAudit(ctx, adminEmail, "delete", id, existing.Name)
	return nil
}

func (s *Service) recordAudit(ctx context.Context, adminEmail, action, entityID, detail string) {
	if s.audits == nil {
		return
	}
	_ = s.audits.Record(ctx, domain.AuditLog{
		AdminEmail: adminEmail,
		Action:     action,
		EntityType: "medicine",
		EntityID:   entityID,
		Detail:     detail,
		CreatedAt:  s.nowFn().UTC(),
	})
}

func (s *Service) notifySafe(ctx context.Context, changes []domain.PriceChange) {
	if s.notifier == nil || len(changes) == 0 {
		return
	}
	if err := s.notifier.NotifyPriceChanges(ctx, changes); err != nil && s.logger != nil {
		s.logger.Warn("push notification failed after medicine mutation", "error", err)
	}
}

func normalizeMedicine(m domain.Medicine) domain.Medicine {
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Barcode = strings.TrimSpace(m.Barcode)
	m.SKU = strings.TrimSpace(m.SKU)
	m.UOM = strings.TrimSpace(m.UOM)
	m.AltUOM = strings.TrimSpace(m.AltUOM)
	return m
}
