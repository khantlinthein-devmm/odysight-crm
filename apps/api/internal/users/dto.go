package users

import (
	"strings"
	"time"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

type UserDTO struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

func toDTO(u User) UserDTO {
	return UserDTO{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (r *CreateUserRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return response.NewAPIError(400, "name is required")
	}
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if !validate.Email(r.Email) {
		return response.NewAPIError(400, "a valid email is required")
	}
	if len(r.Password) < 8 {
		return response.NewAPIError(400, "password must be at least 8 characters")
	}
	if !auth.Role(r.Role).Valid() {
		return response.NewAPIError(400, "a valid role is required")
	}
	return nil
}

type UpdateUserRequest struct {
	Name *string `json:"name"`
	Role *string `json:"role"`
}

func (r *UpdateUserRequest) Validate() error {
	if r.Name != nil {
		v := strings.TrimSpace(*r.Name)
		*r.Name = v
		if v == "" {
			return response.NewAPIError(400, "name must not be empty")
		}
	}
	if r.Role != nil {
		if !auth.Role(*r.Role).Valid() {
			return response.NewAPIError(400, "a valid role is required")
		}
	}
	return nil
}

func (r *UpdateUserRequest) IsEmpty() bool {
	return r.Name == nil && r.Role == nil
}

type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

func (r *ResetPasswordRequest) Validate() error {
	if len(r.NewPassword) < 8 {
		return response.NewAPIError(400, "newPassword must be at least 8 characters")
	}
	return nil
}
