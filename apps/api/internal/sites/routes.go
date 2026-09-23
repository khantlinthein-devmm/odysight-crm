package sites

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermSitesRead)).Get("/", h.List)
	r.With(az.Require(auth.PermSitesCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermSitesRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermSitesUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermSitesDelete)).Delete("/", h.Delete)
	})
	return r
}
