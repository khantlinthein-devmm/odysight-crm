package audit

import (
	"net/http"

	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/audit-logs
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, nil)
	items, total, err := h.service.List(r.Context(), params)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}

	dtos := make([]EntryDTO, 0, len(items))
	for _, e := range items {
		dtos = append(dtos, toDTO(e))
	}
	response.JSON(w, http.StatusOK, pagination.Page[EntryDTO]{Data: dtos, Total: total, Limit: params.Limit, Offset: params.Offset})
}
