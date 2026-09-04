package bookings

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermBookingsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermBookingsCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermBookingsRead))
		r.With(az.Require(auth.PermBookingsUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermBookingsDelete)).Delete("/", h.Delete)
	})
	return r
}
