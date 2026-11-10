package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

type NotificationPayload struct {
	Title        string `json:"title"`
	Body         string `json:"body"`
	URL          string `json:"url"`
	ChangesCount int    `json:"changes_count"`
}

// WebPushNotifier mengirim notifikasi Web Push (VAPID) ke seluruh perangkat karyawan terdaftar.
type WebPushNotifier struct {
	repo       port.PushSubscriptionRepository
	publicKey  string
	privateKey string
	subject    string
	logger     *slog.Logger
	sendFn     func(ctx context.Context, payload []byte, sub *webpush.Subscription, opts *webpush.Options) (*http.Response, error)
}

func NewWebPushNotifier(
	repo port.PushSubscriptionRepository,
	publicKey, privateKey, subject string,
	logger *slog.Logger,
) *WebPushNotifier {
	return &WebPushNotifier{
		repo:       repo,
		publicKey:  strings.TrimSpace(publicKey),
		privateKey: strings.TrimSpace(privateKey),
		subject:    strings.TrimSpace(subject),
		logger:     logger,
		sendFn: func(ctx context.Context, payload []byte, sub *webpush.Subscription, opts *webpush.Options) (*http.Response, error) {
			return webpush.SendNotificationWithContext(ctx, payload, sub, opts)
		},
	}
}

func (n *WebPushNotifier) Enabled() bool {
	return n.publicKey != "" && n.privateKey != ""
}

func (n *WebPushNotifier) PublicKey() string {
	return n.publicKey
}

func (n *WebPushNotifier) NotifyPriceChanges(ctx context.Context, changes []domain.PriceChange) error {
	if !n.Enabled() || len(changes) == 0 {
		return nil
	}

	subs, err := n.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list push subscriptions: %w", err)
	}
	if len(subs) == 0 {
		return nil
	}

	msg := buildNotificationPayload(changes)
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, s := range subs {
		wg.Add(1)
		go func(sub domain.PushSubscription) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			wpSub := &webpush.Subscription{
				Endpoint: sub.Endpoint,
				Keys: webpush.Keys{
					P256dh: sub.P256dh,
					Auth:   sub.Auth,
				},
			}
			opts := &webpush.Options{
				Subscriber:      n.subject,
				VAPIDPublicKey:  n.publicKey,
				VAPIDPrivateKey: n.privateKey,
				TTL:             86400,
			}

			resp, sendErr := n.sendFn(ctx, raw, wpSub, opts)
			if resp != nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
					_ = n.repo.DeleteByEndpoint(ctx, sub.Endpoint)
					return
				}
			}
			if sendErr != nil && n.logger != nil {
				n.logger.Warn("web push send failed", "endpoint", sub.Endpoint, "error", sendErr)
			}
		}(s)
	}
	wg.Wait()
	return nil
}

func buildNotificationPayload(changes []domain.PriceChange) NotificationPayload {
	if len(changes) == 1 {
		c := changes[0]
		body := fmt.Sprintf("%s: %s", c.Medicine.Name, formatIDR(c.NewPrice))
		if c.ChangeType == domain.ChangePriceUpdated {
			body = fmt.Sprintf("%s: %s → %s", c.Medicine.Name, formatIDR(c.OldPrice), formatIDR(c.NewPrice))
		}
		return NotificationPayload{
			Title:        "Update Harga Obat",
			Body:         body,
			URL:          "/",
			ChangesCount: 1,
		}
	}

	first := changes[0].Medicine.Name
	return NotificationPayload{
		Title:        "Update Harga Obat",
		Body:         fmt.Sprintf("%d harga obat diperbarui (termasuk %s)", len(changes), first),
		URL:          "/",
		ChangesCount: len(changes),
	}
}

func formatIDR(amount int64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	s := strconv.FormatInt(amount, 10)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	prefix := "Rp "
	if neg {
		prefix = "-Rp "
	}
	return prefix + strings.Join(parts, ".")
}
