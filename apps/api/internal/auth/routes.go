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

func Routes(h *Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/login", h.Login)
	return r
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var out T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&out); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return out, false
	}
	return out, true
}
