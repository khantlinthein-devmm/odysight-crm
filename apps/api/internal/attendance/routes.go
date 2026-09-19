package attendance

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermAttendanceRead)).Get("/", h.List)
	r.With(az.Require(auth.PermAttendanceManage)).Post("/check-in", h.CheckIn)
	r.With(az.Require(auth.PermAttendanceManage)).Post("/check-out", h.CheckOut)
	return r
}
