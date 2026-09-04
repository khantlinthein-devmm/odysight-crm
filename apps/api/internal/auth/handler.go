package auth

import (
	"net/http"

	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[LoginRequest](w, r)
	if !ok {
		return
	}

	token, user, err := h.service.Login(r.Context(), req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}
