package customers

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermCustomersRead)).Get("/", h.List)
	r.With(az.Require(auth.PermCustomersCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermCustomersRead))
		r.With(az.Require(auth.PermCustomersUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermCustomersDelete)).Delete("/", h.Delete)
	})
	return r
}
