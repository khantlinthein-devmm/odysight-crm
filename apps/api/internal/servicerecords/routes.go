package servicerecords

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermServiceRecordsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermServiceRecordsCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermServiceRecordsRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermServiceRecordsUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermServiceRecordsDelete)).Delete("/", h.Delete)
	})
	return r
}
