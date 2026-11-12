package push

import (
	"context"
	"errors"
	"testing"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type stubPushRepo struct {
	saved []domain.PushSubscription
}

func (s *stubPushRepo) Save(_ context.Context, sub domain.PushSubscription) error {
	s.saved = append(s.saved, sub)
	return nil
}
func (s *stubPushRepo) DeleteByEndpoint(context.Context, string) error { return nil }
func (s *stubPushRepo) ListAll(context.Context) ([]domain.PushSubscription, error) {
	return s.saved, nil
}

func TestPushService_ValidatesHTTPSEndpoint(t *testing.T) {
	repo := &stubPushRepo{}
	svc := NewService(repo, nil)

	invalidEndpoints := []string{
		"http://127.0.0.1:8080/internal",
		"https://127.0.0.1/internal",
		"https://169.254.169.254/latest/meta-data/",
		"https://localhost:8443/push",
		"https://attacker.example.com/webhook",
		"https://user:pass@fcm.googleapis.com/fcm/send/abc",
	}
	for _, ep := range invalidEndpoints {
		err := svc.Subscribe(context.Background(), domain.PushSubscription{
			Endpoint: ep,
			P256dh:   "key",
			Auth:     "auth",
		})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for endpoint %q, got %v", ep, err)
		}
	}

	validEndpoints := []string{
		"https://fcm.googleapis.com/fcm/send/abc",
		"https://updates.push.services.mozilla.com/wpush/v2/xyz",
		"https://wns2-sg2p.notify.windows.com/w/?token=123",
		"https://web.push.apple.com/Q12345",
	}
	for _, ep := range validEndpoints {
		if err := svc.Subscribe(context.Background(), domain.PushSubscription{
			Endpoint: ep,
			P256dh:   "key",
			Auth:     "auth",
		}); err != nil {
			t.Fatalf("expected valid endpoint %q to succeed, got %v", ep, err)
		}
	}
	if len(repo.saved) != len(validEndpoints) {
		t.Fatalf("expected %d saved subscriptions, got %d", len(validEndpoints), len(repo.saved))
	}
}
