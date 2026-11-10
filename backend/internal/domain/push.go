package domain

import "time"

// PushSubscription menyimpan langganan Web Push (VAPID) perangkat karyawan.
type PushSubscription struct {
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}
