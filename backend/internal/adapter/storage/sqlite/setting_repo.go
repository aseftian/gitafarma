package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type SettingRepository struct {
	db *sql.DB
}

func NewSettingRepository(db *sql.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) Get(ctx context.Context, key string) (domain.Setting, error) {
	var s domain.Setting
	var updatedNano int64
	err := r.db.QueryRowContext(ctx, `
		SELECT key, value, description, updated_at
		FROM settings WHERE key = ?
	`, key).Scan(&s.Key, &s.Value, &s.Description, &updatedNano)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Setting{}, domain.ErrNotFound
		}
		return domain.Setting{}, fmt.Errorf("get setting %s: %w", key, err)
	}
	s.UpdatedAt = time.Unix(0, updatedNano).UTC()
	return s, nil
}

func (r *SettingRepository) List(ctx context.Context) ([]domain.Setting, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT key, value, description, updated_at
		FROM settings ORDER BY key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	out := []domain.Setting{}
	for rows.Next() {
		var s domain.Setting
		var updatedNano int64
		if err := rows.Scan(&s.Key, &s.Value, &s.Description, &updatedNano); err != nil {
			return nil, err
		}
		s.UpdatedAt = time.Unix(0, updatedNano).UTC()
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SettingRepository) Upsert(ctx context.Context, setting domain.Setting) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, description, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			description = CASE WHEN excluded.description != '' THEN excluded.description ELSE settings.description END,
			updated_at = excluded.updated_at
	`, setting.Key, setting.Value, setting.Description, setting.UpdatedAt.UnixNano())
	if err != nil {
		return fmt.Errorf("upsert setting %s: %w", setting.Key, err)
	}
	return nil
}

func (r *SettingRepository) EnsureDefaults(ctx context.Context, defaults []domain.Setting) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO settings (key, value, description, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, d := range defaults {
		if _, err := stmt.ExecContext(ctx, d.Key, d.Value, d.Description, d.UpdatedAt.UnixNano()); err != nil {
			return err
		}
	}
	return tx.Commit()
}
