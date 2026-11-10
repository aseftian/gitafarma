package http

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/usecase/auth"
)

const (
	sessionCookieName = "gf_session"
	csrfCookieName    = "gf_csrf"
	csrfHeaderName    = "X-CSRF-Token"
)

type contextKey string

const adminContextKey contextKey = "admin_user"

// AdminFromContext mengambil identitas admin yang telah terautentikasi dari context.
func AdminFromContext(ctx context.Context) (domain.Admin, bool) {
	admin, ok := ctx.Value(adminContextKey).(domain.Admin)
	return admin, ok
}

func RequireAdmin(authSvc *auth.Service, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				writeError(w, logger, domain.ErrUnauthorized)
				return
			}

			admin, err := authSvc.AuthenticateSession(r.Context(), cookie.Value)
			if err != nil {
				writeError(w, logger, err)
				return
			}

			ctx := context.WithValue(r.Context(), adminContextKey, admin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireCSRF() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(csrfCookieName)
			header := r.Header.Get(csrfHeaderName)
			if err != nil || cookie.Value == "" || header == "" {
				writeJSON(w, http.StatusForbidden, errorEnvelope{Error: "CSRF token tidak valid"})
				return
			}
			if len(cookie.Value) != len(header) || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				writeJSON(w, http.StatusForbidden, errorEnvelope{Error: "CSRF token tidak cocok"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
