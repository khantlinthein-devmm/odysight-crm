package cleanerdocs

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

// Routes mounts at /cleaner-documents. Document contents need the dedicated
// cleaner_documents.* permissions; plain cleaners.read only reaches the
// type + expiry status view.
func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermCleanersRead)).Get("/status", h.Statuses)
	r.With(az.Require(auth.PermCleanerDocsRead)).Get("/expiring", h.Expiring)
	r.With(az.Require(auth.PermCleanerDocsRead)).Get("/", h.List)
	r.With(az.Require(auth.PermCleanerDocsManage)).Post("/", h.Create)
	r.Route("/{docId}", func(r chi.Router) {
		r.Use(az.Require(auth.PermCleanerDocsRead))
		r.Get("/", h.Get)
		r.Get("/file", h.File)
		r.With(az.Require(auth.PermCleanerDocsManage)).Patch("/", h.Update)
		r.With(az.Require(auth.PermCleanerDocsManage)).Delete("/", h.Delete)
	})
	return r
}
