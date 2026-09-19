package expenses

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermExpensesRead)).Get("/", h.List)
	r.With(az.Require(auth.PermExpensesManage)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermExpensesManage))
		r.Patch("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}
