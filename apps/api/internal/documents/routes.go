package documents

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermDocumentsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermDocumentsUpload)).Post("/", h.Upload)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermDocumentsRead))
		r.With(az.Require(auth.PermDocumentsUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermDocumentsDelete)).Delete("/", h.Delete)
	})
	return r
}
