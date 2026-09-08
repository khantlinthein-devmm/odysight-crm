package reports

import (
	"net/http"
	"time"

	"github.com/odysight/crm/pkg/response"
)

// Financial handles GET /api/v1/reports/financial
func (h *Handler) Financial(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		from = now.AddDate(0, -5, 0)
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		to = now
	}
	if from.After(to) {
		response.Error(w, http.StatusBadRequest, "from must be before to")
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = "THB"
	}

	report, err := h.service.GetFinancial(r.Context(), from, to, currency)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, report)
}