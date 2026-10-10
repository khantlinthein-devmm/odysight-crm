package attendance

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	// Reading: attendance.read sees everyone; a CLEANER sees only their own.
	// Check-in/out: everyone records only their own attendance; recording it
	// for someone else needs attendance.manage (Super admin and Admin by
	// default, editable in Settings → Roles & permissions).
	r.With(h.scope(auth.PermAttendanceRead)).Get("/", h.List)
	r.With(h.checkScope).Post("/check-in", h.CheckIn)
	r.With(h.checkScope).Post("/check-out", h.CheckOut)
	return r
}
