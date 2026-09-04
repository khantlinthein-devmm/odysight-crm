package visacases

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type VisaCaseDTO struct {
	ID            int64     `json:"id"`
	CaseNumber    string    `json:"caseNumber"`
	ApplicantName string    `json:"applicantName"`
	VisaType      string    `json:"visaType"`
	Destination   string    `json:"destination"`
	AssignedTo    string    `json:"assignedTo"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func toDTO(c VisaCase) VisaCaseDTO {
	return VisaCaseDTO{
		ID:            c.ID,
		CaseNumber:    c.CaseNumber,
		ApplicantName: c.ApplicantName,
		VisaType:      c.VisaType,
		Destination:   c.Destination,
		AssignedTo:    c.AssignedTo,
		Status:        string(c.Status),
		CreatedAt:     c.CreatedAt,
	}
}

type CreateVisaCaseRequest struct {
	ApplicantName string `json:"applicantName"`
	VisaType      string `json:"visaType"`
	Destination   string `json:"destination"`
	AssignedTo    string `json:"assignedTo"`
	Status        string `json:"status"`
}

func (r *CreateVisaCaseRequest) Validate() error {
	r.ApplicantName = strings.TrimSpace(r.ApplicantName)
	r.VisaType = strings.TrimSpace(r.VisaType)
	r.Destination = strings.TrimSpace(r.Destination)
	r.AssignedTo = strings.TrimSpace(r.AssignedTo)

	if r.ApplicantName == "" {
		return response.NewAPIError(400, "applicantName is required")
	}
	if r.VisaType == "" {
		return response.NewAPIError(400, "visaType is required")
	}
	if r.Destination == "" {
		return response.NewAPIError(400, "destination is required")
	}
	if r.AssignedTo == "" {
		return response.NewAPIError(400, "assignedTo is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdateVisaCaseRequest struct {
	ApplicantName *string `json:"applicantName"`
	VisaType      *string `json:"visaType"`
	Destination   *string `json:"destination"`
	AssignedTo    *string `json:"assignedTo"`
	Status        *string `json:"status"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateVisaCaseRequest) Validate() error {
	if r.ApplicantName != nil && strings.TrimSpace(*r.ApplicantName) == "" {
		return response.NewAPIError(400, "applicantName cannot be empty")
	}
	if r.VisaType != nil && strings.TrimSpace(*r.VisaType) == "" {
		return response.NewAPIError(400, "visaType cannot be empty")
	}
	if r.Destination != nil && strings.TrimSpace(*r.Destination) == "" {
		return response.NewAPIError(400, "destination cannot be empty")
	}
	if r.AssignedTo != nil && strings.TrimSpace(*r.AssignedTo) == "" {
		return response.NewAPIError(400, "assignedTo cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateVisaCaseRequest) IsEmpty() bool {
	return r.ApplicantName == nil && r.VisaType == nil && r.Destination == nil &&
		r.AssignedTo == nil && r.Status == nil
}
