package documents

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type DocumentDTO struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	ApplicantName string    `json:"applicantName"`
	FileSizeKb    int64     `json:"fileSizeKb"`
	Status        string    `json:"status"`
	UploadedAt    time.Time `json:"uploadedAt"`
}

func toDTO(d Document) DocumentDTO {
	return DocumentDTO{
		ID:            d.ID,
		Name:          d.Name,
		Type:          d.Type,
		ApplicantName: d.ApplicantName,
		FileSizeKb:    d.FileSizeKb,
		Status:        string(d.Status),
		UploadedAt:    d.UploadedAt,
	}
}

type UploadDocumentRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	ApplicantName string `json:"applicantName"`
	Status        string `json:"status"`
}

func (r *UploadDocumentRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Type = strings.TrimSpace(r.Type)
	r.ApplicantName = strings.TrimSpace(r.ApplicantName)

	if r.Name == "" {
		return response.NewAPIError(400, "name is required")
	}
	if r.Type == "" {
		return response.NewAPIError(400, "type is required")
	}
	if r.ApplicantName == "" {
		return response.NewAPIError(400, "applicantName is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdateDocumentRequest struct {
	Name   *string `json:"name"`
	Type   *string `json:"type"`
	Status *string `json:"status"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateDocumentRequest) Validate() error {
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return response.NewAPIError(400, "name cannot be empty")
	}
	if r.Type != nil && strings.TrimSpace(*r.Type) == "" {
		return response.NewAPIError(400, "type cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateDocumentRequest) IsEmpty() bool {
	return r.Name == nil && r.Type == nil && r.Status == nil
}
