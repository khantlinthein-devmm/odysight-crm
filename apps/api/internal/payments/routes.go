package payments

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermPaymentsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermPaymentsCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermPaymentsRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermPaymentsUpdate)).Patch("/", h.Update)
	})
	return r
}
