package users

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

// List handles GET /api/v1/users
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.List(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]UserDTO, 0, len(items))
	for _, u := range items {
		dtos = append(dtos, toDTO(u))
	}
	response.JSON(w, http.StatusOK, pagination.Page[UserDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}

// Get handles GET /api/v1/users/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	u, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(u))
}

// Create handles POST /api/v1/users
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	req, ok := decodeJSON[CreateUserRequest](w, r)
	if !ok {
		return
	}

	u, err := h.service.Create(r.Context(), caller, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, toDTO(u))
}

// Update handles PATCH /api/v1/users/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[UpdateUserRequest](w, r)
	if !ok {
		return
	}

	u, err := h.service.Update(r.Context(), caller, id, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, toDTO(u))
}

// ResetPassword handles POST /api/v1/users/{id}/reset-password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[ResetPasswordRequest](w, r)
	if !ok {
		return
	}
	if err := req.Validate(); err != nil {
		response.HandleError(w, r, err)
		return
	}

	if err := h.service.ResetPassword(r.Context(), caller, id, req.NewPassword); err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Delete handles DELETE /api/v1/users/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), caller, id); err != nil {
		response.HandleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid user id")
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
