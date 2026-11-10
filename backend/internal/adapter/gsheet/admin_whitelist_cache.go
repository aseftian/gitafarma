package gsheet

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

// CachedAdminWhitelistRepo men-cache whitelist admin dari Google Sheet di memori dengan TTL configurable (default 6 jam).
type CachedAdminWhitelistRepo struct {
	gateway port.AdminWhitelistGateway
	ttl     time.Duration
	nowFn   func() time.Time

	mu        sync.RWMutex
	refreshMu sync.Mutex
	admins    map[string]domain.Admin
	expiresAt time.Time
	loaded    bool
}

func NewCachedAdminWhitelistRepo(gateway port.AdminWhitelistGateway, ttl time.Duration) *CachedAdminWhitelistRepo {
	if ttl <= 0 {
		ttl = 6 * time.Hour
	}
	return &CachedAdminWhitelistRepo{
		gateway: gateway,
		ttl:     ttl,
		nowFn:   time.Now,
		admins:  make(map[string]domain.Admin),
	}
}

func (r *CachedAdminWhitelistRepo) FindByEmail(ctx context.Context, email string) (domain.Admin, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return domain.Admin{}, domain.ErrAdminNotAllowlist
	}

	if admin, ok, valid := r.lookupCached(normalized); valid {
		if !ok {
			return domain.Admin{}, domain.ErrAdminNotAllowlist
		}
		return admin, nil
	}

	if err := r.ensureFresh(ctx); err != nil {
		return domain.Admin{}, err
	}

	admin, ok, _ := r.lookupCached(normalized)
	if !ok {
		return domain.Admin{}, domain.ErrAdminNotAllowlist
	}
	return admin, nil
}

func (r *CachedAdminWhitelistRepo) Refresh(ctx context.Context) error {
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()
	return r.reloadLocked(ctx)
}

func (r *CachedAdminWhitelistRepo) lookupCached(email string) (domain.Admin, bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.loaded || r.nowFn().After(r.expiresAt) {
		return domain.Admin{}, false, false
	}
	admin, ok := r.admins[email]
	return admin, ok, true
}

func (r *CachedAdminWhitelistRepo) ensureFresh(ctx context.Context) error {
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()

	r.mu.RLock()
	stillValid := r.loaded && !r.nowFn().After(r.expiresAt)
	r.mu.RUnlock()
	if stillValid {
		return nil
	}
	return r.reloadLocked(ctx)
}

func (r *CachedAdminWhitelistRepo) reloadLocked(ctx context.Context) error {
	list, err := r.gateway.FetchAdmins(ctx)
	if err != nil {
		return err
	}
	next := make(map[string]domain.Admin, len(list))
	for _, a := range list {
		email := strings.ToLower(strings.TrimSpace(a.Email))
		if email == "" {
			continue
		}
		a.Email = email
		next[email] = a
	}

	r.mu.Lock()
	r.admins = next
	r.expiresAt = r.nowFn().Add(r.ttl)
	r.loaded = true
	r.mu.Unlock()
	return nil
}
