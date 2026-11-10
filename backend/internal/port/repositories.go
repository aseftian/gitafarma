package port

import (
	"context"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

// MedicineSearchQuery mendefinisikan filter pencarian obat dari halaman karyawan maupun admin.
type MedicineSearchQuery struct {
	Query   string
	Barcode string
	SKU     string
	Limit   int
	Offset  int
}

// MedicinePage adalah hasil paginasi pencarian atau daftar obat terbaru.
type MedicinePage struct {
	Items []domain.Medicine `json:"items"`
	Total int               `json:"total"`
}

// MedicineRepository adalah port penyimpanan lokal (SQLite) untuk entitas obat.
type MedicineRepository interface {
	Search(ctx context.Context, q MedicineSearchQuery) (MedicinePage, error)
	FindByID(ctx context.Context, id string) (domain.Medicine, error)
	FindByBarcode(ctx context.Context, barcode string) (domain.Medicine, error)
	ListAll(ctx context.Context) ([]domain.Medicine, error)
	ListRecent(ctx context.Context, since time.Time, limit, offset int) (MedicinePage, error)
	Upsert(ctx context.Context, med domain.Medicine) error
	ApplySyncDiff(ctx context.Context, toUpsert []domain.Medicine, toDeleteIDs []string) error
	Delete(ctx context.Context, id string) error
}

// SettingRepository menyimpan konfigurasi aplikasi seperti parameter recent_days.
type SettingRepository interface {
	Get(ctx context.Context, key string) (domain.Setting, error)
	List(ctx context.Context) ([]domain.Setting, error)
	Upsert(ctx context.Context, setting domain.Setting) error
	EnsureDefaults(ctx context.Context, defaults []domain.Setting) error
}

// AdminWhitelistRepository menyediakan pengecekan whitelist admin dari sheet admins.
type AdminWhitelistRepository interface {
	FindByEmail(ctx context.Context, email string) (domain.Admin, error)
	Refresh(ctx context.Context) error
}

// PushSubscriptionRepository menyimpan endpoint Web Push karyawan.
type PushSubscriptionRepository interface {
	Save(ctx context.Context, sub domain.PushSubscription) error
	DeleteByEndpoint(ctx context.Context, endpoint string) error
	ListAll(ctx context.Context) ([]domain.PushSubscription, error)
}

// AuditLogRepository mencatat aksi administrasi.
type AuditLogRepository interface {
	Record(ctx context.Context, entry domain.AuditLog) error
	ListRecent(ctx context.Context, limit int) ([]domain.AuditLog, error)
}
