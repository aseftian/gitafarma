package setting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

// Service mengelola parameter aplikasi (mis. recent_days, sync_interval) dan audit log.
type Service struct {
	repo   port.SettingRepository
	audits port.AuditLogRepository
	nowFn  func() time.Time
}

func NewService(repo port.SettingRepository, audits port.AuditLogRepository) *Service {
	return &Service{
		repo:   repo,
		audits: audits,
		nowFn:  time.Now,
	}
}

func (s *Service) List(ctx context.Context) ([]domain.Setting, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, key string) (domain.Setting, error) {
	return s.repo.Get(ctx, key)
}

func (s *Service) Update(ctx context.Context, adminEmail, key, value string) (domain.Setting, error) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)

	desc, err := validateSetting(key, value)
	if err != nil {
		return domain.Setting{}, err
	}

	setting := domain.Setting{
		Key:         key,
		Value:       value,
		Description: desc,
		UpdatedAt:   s.nowFn().UTC(),
	}
	if err := s.repo.Upsert(ctx, setting); err != nil {
		return domain.Setting{}, err
	}
	if s.audits != nil {
		_ = s.audits.Record(ctx, domain.AuditLog{
			AdminEmail: adminEmail,
			Action:     "update_setting",
			EntityType: "setting",
			EntityID:   key,
			Detail:     fmt.Sprintf("%s = %s", key, value),
			CreatedAt:  s.nowFn().UTC(),
		})
	}
	return s.repo.Get(ctx, key)
}

func (s *Service) ListAuditLogs(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	if s.audits == nil {
		return []domain.AuditLog{}, nil
	}
	return s.audits.ListRecent(ctx, limit)
}

func validateSetting(key, value string) (string, error) {
	switch key {
	case domain.SettingRecentDays:
		days, err := strconv.Atoi(value)
		if err != nil || days < 1 || days > 365 {
			return "", fmt.Errorf("%w: recent_days harus angka 1 sampai 365", domain.ErrInvalidInput)
		}
		return "Jumlah hari terakhir untuk daftar harga obat terbaru di halaman pencarian", nil
	case domain.SettingSyncInterval:
		d, err := time.ParseDuration(value)
		if err != nil || d < time.Minute {
			return "", fmt.Errorf("%w: sync_interval minimal 1m (contoh: 5m, 15m)", domain.ErrInvalidInput)
		}
		return "Interval sinkronisasi otomatis dari Google Sheet (contoh: 5m, 10m)", nil
	default:
		return "", fmt.Errorf("%w: parameter pengaturan %q tidak dikenal", domain.ErrInvalidInput, key)
	}
}
