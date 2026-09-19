package leads

import (
	"strings"
	"time"

	"github.com/odysight/crm/internal/customers"
	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

// DTOs use camelCase to match the frontend contract.

type LeadDTO struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Status    string    `json:"status"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(l Lead) LeadDTO {
	return LeadDTO{
		ID:        l.ID,
		FirstName: l.FirstName,
		LastName:  l.LastName,
		Email:     l.Email,
		Phone:     l.Phone,
		Status:    string(l.Status),
		Source:    string(l.Source),
		CreatedAt: l.CreatedAt,
	}
}

type CreateLeadRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	Source    string `json:"source"`
}

func (r *CreateLeadRequest) Validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Phone = strings.TrimSpace(r.Phone)

	if r.FirstName == "" {
		return response.NewAPIError(400, "firstName is required")
	}
	// Last name and email are optional: walk-in customers often have neither.
	if r.Email != "" && !validate.Email(r.Email) {
		return response.NewAPIError(400, "email must be a valid address")
	}
	if !validate.Phone(r.Phone) {
		return response.NewAPIError(400, "a valid phone is required")
	}

	status := Status(r.Status)
	if !status.Valid() {
		return response.NewAPIError(400, "invalid status")
	}

	source := Source(r.Source)
	if !source.Valid() {
		return response.NewAPIError(400, "invalid source")
	}

	return nil
}

type UpdateLeadRequest struct {
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Status    *string `json:"status"`
	Source    *string `json:"source"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateLeadRequest) Validate() error {
	if r.FirstName != nil && strings.TrimSpace(*r.FirstName) == "" {
		return response.NewAPIError(400, "firstName cannot be empty")
	}
	// Last name and email may be cleared: both are optional.
	if r.LastName != nil {
		lastName := strings.TrimSpace(*r.LastName)
		r.LastName = &lastName
	}
	if r.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*r.Email))
		if email != "" && !validate.Email(email) {
			return response.NewAPIError(400, "email must be a valid address")
		}
		r.Email = &email
	}
	if r.Phone != nil && !validate.Phone(*r.Phone) {
		return response.NewAPIError(400, "a valid phone is required")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.Source != nil && !Source(*r.Source).Valid() {
		return response.NewAPIError(400, "invalid source")
	}
	return nil
}

func (r *UpdateLeadRequest) IsEmpty() bool {
	return r.FirstName == nil && r.LastName == nil && r.Email == nil &&
		r.Phone == nil && r.Status == nil && r.Source == nil
}

type ConvertLeadRequest struct {
	Address      string `json:"address"`
	PropertyType string `json:"propertyType"`
	Area         string `json:"area"`
	Status       string `json:"status"`
}

func (r *ConvertLeadRequest) Validate() error {
	r.Address = strings.TrimSpace(r.Address)
	r.PropertyType = strings.TrimSpace(r.PropertyType)
	r.Area = strings.TrimSpace(r.Area)
	r.Status = strings.TrimSpace(r.Status)

	if r.Address == "" {
		return response.NewAPIError(400, "address is required")
	}
	if !customers.PropertyType(r.PropertyType).Valid() {
		return response.NewAPIError(400, "invalid propertyType")
	}
	if r.Area == "" {
		return response.NewAPIError(400, "area is required")
	}
	if !customers.Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}
