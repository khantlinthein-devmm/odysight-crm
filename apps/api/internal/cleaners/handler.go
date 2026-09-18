package cleaners

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.List(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]CleanerDTO, 0, len(items))
	for _, b := range items {
		dtos = append(dtos, toDTO(b))
	}
	response.JSON(w, http.StatusOK, pagination.Page[CleanerDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	cleaner, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(cleaner))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateCleanerRequest](w, r)
	if !ok {
		return
	}

	cleaner, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(cleaner))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateCleanerRequest](w, r)
	if !ok {
		return
	}

	cleaner, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(cleaner))
}

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

// Me handles GET /api/v1/cleaners/me — the logged-in cleaner's own profile.
// Runs on the outer Authenticate middleware only, so Role CLEANER (which has
// no broad cleaners.read permission) can still reach its own profile.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	cleaner, err := h.service.Me(r.Context(), id.UserID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(cleaner))
}

// UpdateLocation handles PATCH /api/v1/cleaners/me/location — GPS ping +
// online toggle from the cleaner mobile app.
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	req, ok := decodeJSON[LocationUpdateRequest](w, r)
	if !ok {
		return
	}
	cleaner, err := h.service.UpdateLocation(r.Context(), id.UserID, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(cleaner))
}

// PhonesList handles GET /api/v1/cleaners/{id}/phones.
func (h *Handler) PhonesList(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	phones, err := h.service.ListPhones(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	if phones == nil {
		phones = []PhoneNumber{}
	}
	response.JSON(w, http.StatusOK, phones)
}

// AddPhone handles POST /api/v1/cleaners/{id}/phones.
func (h *Handler) AddPhone(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[AddPhoneRequest](w, r)
	if !ok {
		return
	}

	phone, err := h.service.AddPhone(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, phone)
}

// RemovePhone handles DELETE /api/v1/cleaners/{id}/phones/{phoneId}.
func (h *Handler) RemovePhone(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	raw := chi.URLParam(r, "phoneId")
	phoneID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || phoneID < 1 {
		response.Error(w, http.StatusBadRequest, "invalid phone id")
		return
	}

	if err := h.service.RemovePhone(r.Context(), id, phoneID); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid cleaner id")
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
