package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/response"
)

type LoginResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

func Routes(h *Handler, az *Authorizer, loginLimit func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	if loginLimit != nil {
		r.With(loginLimit).Post("/login", h.Login)
	} else {
		r.Post("/login", h.Login)
	}
	if az != nil {
		r.Group(func(r chi.Router) {
			r.Use(az.Authenticate)
			r.Get("/me", h.Me)
			r.Post("/logout", h.Logout)
			r.With(loginLimit).Post("/change-password", h.ChangePassword)
		})
	}
	return r
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var out T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return out, false
	}
	if dec.More() {
		response.Error(w, http.StatusBadRequest, "invalid JSON body: trailing data")
		return out, false
	}
	return out, true
}
