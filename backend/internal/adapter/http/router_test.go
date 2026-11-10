package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authadapter "github.com/ardisan/gitafarma/backend/internal/adapter/auth"
	"github.com/ardisan/gitafarma/backend/internal/adapter/gsheet"
	"github.com/ardisan/gitafarma/backend/internal/adapter/storage/sqlite"
	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	"github.com/ardisan/gitafarma/backend/internal/usecase/auth"
	"github.com/ardisan/gitafarma/backend/internal/usecase/medicine"
	pushuc "github.com/ardisan/gitafarma/backend/internal/usecase/push"
	"github.com/ardisan/gitafarma/backend/internal/usecase/setting"
	syncuc "github.com/ardisan/gitafarma/backend/internal/usecase/sync"
)

type fakeGoogleVerifier struct{}

func (fakeGoogleVerifier) VerifyIDToken(_ context.Context, token string) (port.VerifiedGoogleIdentity, error) {
	switch token {
	case "admin-token":
		return port.VerifiedGoogleIdentity{Email: "admin@gitafarma.id", Name: "Owner", EmailVerified: true}, nil
	case "stranger-token":
		return port.VerifiedGoogleIdentity{Email: "stranger@gmail.com", Name: "Stranger", EmailVerified: true}, nil
	default:
		return port.VerifiedGoogleIdentity{}, domain.ErrUnauthorized
	}
}

type fakeSheet struct {
	meds   []domain.Medicine
	admins []domain.Admin
}

func (f *fakeSheet) FetchAllMedicines(context.Context) ([]domain.Medicine, error) {
	return f.meds, nil
}
func (f *fakeSheet) UpsertMedicine(_ context.Context, med domain.Medicine) error {
	for i, m := range f.meds {
		if m.ID == med.ID {
			f.meds[i] = med
			return nil
		}
	}
	f.meds = append(f.meds, med)
	return nil
}
func (f *fakeSheet) DeleteMedicine(_ context.Context, id string) error {
	for i, m := range f.meds {
		if m.ID == id {
			f.meds = append(f.meds[:i], f.meds[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}
func (f *fakeSheet) FetchAdmins(context.Context) ([]domain.Admin, error) {
	return f.admins, nil
}

type fakePushNotifier struct{}

func (fakePushNotifier) Enabled() bool     { return true }
func (fakePushNotifier) PublicKey() string { return "test-vapid-pub" }
func (fakePushNotifier) NotifyPriceChanges(context.Context, []domain.PriceChange) error {
	return nil
}

type handlerRoundTripper struct {
	h http.Handler
}

func (rt handlerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rt.h.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func TestHTTPAPI_EndToEnd(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	medRepo := sqlite.NewMedicineRepository(db)
	setRepo := sqlite.NewSettingRepository(db)
	pushRepo := sqlite.NewPushSubscriptionRepository(db)
	auditRepo := sqlite.NewAuditLogRepository(db)
	_ = setRepo.EnsureDefaults(ctx, domain.DefaultSettings(time.Now().UTC()))

	sheet := &fakeSheet{
		meds: []domain.Medicine{
			{ID: "MED-1", Name: "Paracetamol 500mg", Barcode: "899111", SKU: "GF-1", Price: 5000, UOM: "Strip", AltUOM: "Tablet", UpdatedAt: time.Now().UTC()},
		},
		admins: []domain.Admin{
			{Email: "admin@gitafarma.id", Name: "Owner", Role: "admin", Active: true},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	whitelist := gsheet.NewCachedAdminWhitelistRepo(sheet, 6*time.Hour)
	sessionMgr, _ := authadapter.NewJWTSessionManager("integration-test-secret-32-bytes", 8*time.Hour)
	notifier := fakePushNotifier{}

	authSvc := auth.NewService(fakeGoogleVerifier{}, whitelist, sessionMgr, auditRepo)
	medSvc := medicine.NewService(medRepo, setRepo, sheet, auditRepo, notifier, logger)
	setSvc := setting.NewService(setRepo, auditRepo)
	syncSvc := syncuc.NewService(sheet, medRepo, setRepo, whitelist, notifier, 5*time.Minute, logger)
	pushSvc := pushuc.NewService(pushRepo, notifier)

	if _, err := syncSvc.SyncOnce(ctx); err != nil {
		t.Fatalf("initial sync: %v", err)
	}

	router := NewRouter(RouterDeps{
		Logger:      logger,
		MedicineSvc: medSvc,
		AuthSvc:     authSvc,
		SettingSvc:  setSvc,
		SyncSvc:     syncSvc,
		PushSvc:     pushSvc,
	})

	baseURL := "http://gitafarma.test"
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:       jar,
		Transport: handlerRoundTripper{h: router},
	}

	// 1. Public search works without auth
	resp, err := client.Get(baseURL + "/api/v1/medicines?q=parace")
	if err != nil {
		t.Fatalf("public search request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public search status=%d", resp.StatusCode)
	}
	var searchRes port.MedicinePage
	_ = json.NewDecoder(resp.Body).Decode(&searchRes)
	_ = resp.Body.Close()
	if searchRes.Total != 1 {
		t.Fatalf("expected 1 search hit, got %+v", searchRes)
	}

	// 2. Admin endpoint rejected without session
	unauthReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/admin/settings", nil)
	resp, err = client.Do(unauthReq)
	if err != nil {
		t.Fatalf("unauth request error: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthenticated, got %d", resp.StatusCode)
	}

	// 3. Stranger Google login rejected with 403
	resp, err = client.Post(baseURL+"/api/v1/auth/google", "application/json", strings.NewReader(`{"id_token":"stranger-token"}`))
	if err != nil {
		t.Fatalf("stranger login error: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-whitelisted email, got %d", resp.StatusCode)
	}

	// 4. Whitelisted admin Google login succeeds and sets cookies
	resp, err = client.Post(baseURL+"/api/v1/auth/google", "application/json", strings.NewReader(`{"id_token":"admin-token"}`))
	if err != nil {
		t.Fatalf("admin login error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 admin login, got %d", resp.StatusCode)
	}
	var loginBody struct {
		Admin     domain.Admin `json:"admin"`
		CSRFToken string       `json:"csrf_token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&loginBody)
	_ = resp.Body.Close()
	if loginBody.Admin.Email != "admin@gitafarma.id" || loginBody.CSRFToken == "" {
		t.Fatalf("unexpected login body: %+v", loginBody)
	}

	// 5. Mutating admin call without CSRF header rejected (403)
	createPayload := `{"name":"Vitamin C 500mg","barcode":"899222","sku":"GF-2","price":7500,"uom":"Strip","alt_uom":"Tablet"}`
	reqNoCSRF, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/admin/medicines", strings.NewReader(createPayload))
	reqNoCSRF.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(reqNoCSRF)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without CSRF header, got %d", resp.StatusCode)
	}

	// 6. Mutating admin call WITH CSRF header succeeds & writes through to sheet
	reqCreate, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/admin/medicines", strings.NewReader(createPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set(csrfHeaderName, loginBody.CSRFToken)
	resp, _ = client.Do(reqCreate)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("expected 201 created, got %d: %s", resp.StatusCode, string(body))
	}
	_ = resp.Body.Close()
	if len(sheet.meds) != 2 {
		t.Fatalf("expected write-through to sheet, len=%d", len(sheet.meds))
	}

	// 7. Update setting recent_days=14
	reqSet, _ := http.NewRequest(http.MethodPut, baseURL+"/api/v1/admin/settings/recent_days", bytes.NewBufferString(`{"value":"14"}`))
	reqSet.Header.Set("Content-Type", "application/json")
	reqSet.Header.Set(csrfHeaderName, loginBody.CSRFToken)
	resp, _ = client.Do(reqSet)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 update setting, got %d", resp.StatusCode)
	}

	resp, _ = client.Get(baseURL + "/api/v1/medicines/recent")
	var recentRes medicine.RecentResult
	_ = json.NewDecoder(resp.Body).Decode(&recentRes)
	_ = resp.Body.Close()
	if recentRes.RecentDays != 14 || recentRes.Total != 2 {
		t.Fatalf("unexpected recent result after setting update: %+v", recentRes)
	}

	// 8. Admin removed from sheet + manual sync refreshes whitelist -> subsequent request gets 403!
	sheet.admins = nil
	reqSync, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/admin/sync", nil)
	reqSync.Header.Set(csrfHeaderName, loginBody.CSRFToken)
	resp, _ = client.Do(reqSync)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 sync, got %d", resp.StatusCode)
	}

	resp, _ = client.Get(baseURL + "/api/v1/auth/me")
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 after admin removed from sheet, got %d", resp.StatusCode)
	}
}
