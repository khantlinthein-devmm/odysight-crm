package reports

import (
	"net/http"

	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Summary handles GET /api/v1/reports/summary
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetSummary(r.Context())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}
