package contracts

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermContractsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermContractsCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermContractsRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermContractsUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermContractsUpdate)).Post("/renew", h.Renew)
		r.With(az.Require(auth.PermContractsDelete)).Delete("/", h.Delete)
	})
	return r
}
