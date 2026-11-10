package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	"github.com/ardisan/gitafarma/backend/internal/usecase/medicine"
	"github.com/go-chi/chi/v5"
)

type MedicineHandler struct {
	svc    *medicine.Service
	logger *slog.Logger
}

func NewMedicineHandler(svc *medicine.Service, logger *slog.Logger) *MedicineHandler {
	return &MedicineHandler{svc: svc, logger: logger}
}

func (h *MedicineHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, offset := parsePagination(q.Get("page"), q.Get("limit"))

	res, err := h.svc.Search(r.Context(), port.MedicineSearchQuery{
		Query:   q.Get("q"),
		Barcode: q.Get("barcode"),
		SKU:     q.Get("sku"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *MedicineHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, offset := parsePagination(q.Get("page"), q.Get("limit"))

	res, err := h.svc.ListRecent(r.Context(), limit, offset)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type medicineUpsertRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Barcode string `json:"barcode"`
	SKU     string `json:"sku"`
	Price   int64  `json:"price"`
	UOM     string `json:"uom"`
	AltUOM  string `json:"alt_uom"`
}

func (h *MedicineHandler) Create(w http.ResponseWriter, r *http.Request) {
	admin, _ := AdminFromContext(r.Context())
	var req medicineUpsertRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, h.logger, err)
		return
	}

	med, err := h.svc.Create(r.Context(), admin.Email, domain.Medicine{
		ID:      req.ID,
		Name:    req.Name,
		Barcode: req.Barcode,
		SKU:     req.SKU,
		Price:   req.Price,
		UOM:     req.UOM,
		AltUOM:  req.AltUOM,
	})
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, med)
}

func (h *MedicineHandler) Update(w http.ResponseWriter, r *http.Request) {
	admin, _ := AdminFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req medicineUpsertRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, h.logger, err)
		return
	}

	med, err := h.svc.Update(r.Context(), admin.Email, domain.Medicine{
		ID:      id,
		Name:    req.Name,
		Barcode: req.Barcode,
		SKU:     req.SKU,
		Price:   req.Price,
		UOM:     req.UOM,
		AltUOM:  req.AltUOM,
	})
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, med)
}

func (h *MedicineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	admin, _ := AdminFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), admin.Email, id); err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func parsePagination(pageStr, limitStr string) (limit, offset int) {
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	limit = 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		if l > 100 {
			l = 100
		}
		limit = l
	}
	offset = (page - 1) * limit
	return limit, offset
}
