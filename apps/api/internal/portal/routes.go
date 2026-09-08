package portal

import "github.com/go-chi/chi/v5"

func Routes(h *Handler, az *Authorizer) chi.Router {
	r := chi.NewRouter()
	r.Post("/auth/login", h.Login)
	r.Post("/auth/logout", h.Logout)
	r.Group(func(r chi.Router) {
		r.Use(az.Authenticate)
		r.Get("/me", h.Me)
		r.Patch("/me/password", h.ChangePassword)
		r.Get("/bookings", h.Bookings)
		r.Post("/bookings/{id}/feedback", h.Feedback)
	})
	return r
}