package servicerecords

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type ServiceRecordDTO struct {
	ID            int64      `json:"id"`
	BookingNumber string     `json:"bookingNumber"`
	CleanerName   string     `json:"cleanerName"`
	ServiceType   string     `json:"serviceType"`
	Rating        *int       `json:"rating"`
	Status        string     `json:"status"`
	Notes         string     `json:"notes"`
	CompletedAt   *time.Time `json:"completedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func toDTO(r ServiceRecord) ServiceRecordDTO {
	return ServiceRecordDTO{
		ID:            r.ID,
		BookingNumber: r.BookingNumber,
		CleanerName:   r.CleanerName,
		ServiceType:   r.ServiceType,
		Rating:        r.Rating,
		Status:        string(r.Status),
		Notes:         r.Notes,
		CompletedAt:   r.CompletedAt,
		CreatedAt:     r.CreatedAt,
	}
}

type CreateServiceRecordRequest struct {
	BookingNumber string `json:"bookingNumber"`
	CleanerName   string `json:"cleanerName"`
	ServiceType   string `json:"serviceType"`
	Rating        *int   `json:"rating"`
	Status        string `json:"status"`
	Notes         string `json:"notes"`
}

func (r *CreateServiceRecordRequest) Validate() error {
	r.BookingNumber = strings.TrimSpace(r.BookingNumber)
	r.CleanerName = strings.TrimSpace(r.CleanerName)
	r.ServiceType = strings.TrimSpace(r.ServiceType)
	r.Notes = strings.TrimSpace(r.Notes)

	if r.BookingNumber == "" {
		return response.NewAPIError(400, "bookingNumber is required")
	}
	if r.CleanerName == "" {
		return response.NewAPIError(400, "cleanerName is required")
	}
	if r.ServiceType == "" {
		return response.NewAPIError(400, "serviceType is required")
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 5) {
		return response.NewAPIError(400, "rating must be between 1 and 5")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdateServiceRecordRequest struct {
	BookingNumber *string `json:"bookingNumber"`
	CleanerName   *string `json:"cleanerName"`
	ServiceType   *string `json:"serviceType"`
	Rating        *int    `json:"rating"`
	Status        *string `json:"status"`
	Notes         *string `json:"notes"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateServiceRecordRequest) Validate() error {
	if r.BookingNumber != nil && strings.TrimSpace(*r.BookingNumber) == "" {
		return response.NewAPIError(400, "bookingNumber cannot be empty")
	}
	if r.CleanerName != nil && strings.TrimSpace(*r.CleanerName) == "" {
		return response.NewAPIError(400, "cleanerName cannot be empty")
	}
	if r.ServiceType != nil && strings.TrimSpace(*r.ServiceType) == "" {
		return response.NewAPIError(400, "serviceType cannot be empty")
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 5) {
		return response.NewAPIError(400, "rating must be between 1 and 5")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateServiceRecordRequest) IsEmpty() bool {
	return r.BookingNumber == nil && r.CleanerName == nil && r.ServiceType == nil &&
		r.Rating == nil && r.Status == nil && r.Notes == nil
}
