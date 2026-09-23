package checklists

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermChecklistsRead)).Get("/templates", h.ListTemplates)
	r.With(az.Require(auth.PermChecklistsManage)).Post("/templates", h.CreateTemplate)
	r.With(az.Require(auth.PermChecklistsRead)).Get("/bookings/{bookingId}", h.GetByBooking)
	r.With(az.Require(auth.PermChecklistsManage)).Post("/", h.Create)
	r.With(az.Require(auth.PermChecklistsManage)).Patch("/items/{itemId}", h.CompleteItem)
	r.With(az.Require(auth.PermChecklistsManage)).Post("/items/{itemId}/photo", h.UploadPhoto)
	r.With(az.Require(auth.PermChecklistsManage)).Post("/bookings/{bookingId}/confirm", h.Confirm)
	r.With(az.Require(auth.PermChecklistsRead)).Get("/photos/{file}", h.DownloadPhoto)
	return r
}
