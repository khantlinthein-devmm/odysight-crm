package payments

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/payments
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.List(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]PaymentDTO, 0, len(items))
	for _, b := range items {
		dtos = append(dtos, toDTO(b))
	}
	response.JSON(w, http.StatusOK, pagination.Page[PaymentDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// Get handles GET /api/v1/payments/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	payment, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(payment))
}

// Create handles POST /api/v1/payments
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreatePaymentRequest](w, r)
	if !ok {
		return
	}

	payment, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(payment))
}

// Update handles PATCH /api/v1/payments/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdatePaymentRequest](w, r)
	if !ok {
		return
	}

	payment, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(payment))
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid payment id")
		return 0, false
	}
	return id, true
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var out T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return out, false
	}
	if dec.More() {
		response.Error(w, http.StatusBadRequest, "invalid JSON body: trailing data")
		return out, false
	}
	return out, true
}
