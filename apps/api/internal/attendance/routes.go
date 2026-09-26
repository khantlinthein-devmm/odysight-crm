package attendance

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	// Staff with attendance permissions act on anyone. A CLEANER lacks them
	// but may read and record their own attendance; the handler narrows
	// those requests to the caller's own cleaner profile.
	r.With(h.scope(auth.PermAttendanceRead)).Get("/", h.List)
	r.With(h.scope(auth.PermAttendanceManage)).Post("/check-in", h.CheckIn)
	r.With(h.scope(auth.PermAttendanceManage)).Post("/check-out", h.CheckOut)
	return r
}
