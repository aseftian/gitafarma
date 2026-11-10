package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	authadapter "github.com/ardisan/gitafarma/backend/internal/adapter/auth"
	"github.com/ardisan/gitafarma/backend/internal/adapter/gsheet"
	httpadapter "github.com/ardisan/gitafarma/backend/internal/adapter/http"
	pushadapter "github.com/ardisan/gitafarma/backend/internal/adapter/push"
	"github.com/ardisan/gitafarma/backend/internal/adapter/storage/sqlite"
	"github.com/ardisan/gitafarma/backend/internal/config"
	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	"github.com/ardisan/gitafarma/backend/internal/usecase/auth"
	"github.com/ardisan/gitafarma/backend/internal/usecase/medicine"
	pushuc "github.com/ardisan/gitafarma/backend/internal/usecase/push"
	"github.com/ardisan/gitafarma/backend/internal/usecase/setting"
	syncuc "github.com/ardisan/gitafarma/backend/internal/usecase/sync"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runHealthcheck())
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	db, err := sqlite.Open(cfg.StoragePath)
	if err != nil {
		logger.Error("failed to initialize sqlite storage", "path", cfg.StoragePath, "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	medRepo := sqlite.NewMedicineRepository(db)
	setRepo := sqlite.NewSettingRepository(db)
	pushRepo := sqlite.NewPushSubscriptionRepository(db)
	auditRepo := sqlite.NewAuditLogRepository(db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := setRepo.EnsureDefaults(ctx, domain.DefaultSettings(time.Now().UTC())); err != nil {
		logger.Error("failed to seed default settings", "error", err)
		os.Exit(1)
	}

	var medSheet port.MedicineSheetGateway = gsheet.UnconfiguredGateway{}
	var adminSheet port.AdminWhitelistGateway = gsheet.UnconfiguredGateway{}
	if strings.TrimSpace(cfg.SpreadsheetID) != "" {
		client, err := gsheet.NewClient(
			ctx,
			cfg.GoogleCredentialsFile,
			cfg.SpreadsheetID,
			cfg.MedicinesSheetName,
			cfg.AdminsSheetName,
		)
		if err != nil {
			logger.Error("failed to initialize google sheets client", "error", err)
			os.Exit(1)
		}
		medSheet = client
		adminSheet = client
	} else {
		logger.Warn("GSHEET_SPREADSHEET_ID not set; Google Sheet sync & admin whitelist disabled")
	}

	sessionSecret := strings.TrimSpace(cfg.SessionSecret)
	if len(sessionSecret) < 16 {
		if cfg.AppEnv == "production" {
			logger.Error("SESSION_SECRET must be at least 16 characters in production")
			os.Exit(1)
		}
		buf := make([]byte, 32)
		_, _ = rand.Read(buf)
		sessionSecret = hex.EncodeToString(buf)
		logger.Warn("using ephemeral SESSION_SECRET for development")
	}

	sessionMgr, err := authadapter.NewJWTSessionManager(sessionSecret, cfg.SessionTTL)
	if err != nil {
		logger.Error("failed to initialize session manager", "error", err)
		os.Exit(1)
	}

	googleVerifier := authadapter.NewGoogleVerifier(cfg.GoogleOAuthClientID)
	whitelistRepo := gsheet.NewCachedAdminWhitelistRepo(adminSheet, cfg.AdminWhitelistCacheTTL)
	pushNotifier := pushadapter.NewWebPushNotifier(pushRepo, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject, logger)

	authSvc := auth.NewService(googleVerifier, whitelistRepo, sessionMgr, auditRepo)
	medSvc := medicine.NewService(medRepo, setRepo, medSheet, auditRepo, pushNotifier, logger)
	setSvc := setting.NewService(setRepo, auditRepo)
	syncSvc := syncuc.NewService(medSheet, medRepo, setRepo, whitelistRepo, pushNotifier, cfg.SyncInterval, logger)
	pushSvc := pushuc.NewService(pushRepo, pushNotifier)

	syncSvc.StartScheduler(ctx)

	router := httpadapter.NewRouter(httpadapter.RouterDeps{
		Logger:             logger,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		CookieSecure:       cfg.SessionCookieSecure,
		MedicineSvc:        medSvc,
		AuthSvc:            authSvc,
		SettingSvc:         setSvc,
		SyncSvc:            syncSvc,
		PushSvc:            pushSvc,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("http server starting", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down http server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}
}

func runHealthcheck() int {
	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" || strings.HasPrefix(addr, ":") {
		port := strings.TrimPrefix(addr, ":")
		if port == "" {
			port = "8080"
		}
		addr = "127.0.0.1:" + port
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
