package cleaners

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermCleanersRead)).Get("/", h.List)
	r.With(az.Require(auth.PermCleanersCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermCleanersRead))
		r.With(az.Require(auth.PermCleanersUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermCleanersDelete)).Delete("/", h.Delete)
	})
	return r
}
