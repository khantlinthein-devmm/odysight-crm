package cleaners

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

type CleanerDTO struct {
	ID         int64      `json:"id"`
	FirstName  string     `json:"firstName"`
	LastName   string     `json:"lastName"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	LineID     string     `json:"lineId"`
	Skills     string     `json:"skills"`
	Status     string     `json:"status"`
	Area       string     `json:"area"`
	UserID     *int64     `json:"userId"`
	Lat        *float64   `json:"lat"`
	Lng        *float64   `json:"lng"`
	IsOnline   bool       `json:"isOnline"`
	LastSeenAt *time.Time `json:"lastSeenAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func toDTO(c Cleaner) CleanerDTO {
	return CleanerDTO{
		ID:         c.ID,
		FirstName:  c.FirstName,
		LastName:   c.LastName,
		Phone:      c.Phone,
		Email:      c.Email,
		LineID:     c.LineID,
		Skills:     c.Skills,
		Status:     string(c.Status),
		Area:       c.Area,
		UserID:     c.UserID,
		Lat:        c.Lat,
		Lng:        c.Lng,
		IsOnline:   c.IsOnline,
		LastSeenAt: c.LastSeenAt,
		CreatedAt:  c.CreatedAt,
	}
}

type CreateCleanerRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	LineID    string `json:"lineId"`
	Skills    string `json:"skills"`
	Status    string `json:"status"`
	Area      string `json:"area"`
	UserID    *int64 `json:"userId"`
}

func (r *CreateCleanerRequest) Validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.LineID = strings.TrimSpace(r.LineID)
	r.Skills = strings.TrimSpace(r.Skills)
	r.Area = strings.TrimSpace(r.Area)

	if r.FirstName == "" {
		return response.NewAPIError(400, "firstName is required")
	}
	if !validate.Phone(r.Phone) {
		return response.NewAPIError(400, "a valid phone is required")
	}
	// Last name and email are optional: many field staff have neither on file.
	if r.Email != "" && !validate.Email(r.Email) {
		return response.NewAPIError(400, "email must be a valid address")
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
	if r.UserID != nil && *r.UserID < 1 {
		return response.NewAPIError(400, "userId must be a positive id")
	}
	return nil
}

type UpdateCleanerRequest struct {
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	LineID    *string `json:"lineId"`
	Skills    *string `json:"skills"`
	Status    *string `json:"status"`
	Area      *string `json:"area"`
	UserID    *int64  `json:"userId"`
	IsOnline  *bool   `json:"isOnline"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateCleanerRequest) Validate() error {
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
	if r.LineID != nil {
		lineID := strings.TrimSpace(*r.LineID)
		r.LineID = &lineID
	}
	if r.Phone != nil && !validate.Phone(*r.Phone) {
		return response.NewAPIError(400, "a valid phone is required")
	}
	if r.Skills != nil && strings.TrimSpace(*r.Skills) == "" {
		return response.NewAPIError(400, "skills cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.UserID != nil && *r.UserID < 1 {
		return response.NewAPIError(400, "userId must be a positive id")
	}
	return nil
}

func (r *UpdateCleanerRequest) IsEmpty() bool {
	return r.FirstName == nil && r.LastName == nil && r.Phone == nil &&
		r.Email == nil && r.LineID == nil && r.Skills == nil && r.Status == nil &&
		r.Area == nil && r.UserID == nil && r.IsOnline == nil
}

// LocationUpdateRequest is the GPS ping from the cleaner mobile app.
// PATCH /api/v1/cleaners/me/location
type LocationUpdateRequest struct {
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	Area     *string  `json:"area"`
	IsOnline *bool    `json:"isOnline"`
}

func (r *LocationUpdateRequest) Validate() error {
	if r.Lat == nil && r.Lng == nil && r.Area == nil && r.IsOnline == nil {
		return response.NewAPIError(400, errNoFields)
	}
	if r.Lat != nil && (*r.Lat < -90 || *r.Lat > 90) {
		return response.NewAPIError(400, "lat must be between -90 and 90")
	}
	if r.Lng != nil && (*r.Lng < -180 || *r.Lng > 180) {
		return response.NewAPIError(400, "lng must be between -180 and 180")
	}
	if r.Area != nil {
		v := strings.TrimSpace(*r.Area)
		r.Area = &v
	}
	return nil
}

// AddPhoneRequest adds a labeled phone number to a cleaner.
type AddPhoneRequest struct {
	Label string `json:"label"`
	Phone string `json:"phone"`
}

func (r *AddPhoneRequest) Validate() error {
	r.Label = strings.TrimSpace(r.Label)
	r.Phone = strings.TrimSpace(r.Phone)

	if len(r.Label) < 1 || len(r.Label) > 30 {
		return response.NewAPIError(400, "label must be between 1 and 30 characters")
	}
	if !validate.Phone(r.Phone) {
		return response.NewAPIError(400, "a valid phone is required")
	}
	return nil
}
