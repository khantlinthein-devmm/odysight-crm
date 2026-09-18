package cleaners

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	// Self endpoints for the cleaner mobile app. They rely on the outer
	// Authenticate middleware only (no RBAC permission), so Role CLEANER can
	// reach its own profile without broad cleaners.read access.
	r.Get("/me", h.Me)
	r.Patch("/me/location", h.UpdateLocation)
	r.With(az.Require(auth.PermCleanersRead)).Get("/", h.List)
	r.With(az.Require(auth.PermCleanersCreate)).Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Use(az.Require(auth.PermCleanersRead))
		r.Get("/", h.Get)
		r.With(az.Require(auth.PermCleanersUpdate)).Patch("/", h.Update)
		r.With(az.Require(auth.PermCleanersDelete)).Delete("/", h.Delete)
		r.Get("/phones", h.PhonesList)
		r.With(az.Require(auth.PermCleanersUpdate)).Post("/phones", h.AddPhone)
		r.With(az.Require(auth.PermCleanersUpdate)).Delete("/phones/{phoneId}", h.RemovePhone)
	})
	return r
}
