package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/usecase/auth"
)

type AuthHandler struct {
	svc          *auth.Service
	cookieSecure bool
	logger       *slog.Logger
}

func NewAuthHandler(svc *auth.Service, cookieSecure bool, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		svc:          svc,
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type authResponse struct {
	Admin     domain.Admin `json:"admin"`
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`
	CSRFToken string       `json:"csrf_token"`
}

func (h *AuthHandler) LoginGoogle(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, h.logger, err)
		return
	}

	res, err := h.svc.LoginWithGoogle(r.Context(), req.IDToken)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("admin google login failed", "ip", clientIP(r), "error", err)
		}
		writeError(w, h.logger, err)
		return
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		writeError(w, h.logger, err)
		return
	}

	maxAge := int(time.Until(res.ExpiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    res.Token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrfToken,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, authResponse{
		Admin:     res.Admin,
		ExpiresAt: &res.ExpiresAt,
		CSRFToken: csrfToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	admin, ok := AdminFromContext(r.Context())
	if !ok {
		writeError(w, h.logger, domain.ErrUnauthorized)
		return
	}

	var csrfToken string
	if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
		csrfToken = c.Value
	} else {
		var genErr error
		csrfToken, genErr = generateCSRFToken()
		if genErr != nil {
			writeError(w, h.logger, genErr)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     csrfCookieName,
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			Secure:   h.cookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	writeJSON(w, http.StatusOK, authResponse{
		Admin:     admin,
		CSRFToken: csrfToken,
	})
}
