package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authadapter "github.com/ardisan/gitafarma/backend/internal/adapter/auth"
	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

type stubVerifier struct {
	identity port.VerifiedGoogleIdentity
	err      error
}

func (s *stubVerifier) VerifyIDToken(context.Context, string) (port.VerifiedGoogleIdentity, error) {
	return s.identity, s.err
}

type stubWhitelist struct {
	admins map[string]domain.Admin
}

func (s *stubWhitelist) FindByEmail(_ context.Context, email string) (domain.Admin, error) {
	a, ok := s.admins[email]
	if !ok {
		return domain.Admin{}, domain.ErrAdminNotAllowlist
	}
	return a, nil
}

func (s *stubWhitelist) Refresh(context.Context) error { return nil }

type stubAudit struct {
	logs []domain.AuditLog
}

func (s *stubAudit) Record(_ context.Context, entry domain.AuditLog) error {
	s.logs = append(s.logs, entry)
	return nil
}

func (s *stubAudit) ListRecent(context.Context, int) ([]domain.AuditLog, error) {
	return s.logs, nil
}

func TestAuthService_GoogleSSOWithWhitelist(t *testing.T) {
	ctx := context.Background()
	sessions, err := authadapter.NewJWTSessionManager("test-session-secret-123456", 8*time.Hour)
	if err != nil {
		t.Fatalf("new session mgr: %v", err)
	}

	whitelist := &stubWhitelist{
		admins: map[string]domain.Admin{
			"owner@gitafarma.id":    {Email: "owner@gitafarma.id", Name: "Owner", Role: "admin", Active: true},
			"inactive@gitafarma.id": {Email: "inactive@gitafarma.id", Name: "Ex", Role: "admin", Active: false},
		},
	}
	audits := &stubAudit{}

	// 1. Valid & active admin succeeds
	svc := NewService(&stubVerifier{
		identity: port.VerifiedGoogleIdentity{Email: "owner@gitafarma.id", Name: "Owner", EmailVerified: true},
	}, whitelist, sessions, audits)

	res, err := svc.LoginWithGoogle(ctx, "valid-id-token")
	if err != nil {
		t.Fatalf("expected login success, got %v", err)
	}
	if res.Token == "" || res.Admin.Email != "owner@gitafarma.id" {
		t.Fatalf("unexpected login result: %+v", res)
	}
	if len(audits.logs) != 1 {
		t.Fatalf("expected login audit log")
	}

	// AuthenticateSession passes while active
	admin, err := svc.AuthenticateSession(ctx, res.Token)
	if err != nil || admin.Email != "owner@gitafarma.id" {
		t.Fatalf("expected valid session, got %+v err=%v", admin, err)
	}

	// Removed from whitelist -> AuthenticateSession rejects with ErrAdminNotAllowlist (403)
	delete(whitelist.admins, "owner@gitafarma.id")
	if _, err := svc.AuthenticateSession(ctx, res.Token); !errors.Is(err, domain.ErrAdminNotAllowlist) {
		t.Fatalf("expected ErrAdminNotAllowlist after removal, got %v", err)
	}

	// 2. Unverified email rejected
	svcUnverified := NewService(&stubVerifier{
		identity: port.VerifiedGoogleIdentity{Email: "owner@gitafarma.id", EmailVerified: false},
	}, whitelist, sessions, audits)
	if _, err := svcUnverified.LoginWithGoogle(ctx, "tok"); !errors.Is(err, domain.ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}

	// 3. Inactive admin rejected
	svcInactive := NewService(&stubVerifier{
		identity: port.VerifiedGoogleIdentity{Email: "inactive@gitafarma.id", EmailVerified: true},
	}, whitelist, sessions, audits)
	if _, err := svcInactive.LoginWithGoogle(ctx, "tok"); !errors.Is(err, domain.ErrAdminNotAllowlist) {
		t.Fatalf("expected ErrAdminNotAllowlist for inactive, got %v", err)
	}

	// 4. Unknown email rejected
	svcUnknown := NewService(&stubVerifier{
		identity: port.VerifiedGoogleIdentity{Email: "stranger@gmail.com", EmailVerified: true},
	}, whitelist, sessions, audits)
	if _, err := svcUnknown.LoginWithGoogle(ctx, "tok"); !errors.Is(err, domain.ErrAdminNotAllowlist) {
		t.Fatalf("expected ErrAdminNotAllowlist for unknown, got %v", err)
	}
}
