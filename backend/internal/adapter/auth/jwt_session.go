package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	"github.com/golang-jwt/jwt/v5"
)

type sessionClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// JWTSessionManager menerbitkan dan memverifikasi JWT sesi admin (HS256).
type JWTSessionManager struct {
	secret []byte
	ttl    time.Duration
	issuer string
	nowFn  func() time.Time
}

func NewJWTSessionManager(secret string, ttl time.Duration) (*JWTSessionManager, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("%w: SESSION_SECRET minimal 16 karakter", domain.ErrInvalidInput)
	}
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}
	return &JWTSessionManager{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: "gitafarma-api",
		nowFn:  time.Now,
	}, nil
}

func (m *JWTSessionManager) Issue(admin domain.Admin) (string, time.Time, error) {
	now := m.nowFn().UTC()
	exp := now.Add(m.ttl)
	claims := sessionClaims{
		Email: admin.Email,
		Name:  admin.Name,
		Role:  admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   admin.Email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign session token: %w", err)
	}
	return signed, exp, nil
}

func (m *JWTSessionManager) Verify(tokenStr string) (port.SessionClaims, error) {
	if tokenStr == "" {
		return port.SessionClaims{}, domain.ErrUnauthorized
	}

	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		&sessionClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.secret, nil
		},
		jwt.WithTimeFunc(m.nowFn),
		jwt.WithIssuer(m.issuer),
	)
	if err != nil {
		return port.SessionClaims{}, fmt.Errorf("%w: %v", domain.ErrUnauthorized, err)
	}

	claims, ok := parsed.Claims.(*sessionClaims)
	if !ok || !parsed.Valid {
		return port.SessionClaims{}, domain.ErrUnauthorized
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return port.SessionClaims{
		Email:     claims.Email,
		Name:      claims.Name,
		Role:      claims.Role,
		ExpiresAt: expiresAt,
	}, nil
}

var _ port.SessionManager = (*JWTSessionManager)(nil)

func IsErrUnauthorized(err error) bool {
	return errors.Is(err, domain.ErrUnauthorized)
}
