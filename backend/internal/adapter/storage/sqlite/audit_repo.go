package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type AuditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Record(ctx context.Context, entry domain.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (admin_email, action, entity_type, entity_id, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, entry.AdminEmail, entry.Action, entry.EntityType, entry.EntityID, entry.Detail, entry.CreatedAt.UnixNano())
	if err != nil {
		return fmt.Errorf("record audit log: %w", err)
	}
	return nil
}

func (r *AuditLogRepository) ListRecent(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, admin_email, action, entity_type, entity_id, detail, created_at
		FROM audit_logs
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	out := []domain.AuditLog{}
	for rows.Next() {
		var entry domain.AuditLog
		var createdNano int64
		if err := rows.Scan(&entry.ID, &entry.AdminEmail, &entry.Action, &entry.EntityType, &entry.EntityID, &entry.Detail, &createdNano); err != nil {
			return nil, err
		}
		entry.CreatedAt = time.Unix(0, createdNano).UTC()
		out = append(out, entry)
	}
	return out, rows.Err()
}
