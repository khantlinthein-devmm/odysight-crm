package sites

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

const (
	maxNameLen    = 120
	maxAddressLen = 500
	maxNotesLen   = 1000
)

type SiteDTO struct {
	ID           int64     `json:"id"`
	CustomerID   int64     `json:"customerId"`
	CustomerName string    `json:"customerName"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	Area         string    `json:"area"`
	PropertyType string    `json:"propertyType"`
	ContactName  string    `json:"contactName"`
	ContactPhone string    `json:"contactPhone"`
	ContactEmail string    `json:"contactEmail"`
	Notes        string    `json:"notes"`
	Status       string    `json:"status"`
	Lat          *float64  `json:"lat"`
	Lng          *float64  `json:"lng"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func toDTO(s Site) SiteDTO {
	return SiteDTO{
		ID:           s.ID,
		CustomerID:   s.CustomerID,
		CustomerName: s.CustomerName,
		Name:         s.Name,
		Address:      s.Address,
		Area:         s.Area,
		PropertyType: string(s.PropertyType),
		ContactName:  s.ContactName,
		ContactPhone: s.ContactPhone,
		ContactEmail: s.ContactEmail,
		Notes:        s.Notes,
		Status:       string(s.Status),
		Lat:          s.Lat,
		Lng:          s.Lng,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

type CreateSiteRequest struct {
	CustomerID   int64    `json:"customerId"`
	Name         string   `json:"name"`
	Address      string   `json:"address"`
	Area         string   `json:"area"`
	PropertyType string   `json:"propertyType"`
	ContactName  string   `json:"contactName"`
	ContactPhone string   `json:"contactPhone"`
	ContactEmail string   `json:"contactEmail"`
	Notes        string   `json:"notes"`
	Status       string   `json:"status"`
	Lat          *float64 `json:"lat"`
	Lng          *float64 `json:"lng"`
}

func (r *CreateSiteRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Address = strings.TrimSpace(r.Address)
	r.Area = strings.TrimSpace(r.Area)
	r.PropertyType = strings.TrimSpace(r.PropertyType)
	r.ContactName = strings.TrimSpace(r.ContactName)
	r.ContactPhone = strings.TrimSpace(r.ContactPhone)
	r.ContactEmail = strings.ToLower(strings.TrimSpace(r.ContactEmail))
	r.Notes = strings.TrimSpace(r.Notes)
	r.Status = strings.TrimSpace(r.Status)

	if r.CustomerID < 1 {
		return response.NewAPIError(400, "customerId is required")
	}
	if r.Name == "" {
		return response.NewAPIError(400, "name is required")
	}
	if r.PropertyType == "" {
		r.PropertyType = string(PropertyOther)
	}
	if r.Status == "" {
		r.Status = string(StatusActive)
	}
	return validateCommon(r.Name, r.Address, r.Notes, r.PropertyType, r.Status,
		r.ContactPhone, r.ContactEmail, r.Lat, r.Lng)
}

type UpdateSiteRequest struct {
	Name         *string  `json:"name"`
	Address      *string  `json:"address"`
	Area         *string  `json:"area"`
	PropertyType *string  `json:"propertyType"`
	ContactName  *string  `json:"contactName"`
	ContactPhone *string  `json:"contactPhone"`
	ContactEmail *string  `json:"contactEmail"`
	Notes        *string  `json:"notes"`
	Status       *string  `json:"status"`
	Lat          *float64 `json:"lat"`
	Lng          *float64 `json:"lng"`
}

func (r *UpdateSiteRequest) Validate() error {
	trimInPlace(r.Name)
	trimInPlace(r.Address)
	trimInPlace(r.Area)
	trimInPlace(r.PropertyType)
	trimInPlace(r.ContactName)
	trimInPlace(r.ContactPhone)
	trimInPlace(r.Notes)
	trimInPlace(r.Status)
	if r.ContactEmail != nil {
		v := strings.ToLower(strings.TrimSpace(*r.ContactEmail))
		r.ContactEmail = &v
	}

	if r.Name != nil && *r.Name == "" {
		return response.NewAPIError(400, "name cannot be empty")
	}
	return validateCommon(
		deref(r.Name), deref(r.Address), deref(r.Notes),
		deref(r.PropertyType), deref(r.Status),
		deref(r.ContactPhone), deref(r.ContactEmail), r.Lat, r.Lng)
}

func (r *UpdateSiteRequest) IsEmpty() bool {
	return r.Name == nil && r.Address == nil && r.Area == nil &&
		r.PropertyType == nil && r.ContactName == nil && r.ContactPhone == nil &&
		r.ContactEmail == nil && r.Notes == nil && r.Status == nil &&
		r.Lat == nil && r.Lng == nil
}

// validateCommon checks the fields shared by create and update. Empty values
// mean "not supplied" and are skipped.
func validateCommon(name, address, notes, propertyType, status, phone, email string, lat, lng *float64) error {
	if len(name) > maxNameLen {
		return response.NewAPIError(400, "name is too long")
	}
	if len(address) > maxAddressLen {
		return response.NewAPIError(400, "address is too long")
	}
	if len(notes) > maxNotesLen {
		return response.NewAPIError(400, "notes is too long")
	}
	if propertyType != "" && !PropertyType(propertyType).Valid() {
		return response.NewAPIError(400, "invalid propertyType")
	}
	if status != "" && !Status(status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if phone != "" && !validate.Phone(phone) {
		return response.NewAPIError(400, "contactPhone must be a valid phone number")
	}
	if email != "" && !validate.Email(email) {
		return response.NewAPIError(400, "contactEmail must be a valid address")
	}
	if lat != nil && (*lat < -90 || *lat > 90) {
		return response.NewAPIError(400, "lat must be between -90 and 90")
	}
	if lng != nil && (*lng < -180 || *lng > 180) {
		return response.NewAPIError(400, "lng must be between -180 and 180")
	}
	return nil
}

func trimInPlace(v *string) {
	if v != nil {
		*v = strings.TrimSpace(*v)
	}
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
