package cleaners

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type CleanerDTO struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Skills    string    `json:"skills"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(c Cleaner) CleanerDTO {
	return CleanerDTO{
		ID:        c.ID,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		Phone:     c.Phone,
		Email:     c.Email,
		Skills:    c.Skills,
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt,
	}
}

type CreateCleanerRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Skills    string `json:"skills"`
	Status    string `json:"status"`
}

func (r *CreateCleanerRequest) Validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Skills = strings.TrimSpace(r.Skills)

	if r.FirstName == "" {
		return response.NewAPIError(400, "firstName is required")
	}
	if r.LastName == "" {
		return response.NewAPIError(400, "lastName is required")
	}
	if r.Phone == "" {
		return response.NewAPIError(400, "phone is required")
	}
	if r.Email == "" || !strings.Contains(r.Email, "@") {
		return response.NewAPIError(400, "a valid email is required")
	}
	if r.Skills == "" {
		return response.NewAPIError(400, "skills is required")
	}
	if r.Status == "" {
		r.Status = string(StatusAvailable)
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdateCleanerRequest struct {
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	Skills    *string `json:"skills"`
	Status    *string `json:"status"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateCleanerRequest) Validate() error {
	if r.FirstName != nil && strings.TrimSpace(*r.FirstName) == "" {
		return response.NewAPIError(400, "firstName cannot be empty")
	}
	if r.LastName != nil && strings.TrimSpace(*r.LastName) == "" {
		return response.NewAPIError(400, "lastName cannot be empty")
	}
	if r.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*r.Email))
		if email == "" || !strings.Contains(email, "@") {
			return response.NewAPIError(400, "a valid email is required")
		}
		r.Email = &email
	}
	if r.Skills != nil && strings.TrimSpace(*r.Skills) == "" {
		return response.NewAPIError(400, "skills cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateCleanerRequest) IsEmpty() bool {
	return r.FirstName == nil && r.LastName == nil && r.Phone == nil &&
		r.Email == nil && r.Skills == nil && r.Status == nil
}
