package sites

import (
	"strings"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

type SiteDTO struct {
	ID          int64    `json:"id"`
	CustomerID  int64    `json:"customerId"`
	Name        string   `json:"name"`
	Address     string   `json:"address"`
	ContactName string   `json:"contactName"`
	Phone       string   `json:"phone"`
	Email       string   `json:"email"`
	Notes       string   `json:"notes"`
	Status      string   `json:"status"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	IsDefault   bool     `json:"isDefault"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

func toDTO(s Site) SiteDTO {
	return SiteDTO{
		ID:          s.ID,
		CustomerID:  s.CustomerID,
		Name:        s.Name,
		Address:     s.Address,
		ContactName: s.ContactName,
		Phone:       s.Phone,
		Email:       s.Email,
		Notes:       s.Notes,
		Status:      string(s.Status),
		Latitude:    s.Latitude,
		Longitude:   s.Longitude,
		IsDefault:   s.IsDefault,
		CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type CreateSiteRequest struct {
	CustomerID  int64    `json:"customerId"`
	Name        string   `json:"name"`
	Address     string   `json:"address"`
	ContactName string   `json:"contactName"`
	Phone       string   `json:"phone"`
	Email       string   `json:"email"`
	Notes       string   `json:"notes"`
	Status      string   `json:"status"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	IsDefault   bool     `json:"isDefault"`
}

func (r *CreateSiteRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Address = strings.TrimSpace(r.Address)
	r.ContactName = strings.TrimSpace(r.ContactName)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Notes = strings.TrimSpace(r.Notes)
	r.Status = strings.TrimSpace(r.Status)
	if r.Status == "" {
		r.Status = string(StatusActive)
	}

	if r.CustomerID < 1 {
		return response.NewAPIError(400, "customerId is required")
	}
	if r.Name == "" {
		return response.NewAPIError(400, "name is required")
	}
	if r.Address == "" {
		return response.NewAPIError(400, "address is required")
	}
	if r.Email != "" && !validate.Email(r.Email) {
		return response.NewAPIError(400, "email must be a valid address")
	}
	if r.Phone != "" && !validate.Phone(r.Phone) {
		return response.NewAPIError(400, "phone must be valid")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return response.NewAPIError(400, "latitude must be between -90 and 90")
	}
	if r.Longitude != nil && (*r.Longitude < -180 || *r.Longitude > 180) {
		return response.NewAPIError(400, "longitude must be between -180 and 180")
	}
	return nil
}

type UpdateSiteRequest struct {
	Name        *string  `json:"name"`
	Address     *string  `json:"address"`
	ContactName *string  `json:"contactName"`
	Phone       *string  `json:"phone"`
	Email       *string  `json:"email"`
	Notes       *string  `json:"notes"`
	Status      *string  `json:"status"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	IsDefault   *bool    `json:"isDefault"`
}

func (r *UpdateSiteRequest) Validate() error {
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return response.NewAPIError(400, "name cannot be empty")
	}
	if r.Address != nil && strings.TrimSpace(*r.Address) == "" {
		return response.NewAPIError(400, "address cannot be empty")
	}
	if r.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*r.Email))
		if email != "" && !validate.Email(email) {
			return response.NewAPIError(400, "email must be a valid address")
		}
		r.Email = &email
	}
	if r.Phone != nil {
		phone := strings.TrimSpace(*r.Phone)
		if phone != "" && !validate.Phone(phone) {
			return response.NewAPIError(400, "phone must be valid")
		}
		r.Phone = &phone
	}
	if r.Status != nil && !Status(strings.TrimSpace(*r.Status)).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return response.NewAPIError(400, "latitude must be between -90 and 90")
	}
	if r.Longitude != nil && (*r.Longitude < -180 || *r.Longitude > 180) {
		return response.NewAPIError(400, "longitude must be between -180 and 180")
	}
	return nil
}

func (r *UpdateSiteRequest) IsEmpty() bool {
	return r.Name == nil && r.Address == nil && r.ContactName == nil &&
		r.Phone == nil && r.Email == nil && r.Notes == nil &&
		r.Status == nil && r.Latitude == nil && r.Longitude == nil &&
		r.IsDefault == nil
}
