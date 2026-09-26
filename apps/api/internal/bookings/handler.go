package bookings

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
	if ident, ok := auth.IdentityFromContext(r.Context()); ok && ident.Role == auth.RoleCleaner {
		c, linked, err := h.service.cleanerFor(r.Context(), ident.UserID)
		if err != nil {
			response.HandleError(w, r, err)
			return
		}
		if !linked {
			response.JSON(w, http.StatusOK, pagination.Page[BookingDTO]{Data: []BookingDTO{}, Limit: params.Limit, Offset: params.Offset})
			return
		}
		params.CleanerID = c.ID
		params.Available = false
	}
	items, total, err := h.service.List(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]BookingDTO, 0, len(items))
	for _, b := range items {
		dtos = append(dtos, toDTO(b))
	}
	response.JSON(w, http.StatusOK, pagination.Page[BookingDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	booking, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	if ident, ok := auth.IdentityFromContext(r.Context()); ok && ident.Role == auth.RoleCleaner {
		c, linked, err := h.service.cleanerFor(r.Context(), ident.UserID)
		if err != nil {
			response.HandleError(w, r, err)
			return
		}
		if !inPool(booking) && (!linked || !assignedTo(booking, c)) {
			response.Error(w, http.StatusNotFound, "booking not found")
			return
		}
	}
	response.JSON(w, http.StatusOK, toDTO(booking))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[CreateBookingRequest](w, r)
	if !ok {
		return
	}

	booking, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(booking))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateBookingRequest](w, r)
	if !ok {
		return
	}

	if ident, ok := auth.IdentityFromContext(r.Context()); ok && ident.Role == auth.RoleCleaner {
		c, linked, err := h.service.cleanerFor(r.Context(), ident.UserID)
		if err != nil {
			response.HandleError(w, r, err)
			return
		}
		if !linked {
			response.Error(w, http.StatusForbidden, "no cleaner profile linked to this login; ask the office to link your account")
			return
		}
		current, err := h.service.Get(r.Context(), id)
		if err != nil {
			response.HandleError(w, r, err)
			return
		}
		if err := checkCleanerUpdate(current, c, req); err != nil {
			response.HandleError(w, r, err)
			return
		}
	}

	booking, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(booking))
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

// Available handles GET /api/v1/bookings/available — the unassigned job pool
// for the cleaner mobile app, scoped to the cleaner's own area.
func (h *Handler) Available(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	params := pagination.Parse(r, 20, nil)
	items, total, _, err := h.service.Available(r.Context(), id.UserID, params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	dtos := make([]BookingDTO, 0, len(items))
	for _, b := range items {
		dtos = append(dtos, toDTO(b))
	}
	response.JSON(w, http.StatusOK, pagination.Page[BookingDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// Accept handles POST /api/v1/bookings/{id}/accept — the logged-in cleaner
// takes a pending booking. First tap wins, late tappers get 409.
func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	bookingID, ok := parseID(w, r)
	if !ok {
		return
	}
	booking, err := h.service.Accept(r.Context(), id.UserID, bookingID)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(booking))
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid booking id")
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
