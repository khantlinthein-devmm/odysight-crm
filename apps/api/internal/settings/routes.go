package settings

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermSettingsRead)).Get("/", h.Get)
	r.With(az.Require(auth.PermSettingsManage)).Patch("/", h.Update)
	return r
}
