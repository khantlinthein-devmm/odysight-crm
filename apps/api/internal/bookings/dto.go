package bookings

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type BookingDTO struct {
	ID              int64     `json:"id"`
	BookingNumber   string    `json:"bookingNumber"`
	CustomerName    string    `json:"customerName"`
	ServiceType     string    `json:"serviceType"`
	ScheduledFor    time.Time `json:"scheduledFor"`
	DurationMinutes int       `json:"durationMinutes"`
	Address         string    `json:"address"`
	AssignedCleaner string    `json:"assignedCleaner"`
	Status          string    `json:"status"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"createdAt"`
}

func toDTO(b Booking) BookingDTO {
	return BookingDTO{
		ID:              b.ID,
		BookingNumber:   b.BookingNumber,
		CustomerName:    b.CustomerName,
		ServiceType:     string(b.ServiceType),
		ScheduledFor:    b.ScheduledFor,
		DurationMinutes: b.DurationMinutes,
		Address:         b.Address,
		AssignedCleaner: b.AssignedCleaner,
		Status:          string(b.Status),
		Notes:           b.Notes,
		CreatedAt:       b.CreatedAt,
	}
}

type CreateBookingRequest struct {
	CustomerName    string `json:"customerName"`
	ServiceType     string `json:"serviceType"`
	ScheduledFor    string `json:"scheduledFor"`
	DurationMinutes int    `json:"durationMinutes"`
	Address         string `json:"address"`
	AssignedCleaner string `json:"assignedCleaner"`
	Status          string `json:"status"`
	Notes           string `json:"notes"`
}

func (r *CreateBookingRequest) Validate() error {
	r.CustomerName = strings.TrimSpace(r.CustomerName)
	r.ServiceType = strings.TrimSpace(r.ServiceType)
	r.Address = strings.TrimSpace(r.Address)
	r.AssignedCleaner = strings.TrimSpace(r.AssignedCleaner)
	r.Notes = strings.TrimSpace(r.Notes)
	r.ScheduledFor = strings.TrimSpace(r.ScheduledFor)

	if r.CustomerName == "" {
		return response.NewAPIError(400, "customerName is required")
	}
	if !ServiceType(r.ServiceType).Valid() {
		return response.NewAPIError(400, "invalid serviceType")
	}
	if r.ScheduledFor == "" {
		return response.NewAPIError(400, "scheduledFor is required")
	}
	if _, err := time.Parse(time.RFC3339, r.ScheduledFor); err != nil {
		return response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
	}
	if r.DurationMinutes <= 0 {
		return response.NewAPIError(400, "durationMinutes must be greater than zero")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdateBookingRequest struct {
	CustomerName    *string `json:"customerName"`
	ServiceType     *string `json:"serviceType"`
	ScheduledFor    *string `json:"scheduledFor"`
	DurationMinutes *int    `json:"durationMinutes"`
	Address         *string `json:"address"`
	AssignedCleaner *string `json:"assignedCleaner"`
	Status          *string `json:"status"`
	Notes           *string `json:"notes"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateBookingRequest) Validate() error {
	if r.CustomerName != nil && strings.TrimSpace(*r.CustomerName) == "" {
		return response.NewAPIError(400, "customerName cannot be empty")
	}
	if r.ServiceType != nil && !ServiceType(*r.ServiceType).Valid() {
		return response.NewAPIError(400, "invalid serviceType")
	}
	if r.ScheduledFor != nil {
		if _, err := time.Parse(time.RFC3339, *r.ScheduledFor); err != nil {
			return response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
		}
	}
	if r.DurationMinutes != nil && *r.DurationMinutes <= 0 {
		return response.NewAPIError(400, "durationMinutes must be greater than zero")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateBookingRequest) IsEmpty() bool {
	return r.CustomerName == nil && r.ServiceType == nil && r.ScheduledFor == nil &&
		r.DurationMinutes == nil && r.Address == nil && r.AssignedCleaner == nil &&
		r.Status == nil && r.Notes == nil
}
