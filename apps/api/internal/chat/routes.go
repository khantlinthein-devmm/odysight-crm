package chat

import (
	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
)

// Routes serves /chat. Every signed-in staff member can chat; managing the
// chat groups needs users.manage.
func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.Get("/contacts", h.Contacts)
	r.Get("/unread", h.Unread)
	r.Get("/conversations", h.Conversations)
	r.Post("/conversations", h.Open)
	r.Get("/conversations/{id}/messages", h.Messages)
	r.Post("/conversations/{id}/messages", h.Send)
	r.Post("/conversations/{id}/read", h.Read)
	r.Post("/conversations/{id}/typing", h.Typing)
	r.Get("/events", h.Stream)
	r.Get("/files/{name}", h.File)
	r.Get("/push/key", h.PushKey)
	r.Post("/push/subscribe", h.Subscribe)
	r.Post("/push/unsubscribe", h.Unsubscribe)
	r.With(az.Require(auth.PermUsersRead)).Get("/groups", h.Groups)
	r.With(az.Require(auth.PermUsersManage)).Post("/groups", h.CreateGroup)
	r.With(az.Require(auth.PermUsersManage)).Patch("/groups/{id}", h.UpdateGroup)
	r.With(az.Require(auth.PermUsersManage)).Delete("/groups/{id}", h.DeleteGroup)
	return r
}
