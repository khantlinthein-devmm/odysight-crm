package bookings

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermBookingsRead)).Get("/", h.List)
	// Cleaner mobile pool. Static route wins over /{id} in chi, but it is
	// registered first for clarity.
	r.With(az.Require(auth.PermBookingsRead)).Get("/available", h.Available)
	r.With(az.Require(auth.PermBookingsCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermBookingsRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermBookingsUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermBookingsDelete)).Delete("/", h.Delete)
		r.With(az.Require(auth.PermBookingsUpdate)).Post("/accept", h.Accept)
	})
	return r
}
