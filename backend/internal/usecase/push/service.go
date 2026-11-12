package push

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

// allowedPushDomainSuffixes membatasi endpoint Web Push hanya ke penyedia resmi browser
// untuk mencegah Server-Side Request Forgery (OWASP A10:2021).
var allowedPushDomainSuffixes = []string{
	".googleapis.com",
	".google.com",
	".push.services.mozilla.com",
	".mozaws.net",
	".notify.windows.com",
	".push.apple.com",
}

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
	sub.UserAgent = strings.TrimSpace(sub.UserAgent)
	if len(sub.UserAgent) > 512 {
		sub.UserAgent = sub.UserAgent[:512]
	}

	if sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "" {
		return fmt.Errorf("%w: endpoint, p256dh, dan auth wajib diisi", domain.ErrInvalidInput)
	}
	if len(sub.Endpoint) > 2048 || len(sub.P256dh) > 256 || len(sub.Auth) > 256 {
		return fmt.Errorf("%w: ukuran payload subscription melebihi batas", domain.ErrInvalidInput)
	}
	if err := validatePushEndpointURL(sub.Endpoint); err != nil {
		return err
	}

	sub.CreatedAt = s.nowFn().UTC()
	return s.repo.Save(ctx, sub)
}

func validatePushEndpointURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" || u.User != nil {
		return fmt.Errorf("%w: endpoint push harus berupa URL https yang valid", domain.ErrInvalidInput)
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" || host == "localhost" || net.ParseIP(host) != nil {
		return fmt.Errorf("%w: host endpoint push tidak diizinkan", domain.ErrInvalidInput)
	}
	for _, suffix := range allowedPushDomainSuffixes {
		if strings.HasSuffix(host, suffix) || host == strings.TrimPrefix(suffix, ".") {
			return nil
		}
	}
	return fmt.Errorf("%w: domain endpoint push tidak termasuk penyedia Web Push resmi", domain.ErrInvalidInput)
}

func (s *Service) Unsubscribe(ctx context.Context, endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return fmt.Errorf("%w: endpoint wajib diisi", domain.ErrInvalidInput)
	}
	return s.repo.DeleteByEndpoint(ctx, endpoint)
}
