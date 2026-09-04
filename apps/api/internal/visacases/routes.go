package visacases

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermVisaCasesRead)).Get("/", h.List)
	r.With(az.Require(auth.PermVisaCasesCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermVisaCasesRead))
		r.With(az.Require(auth.PermVisaCasesUpdate)).Patch("/", h.Update)
	})
	return r
}
