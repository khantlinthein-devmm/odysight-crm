package reports

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermReportsRead)).Get("/summary", h.Summary)
	r.With(az.Require(auth.PermReportsRead)).Get("/financial", h.Financial)
	r.With(az.Require(auth.PermReportsRead)).Get("/commercial", h.Commercial)
	r.With(az.Require(auth.PermReportsRead)).Get("/booking-profit", h.BookingProfit)
	// Job costs are entered by staff who can edit bookings AND see profit;
	// cleaners never reach them.
	r.With(az.Require(auth.PermReportsRead), az.Require(auth.PermBookingsUpdate)).Put("/booking-costs/{bookingId}", h.SaveBookingCosts)
	return r
}
