package leads

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/customers"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/leads
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.List(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]LeadDTO, 0, len(items))
	for _, b := range items {
		dtos = append(dtos, toDTO(b))
	}
	response.JSON(w, http.StatusOK, pagination.Page[LeadDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// Get handles GET /api/v1/leads/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	lead, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(lead))
}

// Create handles POST /api/v1/leads
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateLeadRequest](w, r)
	if !ok {
		return
	}

	lead, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(lead))
}

// Update handles PATCH /api/v1/leads/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateLeadRequest](w, r)
	if !ok {
		return
	}

	lead, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(lead))
}

// Delete handles DELETE /api/v1/leads/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Convert handles POST /api/v1/leads/{id}/convert — turns a lead into a customer.
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[ConvertLeadRequest](w, r)
	if !ok {
		return
	}

	customer, err := h.service.Convert(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, customers.ToDTO(customer))
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid lead id")
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
