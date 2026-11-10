package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

type errorEnvelope struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.ErrInvalidInput
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return domain.ErrInvalidInput
	}
	return nil
}

func writeError(w http.ResponseWriter, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: err.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		writeJSON(w, http.StatusUnauthorized, errorEnvelope{Error: "autentikasi diperlukan"})
	case errors.Is(err, domain.ErrEmailNotVerified):
		writeJSON(w, http.StatusForbidden, errorEnvelope{Error: "email google belum terverifikasi"})
	case errors.Is(err, domain.ErrAdminNotAllowlist), errors.Is(err, domain.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errorEnvelope{Error: "email tidak terdaftar atau tidak aktif sebagai admin"})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorEnvelope{Error: "data tidak ditemukan"})
	case errors.Is(err, domain.ErrAlreadyExists):
		writeJSON(w, http.StatusConflict, errorEnvelope{Error: "data sudah ada"})
	case errors.Is(err, domain.ErrSheetNotConfigured):
		writeJSON(w, http.StatusServiceUnavailable, errorEnvelope{Error: "integrasi Google Sheet belum dikonfigurasi"})
	default:
		if logger != nil {
			logger.Error("internal http error", "error", err)
		}
		writeJSON(w, http.StatusInternalServerError, errorEnvelope{Error: "terjadi kesalahan pada server"})
	}
}
