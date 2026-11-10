package domain

import "time"

// Setting menyimpan parameter konfigurasi aplikasi yang dapat diubah pengelola.
type Setting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const (
	SettingRecentDays   = "recent_days"
	SettingSyncInterval = "sync_interval"
)

// DefaultSettings mengembalikan nilai awal parameter aplikasi saat belum tersimpan di database.
func DefaultSettings(now time.Time) []Setting {
	return []Setting{
		{
			Key:         SettingRecentDays,
			Value:       "7",
			Description: "Jumlah hari terakhir untuk daftar harga obat terbaru di halaman pencarian",
			UpdatedAt:   now,
		},
		{
			Key:         SettingSyncInterval,
			Value:       "5m",
			Description: "Interval sinkronisasi otomatis dari Google Sheet (contoh: 5m, 10m)",
			UpdatedAt:   now,
		},
	}
}
