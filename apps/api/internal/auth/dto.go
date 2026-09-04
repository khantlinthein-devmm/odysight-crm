package auth

import (
	"strings"

	"github.com/odysight/crm/pkg/response"
)

type UserDTO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func toDTO(u User) UserDTO {
	return UserDTO{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
		Role:  string(u.Role),
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if r.Email == "" || !strings.Contains(r.Email, "@") {
		return response.NewAPIError(400, "a valid email is required")
	}
	if r.Password == "" {
		return response.NewAPIError(400, "password is required")
	}
	return nil
}
