package push

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type memPushRepo struct {
	mu   sync.Mutex
	subs map[string]domain.PushSubscription
}

func (m *memPushRepo) Save(_ context.Context, sub domain.PushSubscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subs[sub.Endpoint] = sub
	return nil
}

func (m *memPushRepo) DeleteByEndpoint(_ context.Context, endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subs, endpoint)
	return nil
}

func (m *memPushRepo) ListAll(context.Context) ([]domain.PushSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.PushSubscription, 0, len(m.subs))
	for _, s := range m.subs {
		out = append(out, s)
	}
	return out, nil
}

func TestWebPushNotifier_CleansUpGoneSubscriptions(t *testing.T) {
	repo := &memPushRepo{
		subs: map[string]domain.PushSubscription{
			"https://push/active": {Endpoint: "https://push/active", P256dh: "p1", Auth: "a1"},
			"https://push/gone":   {Endpoint: "https://push/gone", P256dh: "p2", Auth: "a2"},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	notifier := NewWebPushNotifier(repo, "pub", "priv", "mailto:test@example.com", logger)
	notifier.sendFn = func(_ context.Context, payload []byte, sub *webpush.Subscription, _ *webpush.Options) (*http.Response, error) {
		if !strings.Contains(string(payload), "Rp 5.500") {
			t.Fatalf("unexpected payload: %s", string(payload))
		}
		code := http.StatusCreated
		if sub.Endpoint == "https://push/gone" {
			code = http.StatusGone
		}
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	err := notifier.NotifyPriceChanges(context.Background(), []domain.PriceChange{
		{
			Medicine:   domain.Medicine{ID: "1", Name: "Paracetamol 500mg", Price: 5500},
			OldPrice:   5000,
			NewPrice:   5500,
			ChangeType: domain.ChangePriceUpdated,
		},
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}

	remaining, _ := repo.ListAll(context.Background())
	if len(remaining) != 1 || remaining[0].Endpoint != "https://push/active" {
		t.Fatalf("expected 410 subscription removed, remaining=%+v", remaining)
	}
}
