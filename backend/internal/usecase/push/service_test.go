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

	err := svc.Subscribe(context.Background(), domain.PushSubscription{
		Endpoint: "http://127.0.0.1:8080/internal",
		P256dh:   "key",
		Auth:     "auth",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for non-https endpoint, got %v", err)
	}

	err = svc.Subscribe(context.Background(), domain.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		P256dh:   "key",
		Auth:     "auth",
	})
	if err != nil || len(repo.saved) != 1 {
		t.Fatalf("expected valid https subscription saved, err=%v", err)
	}
}
