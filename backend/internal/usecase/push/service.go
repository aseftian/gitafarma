package push

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

type Service struct {
	repo     port.PushSubscriptionRepository
	notifier port.PushNotifier
	nowFn    func() time.Time
}

func NewService(repo port.PushSubscriptionRepository, notifier port.PushNotifier) *Service {
	return &Service{
		repo:     repo,
		notifier: notifier,
		nowFn:    time.Now,
	}
}

func (s *Service) PublicKey() (string, bool) {
	if s.notifier == nil {
		return "", false
	}
	return s.notifier.PublicKey(), s.notifier.Enabled()
}

func (s *Service) Subscribe(ctx context.Context, sub domain.PushSubscription) error {
	sub.Endpoint = strings.TrimSpace(sub.Endpoint)
	sub.P256dh = strings.TrimSpace(sub.P256dh)
	sub.Auth = strings.TrimSpace(sub.Auth)
	if sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "" {
		return fmt.Errorf("%w: endpoint, p256dh, dan auth wajib diisi", domain.ErrInvalidInput)
	}
	u, err := url.ParseRequestURI(sub.Endpoint)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" {
		return fmt.Errorf("%w: endpoint push harus berupa URL https yang valid", domain.ErrInvalidInput)
	}
	sub.CreatedAt = s.nowFn().UTC()
	return s.repo.Save(ctx, sub)
}

func (s *Service) Unsubscribe(ctx context.Context, endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return fmt.Errorf("%w: endpoint wajib diisi", domain.ErrInvalidInput)
	}
	return s.repo.DeleteByEndpoint(ctx, endpoint)
}
