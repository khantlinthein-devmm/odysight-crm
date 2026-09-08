package users

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermUsersRead)).Get("/", h.List)
	r.With(az.Require(auth.PermUsersManage)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermUsersRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermUsersManage)).Patch("/", h.Update)
		r.With(az.Require(auth.PermUsersManage)).Post("/reset-password", h.ResetPassword)
		r.With(az.Require(auth.PermUsersManage)).Delete("/", h.Delete)
	})
	return r
}
