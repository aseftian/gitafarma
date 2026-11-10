package domain

import "time"

// Admin merepresentasikan pengguna pengelola yang terdaftar di sheet admins.
type Admin struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

// AuditLog mencatat aksi mutasi yang dilakukan admin untuk keperluan jejak audit.
type AuditLog struct {
	ID         int64     `json:"id"`
	AdminEmail string    `json:"admin_email"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Detail     string    `json:"detail"`
	CreatedAt  time.Time `json:"created_at"`
}
