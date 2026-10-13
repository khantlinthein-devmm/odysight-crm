package reports

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

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
	out, err := h.service.BookingProfits(r.Context(), ids)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

// SaveBookingCosts handles PUT /api/v1/reports/booking-costs/{bookingId}.
func (h *Handler) SaveBookingCosts(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "bookingId"), 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "invalid booking id")
		return
	}
	var req SaveCostsRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	out, err := h.service.SaveBookingCosts(r.Context(), id, identity.UserID, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}
