package gsheet

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"google.golang.org/api/googleapi"
)

func TestParseMedicinesSheet(t *testing.T) {
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	rows := [][]any{
		{"id", "nama_obat", "barcode", "sku", "harga", "satuan", "satuan_alternatif", "updated_at"},
		{"MED-1", "Paracetamol 500mg", "8991111", "GF-001", "Rp 5.500", "Strip", "Tablet", "2026-11-09T12:00:00Z"},
		{"", "Amoxicillin 500mg", "8992222", "GF-002", float64(12000), "Strip"},
		{"", ""},
	}

	meds, err := ParseMedicinesSheet(rows, now)
	if err != nil {
		t.Fatalf("parse medicines: %v", err)
	}
	if len(meds) != 2 {
		t.Fatalf("expected 2 medicines, got %d", len(meds))
	}
	if meds[0].Price != 5500 || meds[0].UOM != "Strip" || meds[0].AltUOM != "Tablet" {
		t.Fatalf("unexpected meds[0]: %+v", meds[0])
	}
	if meds[1].ID != "sku:GF-002" || meds[1].Price != 12000 || !meds[1].UpdatedAt.Equal(now) {
		t.Fatalf("unexpected meds[1]: %+v", meds[1])
	}
}

func TestParsePriceVariants(t *testing.T) {
	cases := map[string]int64{
		"15000":      15000,
		"15.000":     15000,
		"Rp 15.000":  15000,
		"15000.00":   15000,
		"1,250,000":  1250000,
		"1.250.000":  1250000,
		"Rp 2.500,00": 2500,
	}
	for input, want := range cases {
		got, err := ParsePrice(input)
		if err != nil || got != want {
			t.Fatalf("ParsePrice(%q) = (%d, %v), want %d", input, got, err, want)
		}
	}
}

func TestParseAdminsSheet(t *testing.T) {
	rows := [][]any{
		{"email", "name", "role", "active"},
		{"Owner@GitaFarma.id ", "Owner", "superadmin", "TRUE"},
		{"staff@gitafarma.id", "Staff", "", "false"},
		{"other@gitafarma.id", "Other", "admin", "aktif"},
	}

	admins, err := ParseAdminsSheet(rows)
	if err != nil {
		t.Fatalf("parse admins: %v", err)
	}
	if len(admins) != 3 {
		t.Fatalf("expected 3 admins, got %d", len(admins))
	}
	if admins[0].Email != "owner@gitafarma.id" || !admins[0].Active {
		t.Fatalf("unexpected admin[0]: %+v", admins[0])
	}
	if admins[1].Active || admins[1].Role != "admin" {
		t.Fatalf("unexpected admin[1]: %+v", admins[1])
	}
	if !admins[2].Active {
		t.Fatalf("expected aktif=true for admin[2]")
	}
}

type stubAdminGateway struct {
	calls atomic.Int32
	list  []domain.Admin
}

func (s *stubAdminGateway) FetchAdmins(context.Context) ([]domain.Admin, error) {
	s.calls.Add(1)
	return s.list, nil
}

func TestCachedAdminWhitelistRepo_6HourTTL(t *testing.T) {
	ctx := context.Background()
	gw := &stubAdminGateway{
		list: []domain.Admin{
			{Email: "admin@gitafarma.id", Name: "Admin", Role: "admin", Active: true},
		},
	}

	currentTime := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	repo := NewCachedAdminWhitelistRepo(gw, 6*time.Hour)
	repo.nowFn = func() time.Time { return currentTime }

	// First call loads from gateway
	a, err := repo.FindByEmail(ctx, "Admin@GitaFarma.id")
	if err != nil || !a.Active {
		t.Fatalf("expected active admin, got %+v err=%v", a, err)
	}
	if gw.calls.Load() != 1 {
		t.Fatalf("expected 1 gateway call, got %d", gw.calls.Load())
	}

	// Within 6h -> cache hit
	currentTime = currentTime.Add(5 * time.Hour)
	if _, err := repo.FindByEmail(ctx, "admin@gitafarma.id"); err != nil {
		t.Fatalf("expected cache hit: %v", err)
	}
	if gw.calls.Load() != 1 {
		t.Fatalf("expected cached hit without gateway call, got %d", gw.calls.Load())
	}

	// Past 6h TTL -> reloads and sees deactivation
	gw.list = []domain.Admin{{Email: "admin@gitafarma.id", Name: "Admin", Role: "admin", Active: false}}
	currentTime = currentTime.Add(2 * time.Hour)
	a, err = repo.FindByEmail(ctx, "admin@gitafarma.id")
	if err != nil || a.Active {
		t.Fatalf("expected reloaded inactive admin, got %+v err=%v", a, err)
	}
	if gw.calls.Load() != 2 {
		t.Fatalf("expected 2 gateway calls after 6h expiry, got %d", gw.calls.Load())
	}

	// Unknown email returns ErrAdminNotAllowlist
	if _, err := repo.FindByEmail(ctx, "unknown@gitafarma.id"); !errors.Is(err, domain.ErrAdminNotAllowlist) {
		t.Fatalf("expected ErrAdminNotAllowlist, got %v", err)
	}
}

func TestRetryWithBackoff(t *testing.T) {
	cfg := retryConfig{maxAttempts: 3, baseDelay: time.Millisecond, maxDelay: 5 * time.Millisecond}

	// Retries 429 quota error and succeeds on attempt 3
	var attempts int
	val, err := retryWithBackoff(context.Background(), cfg, func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", &googleapi.Error{Code: http.StatusTooManyRequests, Message: "quota"}
		}
		return "ok", nil
	})
	if err != nil || val != "ok" || attempts != 3 {
		t.Fatalf("expected success after 3 attempts, got val=%q err=%v attempts=%d", val, err, attempts)
	}

	// Does NOT retry 403 permission error
	attempts = 0
	_, err = retryWithBackoff(context.Background(), cfg, func() (string, error) {
		attempts++
		return "", &googleapi.Error{Code: http.StatusForbidden, Message: "forbidden"}
	})
	if err == nil || attempts != 1 {
		t.Fatalf("expected immediate failure without retry for 403, attempts=%d err=%v", attempts, err)
	}
}
