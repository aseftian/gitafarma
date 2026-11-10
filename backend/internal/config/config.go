package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config menampung seluruh konfigurasi runtime backend dari environment variables.
type Config struct {
	AppEnv                  string
	HTTPAddr                string
	CORSAllowedOrigins      []string
	StorageDriver           string
	StoragePath             string
	GoogleCredentialsFile   string
	SpreadsheetID           string
	MedicinesSheetName      string
	AdminsSheetName         string
	SyncInterval            time.Duration
	GoogleOAuthClientID     string
	SessionSecret           string
	SessionTTL              time.Duration
	SessionCookieSecure     bool
	AdminWhitelistCacheTTL  time.Duration
	VAPIDPublicKey          string
	VAPIDPrivateKey         string
	VAPIDSubject            string
}

// Load membaca konfigurasi dari environment variables dengan nilai default yang aman.
func Load() (Config, error) {
	syncInterval, err := parseDurationEnv("SYNC_INTERVAL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := parseDurationEnv("SESSION_TTL", 8*time.Hour)
	if err != nil {
		return Config{}, err
	}
	whitelistTTL, err := parseDurationEnv("ADMIN_WHITELIST_CACHE_TTL", 6*time.Hour)
	if err != nil {
		return Config{}, err
	}
	cookieSecure, err := parseBoolEnv("SESSION_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	originsRaw := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	var origins []string
	for _, o := range strings.Split(originsRaw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	return Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		HTTPAddr:               getEnv("HTTP_ADDR", ":8080"),
		CORSAllowedOrigins:     origins,
		StorageDriver:          getEnv("STORAGE_DRIVER", "sqlite"),
		StoragePath:            getEnv("STORAGE_PATH", "./data/gitafarma.db"),
		GoogleCredentialsFile:  getEnv("GOOGLE_APPLICATION_CREDENTIALS", "./credentials/service-account.json"),
		SpreadsheetID:          os.Getenv("GSHEET_SPREADSHEET_ID"),
		MedicinesSheetName:     getEnv("GSHEET_MEDICINES_SHEET", "medicines"),
		AdminsSheetName:        getEnv("GSHEET_ADMINS_SHEET", "admins"),
		SyncInterval:           syncInterval,
		GoogleOAuthClientID:    os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		SessionSecret:          os.Getenv("SESSION_SECRET"),
		SessionTTL:             sessionTTL,
		SessionCookieSecure:    cookieSecure,
		AdminWhitelistCacheTTL: whitelistTTL,
		VAPIDPublicKey:         os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:        os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:           getEnv("VAPID_SUBJECT", "mailto:admin@gitafarma.local"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}

func parseBoolEnv(key string, fallback bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid bool for %s: %w", key, err)
	}
	return b, nil
}
