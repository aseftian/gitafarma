package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type PushSubscriptionRepository struct {
	db *sql.DB
}

func NewPushSubscriptionRepository(db *sql.DB) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{db: db}
}

func (r *PushSubscriptionRepository) Save(ctx context.Context, sub domain.PushSubscription) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO push_subscriptions (endpoint, p256dh, auth, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET
			p256dh = excluded.p256dh,
			auth = excluded.auth,
			user_agent = excluded.user_agent
	`, sub.Endpoint, sub.P256dh, sub.Auth, sub.UserAgent, sub.CreatedAt.UnixNano())
	if err != nil {
		return fmt.Errorf("save push subscription: %w", err)
	}
	return nil
}

func (r *PushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint)
	if err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return nil
}

func (r *PushSubscriptionRepository) ListAll(ctx context.Context) ([]domain.PushSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT endpoint, p256dh, auth, user_agent, created_at
		FROM push_subscriptions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}
	defer rows.Close()

	out := []domain.PushSubscription{}
	for rows.Next() {
		var sub domain.PushSubscription
		var createdNano int64
		if err := rows.Scan(&sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.UserAgent, &createdNano); err != nil {
			return nil, err
		}
		sub.CreatedAt = time.Unix(0, createdNano).UTC()
		out = append(out, sub)
	}
	return out, rows.Err()
}
