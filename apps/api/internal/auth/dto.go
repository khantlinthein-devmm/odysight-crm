package auth

import (
	"strings"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
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
	if !validate.Email(r.Email) {
		return response.NewAPIError(400, "a valid email is required")
	}
	if r.Password == "" {
		return response.NewAPIError(400, "password is required")
	}
	return nil
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (r *ChangePasswordRequest) Validate() error {
	if r.CurrentPassword == "" {
		return response.NewAPIError(400, "currentPassword is required")
	}
	if len(r.NewPassword) < 8 {
		return response.NewAPIError(400, "newPassword must be at least 8 characters")
	}
	return nil
}
