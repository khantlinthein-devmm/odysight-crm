package applicants

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermApplicantsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermApplicantsCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermApplicantsRead))
		r.With(az.Require(auth.PermApplicantsUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermApplicantsDelete)).Delete("/", h.Delete)
	})
	return r
}
