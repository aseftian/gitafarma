package port

import (
	"context"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

// MedicineSheetGateway mengelola baca/tulis data obat ke Google Sheets.
type MedicineSheetGateway interface {
	FetchAllMedicines(ctx context.Context) ([]domain.Medicine, error)
	UpsertMedicine(ctx context.Context, med domain.Medicine) error
	DeleteMedicine(ctx context.Context, id string) error
}

// AdminWhitelistGateway mengambil daftar admin yang diizinkan dari sheet admins.
type AdminWhitelistGateway interface {
	FetchAdmins(ctx context.Context) ([]domain.Admin, error)
}

// VerifiedGoogleIdentity adalah hasil verifikasi ID token Google yang terpercaya.
type VerifiedGoogleIdentity struct {
	Email         string
	Name          string
	EmailVerified bool
}

// GoogleIDTokenVerifier memverifikasi Google ID token dari client GIS.
type GoogleIDTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (VerifiedGoogleIdentity, error)
}

// SessionClaims merepresentasikan identitas admin pada token sesi.
type SessionClaims struct {
	Email     string
	Name      string
	Role      string
	ExpiresAt time.Time
}

// SessionManager menerbitkan dan memvalidasi token sesi admin.
type SessionManager interface {
	Issue(admin domain.Admin) (token string, expiresAt time.Time, err error)
	Verify(token string) (SessionClaims, error)
}

// PushNotifier mengirim notifikasi Web Push ke perangkat karyawan.
type PushNotifier interface {
	Enabled() bool
	PublicKey() string
	NotifyPriceChanges(ctx context.Context, changes []domain.PriceChange) error
}

// SyncResult meringkas hasil satu kali siklus sinkronisasi Google Sheet -> SQLite.
type SyncResult struct {
	TotalFetched int `json:"total_fetched"`
	Upserted     int `json:"upserted"`
	Deleted      int `json:"deleted"`
	PriceChanges int `json:"price_changes"`
}

// SyncRunner menjalankan sinkronisasi manual maupun terjadwal.
type SyncRunner interface {
	SyncOnce(ctx context.Context) (SyncResult, error)
}
