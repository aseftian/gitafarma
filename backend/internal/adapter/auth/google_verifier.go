package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	"google.golang.org/api/idtoken"
)

// GoogleVerifier memvalidasi Google ID Token menggunakan Google Identity public keys.
type GoogleVerifier struct {
	clientID string
}

func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{clientID: strings.TrimSpace(clientID)}
}

func (v *GoogleVerifier) VerifyIDToken(ctx context.Context, rawToken string) (port.VerifiedGoogleIdentity, error) {
	if v.clientID == "" {
		return port.VerifiedGoogleIdentity{}, fmt.Errorf("%w: GOOGLE_OAUTH_CLIENT_ID belum dikonfigurasi", domain.ErrUnauthorized)
	}
	if strings.TrimSpace(rawToken) == "" {
		return port.VerifiedGoogleIdentity{}, fmt.Errorf("%w: id_token wajib diisi", domain.ErrInvalidInput)
	}

	payload, err := idtoken.Validate(ctx, rawToken, v.clientID)
	if err != nil {
		return port.VerifiedGoogleIdentity{}, fmt.Errorf("%w: token google tidak valid: %v", domain.ErrUnauthorized, err)
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return port.VerifiedGoogleIdentity{}, fmt.Errorf("%w: email tidak ditemukan di token google", domain.ErrUnauthorized)
	}

	return port.VerifiedGoogleIdentity{
		Email:         email,
		Name:          strings.TrimSpace(name),
		EmailVerified: verified,
	}, nil
}
