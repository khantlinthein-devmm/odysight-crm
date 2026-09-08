package invoices

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermInvoicesRead)).Get("/", h.List)
	r.With(az.Require(auth.PermInvoicesCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermInvoicesRead))
		r.Get("/", h.Get)
		r.Get("/pdf", h.PDF)
		r.With(az.Require(auth.PermInvoicesUpdate)).Post("/email", h.Email)
		r.With(az.Require(auth.PermInvoicesUpdate)).Patch("/", h.Update)
	})
	return r
}