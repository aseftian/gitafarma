package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ardisan/gitafarma/backend/internal/usecase/setting"
	syncuc "github.com/ardisan/gitafarma/backend/internal/usecase/sync"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	settingSvc *setting.Service
	syncSvc    *syncuc.Service
	logger     *slog.Logger
}

func NewAdminHandler(settingSvc *setting.Service, syncSvc *syncuc.Service, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		settingSvc: settingSvc,
		syncSvc:    syncSvc,
		logger:     logger,
	}
}

func (h *AdminHandler) ListSettings(w http.ResponseWriter, r *http.Request) {
	list, err := h.settingSvc.List(r.Context())
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

type updateSettingRequest struct {
	Value string `json:"value"`
}

func (h *AdminHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	admin, _ := AdminFromContext(r.Context())
	key := chi.URLParam(r, "key")

	var req updateSettingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, h.logger, err)
		return
	}

	updated, err := h.settingSvc.Update(r.Context(), admin.Email, key, req.Value)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *AdminHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	res, err := h.syncSvc.SyncAllManual(r.Context())
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		if l > 200 {
			l = 200
		}
		limit = l
	}
	logs, err := h.settingSvc.ListAuditLogs(r.Context(), limit)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": logs})
}
