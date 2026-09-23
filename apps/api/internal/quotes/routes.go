package quotes

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermQuotesRead)).Get("/", h.List)
	r.With(az.Require(auth.PermQuotesCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermQuotesRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermQuotesUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermQuotesApprove)).Post("/approve", h.Approve)
	})
	return r
}
