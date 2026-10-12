package settings

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/odysight/crm/internal/auth"
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
	if ident, ok := auth.IdentityFromContext(r.Context()); ok && ident.Role == auth.RoleCleaner {
		all = withoutPrices(all)
	}
	response.JSON(w, http.StatusOK, all)
}

// withoutPrices blanks the service catalog prices: cleaners need service
// names and durations, not what the company charges.
func withoutPrices(all map[string]json.RawMessage) map[string]json.RawMessage {
	raw, ok := all[KeyServices]
	if !ok {
		return all
	}
	var items []ServiceItem
	if err := json.Unmarshal(raw, &items); err != nil {
		delete(all, KeyServices)
		return all
	}
	for i := range items {
		items[i].BasePrice = 0
		items[i].PricePerSqm = 0
	}
	if b, err := json.Marshal(items); err == nil {
		all[KeyServices] = b
	} else {
		delete(all, KeyServices)
	}
	return all
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
