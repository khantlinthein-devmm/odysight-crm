package leads

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermLeadsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermLeadsCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermLeadsRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermLeadsUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermCustomersCreate)).Post("/convert", h.Convert)
		r.With(az.Require(auth.PermLeadsDelete)).Delete("/", h.Delete)
	})
	return r
}
