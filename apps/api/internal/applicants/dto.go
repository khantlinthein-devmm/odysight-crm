package applicants

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type ApplicantDTO struct {
	ID          int64     `json:"id"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Nationality string    `json:"nationality"`
	VisaType    string    `json:"visaType"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

func toDTO(a Applicant) ApplicantDTO {
	return ApplicantDTO{
		ID:          a.ID,
		FirstName:   a.FirstName,
		LastName:    a.LastName,
		Email:       a.Email,
		Phone:       a.Phone,
		Nationality: a.Nationality,
		VisaType:    a.VisaType,
		Status:      string(a.Status),
		CreatedAt:   a.CreatedAt,
	}
}

type CreateApplicantRequest struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Nationality string `json:"nationality"`
	VisaType    string `json:"visaType"`
	Status      string `json:"status"`
}

func (r *CreateApplicantRequest) Validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Phone = strings.TrimSpace(r.Phone)
	r.Nationality = strings.TrimSpace(r.Nationality)
	r.VisaType = strings.TrimSpace(r.VisaType)

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
	if r.Nationality == "" {
		return response.NewAPIError(400, "nationality is required")
	}
	if r.VisaType == "" {
		return response.NewAPIError(400, "visaType is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdateApplicantRequest struct {
	FirstName   *string `json:"firstName"`
	LastName    *string `json:"lastName"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	Nationality *string `json:"nationality"`
	VisaType    *string `json:"visaType"`
	Status      *string `json:"status"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateApplicantRequest) Validate() error {
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
	if r.Nationality != nil && strings.TrimSpace(*r.Nationality) == "" {
		return response.NewAPIError(400, "nationality cannot be empty")
	}
	if r.VisaType != nil && strings.TrimSpace(*r.VisaType) == "" {
		return response.NewAPIError(400, "visaType cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateApplicantRequest) IsEmpty() bool {
	return r.FirstName == nil && r.LastName == nil && r.Email == nil &&
		r.Phone == nil && r.Nationality == nil && r.VisaType == nil && r.Status == nil
}
