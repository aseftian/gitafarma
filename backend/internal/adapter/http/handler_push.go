package http

import (
	"log/slog"
	"net/http"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	pushuc "github.com/ardisan/gitafarma/backend/internal/usecase/push"
)

type PushHandler struct {
	svc    *pushuc.Service
	logger *slog.Logger
}

func NewPushHandler(svc *pushuc.Service, logger *slog.Logger) *PushHandler {
	return &PushHandler{svc: svc, logger: logger}
}

func (h *PushHandler) VAPIDPublicKey(w http.ResponseWriter, _ *http.Request) {
	key, enabled := h.svc.PublicKey()
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":    enabled,
		"public_key": key,
	})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
	Keys     *struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, h.logger, err)
		return
	}

	p256dh := req.P256dh
	authKey := req.Auth
	if req.Keys != nil {
		if req.Keys.P256dh != "" {
			p256dh = req.Keys.P256dh
		}
		if req.Keys.Auth != "" {
			authKey = req.Keys.Auth
		}
	}

	err := h.svc.Subscribe(r.Context(), domain.PushSubscription{
		Endpoint:  req.Endpoint,
		P256dh:    p256dh,
		Auth:      authKey,
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "subscribed"})
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req unsubscribeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, h.logger, err)
		return
	}
	if err := h.svc.Unsubscribe(r.Context(), req.Endpoint); err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
