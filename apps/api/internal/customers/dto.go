package customers

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type CustomerDTO struct {
	ID           int64     `json:"id"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	PropertyType string    `json:"propertyType"`
	Area         string    `json:"area"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
}

func toDTO(c Customer) CustomerDTO {
	return CustomerDTO{
		ID:           c.ID,
		FirstName:    c.FirstName,
		LastName:     c.LastName,
		Email:        c.Email,
		Phone:        c.Phone,
		Address:      c.Address,
		PropertyType: string(c.PropertyType),
		Area:         c.Area,
		Status:       string(c.Status),
		CreatedAt:    c.CreatedAt,
	}
}

type CreateCustomerRequest struct {
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	PropertyType string `json:"propertyType"`
	Area         string `json:"area"`
	Status       string `json:"status"`
}

func (r *CreateCustomerRequest) Validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Phone = strings.TrimSpace(r.Phone)
	r.Address = strings.TrimSpace(r.Address)
	r.PropertyType = strings.TrimSpace(r.PropertyType)
	r.Area = strings.TrimSpace(r.Area)

	if r.FirstName == "" {
		return response.NewAPIError(400, "firstName is required")
	}
	if r.LastName == "" {
		return response.NewAPIError(400, "lastName is required")
	}
	if r.Email == "" || !strings.Contains(r.Email, "@") {
		return response.NewAPIError(400, "a valid email is required")
	}
	if r.Phone == "" {
		return response.NewAPIError(400, "phone is required")
	}
	if r.Address == "" {
		return response.NewAPIError(400, "address is required")
	}
	if !PropertyType(r.PropertyType).Valid() {
		return response.NewAPIError(400, "invalid propertyType")
	}
	if r.Area == "" {
		return response.NewAPIError(400, "area is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdateCustomerRequest struct {
	FirstName    *string `json:"firstName"`
	LastName     *string `json:"lastName"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	Address      *string `json:"address"`
	PropertyType *string `json:"propertyType"`
	Area         *string `json:"area"`
	Status       *string `json:"status"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateCustomerRequest) Validate() error {
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
	if r.Address != nil && strings.TrimSpace(*r.Address) == "" {
		return response.NewAPIError(400, "address cannot be empty")
	}
	if r.PropertyType != nil && !PropertyType(*r.PropertyType).Valid() {
		return response.NewAPIError(400, "invalid propertyType")
	}
	if r.Area != nil && strings.TrimSpace(*r.Area) == "" {
		return response.NewAPIError(400, "area cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateCustomerRequest) IsEmpty() bool {
	return r.FirstName == nil && r.LastName == nil && r.Email == nil &&
		r.Phone == nil && r.Address == nil && r.PropertyType == nil &&
		r.Area == nil && r.Status == nil
}
