package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/usecase/auth"
	"github.com/ardisan/gitafarma/backend/internal/usecase/medicine"
	pushuc "github.com/ardisan/gitafarma/backend/internal/usecase/push"
	"github.com/ardisan/gitafarma/backend/internal/usecase/setting"
	syncuc "github.com/ardisan/gitafarma/backend/internal/usecase/sync"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type RouterDeps struct {
	Logger             *slog.Logger
	CORSAllowedOrigins []string
	CookieSecure       bool
	MedicineSvc        *medicine.Service
	AuthSvc            *auth.Service
	SettingSvc         *setting.Service
	SyncSvc            *syncuc.Service
	PushSvc            *pushuc.Service
}

// NewRouter membangun HTTP router utama beserta CORS, auth, CSRF, dan endpoint API v1.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(20 * time.Second))
	r.Use(SecurityHeaders())

	if len(deps.CORSAllowedOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   deps.CORSAllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", csrfHeaderName},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	medHandler := NewMedicineHandler(deps.MedicineSvc, deps.Logger)
	authHandler := NewAuthHandler(deps.AuthSvc, deps.CookieSecure, deps.Logger)
	adminHandler := NewAdminHandler(deps.SettingSvc, deps.SyncSvc, deps.Logger)
	pushHandler := NewPushHandler(deps.PushSvc, deps.Logger)

	publicLimiter := NewRateLimiter(120, 30)
	authLimiter := NewRateLimiter(15, 5)

	healthHandler := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}

	r.Get("/healthz", healthHandler)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", healthHandler)

		// Public employee endpoints (rate-limited per IP)
		api.Group(func(pub chi.Router) {
			pub.Use(publicLimiter.Middleware())
			pub.Get("/medicines", medHandler.Search)
			pub.Get("/medicines/recent", medHandler.ListRecent)
			pub.Get("/push/vapid-public-key", pushHandler.VAPIDPublicKey)
			pub.Post("/push/subscribe", pushHandler.Subscribe)
			pub.Post("/push/unsubscribe", pushHandler.Unsubscribe)
		})

		// Auth endpoints
		api.With(authLimiter.Middleware()).Post("/auth/google", authHandler.LoginGoogle)
		api.With(RequireCSRF()).Post("/auth/logout", authHandler.Logout)
		api.With(RequireAdmin(deps.AuthSvc, deps.Logger)).Get("/auth/me", authHandler.Me)

		// Protected Admin endpoints (session cookie + whitelist re-check + CSRF)
		api.Route("/admin", func(adm chi.Router) {
			adm.Use(RequireAdmin(deps.AuthSvc, deps.Logger))
			adm.Use(RequireCSRF())

			adm.Post("/medicines", medHandler.Create)
			adm.Put("/medicines/{id}", medHandler.Update)
			adm.Delete("/medicines/{id}", medHandler.Delete)
			adm.Get("/settings", adminHandler.ListSettings)
			adm.Put("/settings/{key}", adminHandler.UpdateSetting)
			adm.Post("/sync", adminHandler.TriggerSync)
			adm.Get("/audit-logs", adminHandler.ListAuditLogs)
		})
	})

	return r
}
