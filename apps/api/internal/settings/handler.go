package settings

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Get handles GET /api/v1/settings
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	all, err := h.service.GetAll(r.Context())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, all)
}

// Update handles PATCH /api/v1/settings with a partial object of key -> value.
// Unknown keys are rejected by the service validator.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	limited := io.LimitReader(r.Body, 1<<20)
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(limited).Decode(&raw); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	updated, err := h.service.Update(r.Context(), raw)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, updated)
}
