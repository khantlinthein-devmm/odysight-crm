package reports

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/response"
)

// BookingProfit handles GET /api/v1/reports/booking-profit?ids=1,2,3
func (h *Handler) BookingProfit(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("ids"))
	ids := []int64{}
	seen := map[int64]bool{}
	if raw != "" {
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil || id < 1 {
				response.Error(w, http.StatusBadRequest, "ids must be a comma-separated list of booking ids")
				return
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	if len(ids) > MaxProfitBookings {
		response.Error(w, http.StatusBadRequest, "at most 200 ids per request")
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	showRates := auth.HasPermission(identity.Role, auth.PermPayrollRead)
	out, err := h.service.BookingProfits(r.Context(), ids, showRates)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}
