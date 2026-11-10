package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

type LoginResult struct {
	Admin     domain.Admin `json:"admin"`
	Token     string       `json:"-"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// Service menangani login Google SSO admin dan validasi ulang sesi terhadap whitelist.
type Service struct {
	verifier  port.GoogleIDTokenVerifier
	whitelist port.AdminWhitelistRepository
	sessions  port.SessionManager
	audits    port.AuditLogRepository
	nowFn     func() time.Time
}

func NewService(
	verifier port.GoogleIDTokenVerifier,
	whitelist port.AdminWhitelistRepository,
	sessions port.SessionManager,
	audits port.AuditLogRepository,
) *Service {
	return &Service{
		verifier:  verifier,
		whitelist: whitelist,
		sessions:  sessions,
		audits:    audits,
		nowFn:     time.Now,
	}
}

func (s *Service) LoginWithGoogle(ctx context.Context, idToken string) (LoginResult, error) {
	identity, err := s.verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return LoginResult{}, err
	}
	if !identity.EmailVerified {
		return LoginResult{}, domain.ErrEmailNotVerified
	}

	admin, err := s.whitelist.FindByEmail(ctx, identity.Email)
	if err != nil {
		return LoginResult{}, err
	}
	if !admin.Active {
		return LoginResult{}, domain.ErrAdminNotAllowlist
	}
	if admin.Name == "" {
		admin.Name = identity.Name
	}

	token, exp, err := s.sessions.Issue(admin)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue session: %w", err)
	}

	if s.audits != nil {
		_ = s.audits.Record(ctx, domain.AuditLog{
			AdminEmail: admin.Email,
			Action:     "login",
			EntityType: "auth",
			EntityID:   admin.Email,
			Detail:     "Google SSO login",
			CreatedAt:  s.nowFn().UTC(),
		})
	}

	return LoginResult{
		Admin:     admin,
		Token:     token,
		ExpiresAt: exp,
	}, nil
}

func (s *Service) AuthenticateSession(ctx context.Context, token string) (domain.Admin, error) {
	claims, err := s.sessions.Verify(token)
	if err != nil {
		return domain.Admin{}, err
	}

	admin, err := s.whitelist.FindByEmail(ctx, claims.Email)
	if err != nil {
		return domain.Admin{}, err
	}
	if !admin.Active {
		return domain.Admin{}, domain.ErrAdminNotAllowlist
	}
	if admin.Name == "" {
		admin.Name = claims.Name
	}
	return admin, nil
}
