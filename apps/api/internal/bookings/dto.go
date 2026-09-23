package bookings

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

type CleanerBriefDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type BookingDTO struct {
	ID              int64             `json:"id"`
	BookingNumber   string            `json:"bookingNumber"`
	CustomerName    string            `json:"customerName"`
	CustomerEmail   string            `json:"customerEmail"`
	CustomerID      *int64            `json:"customerId"`
	SiteID          *int64            `json:"siteId"`
	ContractID      *int64            `json:"contractId"`
	ServiceType     string            `json:"serviceType"`
	ScheduledFor    time.Time         `json:"scheduledFor"`
	DurationMinutes int               `json:"durationMinutes"`
	Address         string            `json:"address"`
	Area            string            `json:"area"`
	AssignedCleaner string            `json:"assignedCleaner"`
	Status          string            `json:"status"`
	Notes           string            `json:"notes"`
	IsRecurring     bool              `json:"isRecurring"`
	Recurrence      string            `json:"recurrence"`
	SeriesID        *string           `json:"seriesId"`
	CreatedAt       time.Time         `json:"createdAt"`
	Cleaners        []CleanerBriefDTO `json:"cleaners"`
}

func toDTO(b Booking) BookingDTO {
	cleaners := make([]CleanerBriefDTO, 0, len(b.Cleaners))
	for _, c := range b.Cleaners {
		cleaners = append(cleaners, CleanerBriefDTO{ID: c.ID, Name: c.Name, Role: c.Role})
	}
	return BookingDTO{
		ID:              b.ID,
		BookingNumber:   b.BookingNumber,
		CustomerName:    b.CustomerName,
		CustomerEmail:   b.CustomerEmail,
		CustomerID:      b.CustomerID,
		SiteID:          b.SiteID,
		ContractID:      b.ContractID,
		ServiceType:     string(b.ServiceType),
		ScheduledFor:    b.ScheduledFor,
		DurationMinutes: b.DurationMinutes,
		Address:         b.Address,
		Area:            b.Area,
		AssignedCleaner: b.AssignedCleaner,
		Status:          string(b.Status),
		Notes:           b.Notes,
		IsRecurring:     b.IsRecurring,
		Recurrence:      b.Recurrence,
		SeriesID:        b.SeriesID,
		CreatedAt:       b.CreatedAt,
		Cleaners:        cleaners,
	}
}

type CreateBookingRequest struct {
	CustomerName    string `json:"customerName"`
	CustomerEmail   string `json:"customerEmail"`
	CustomerID      *int64 `json:"customerId"`
	SiteID          *int64 `json:"siteId"`
	ContractID      *int64 `json:"contractId"`
	ServiceType     string `json:"serviceType"`
	ScheduledFor    string `json:"scheduledFor"`
	DurationMinutes int    `json:"durationMinutes"`
	Address         string `json:"address"`
	Area            string `json:"area"`
	AssignedCleaner string `json:"assignedCleaner"`
	Status          string `json:"status"`
	Notes           string `json:"notes"`
	// IsRecurring marks the booking as the next occurrence of a repeating
	// schedule. Recurrence must be set whenever IsRecurring is true.
	IsRecurring bool   `json:"isRecurring"`
	Recurrence  string `json:"recurrence"`
	// CleanerIDs selects real cleaners; the first element becomes the primary
	// cleaner and the rest are crew. Overrides assignedCleaner when present.
	CleanerIDs []int64 `json:"cleanerIds"`
}

func (r *CreateBookingRequest) Validate() error {
	r.CustomerName = strings.TrimSpace(r.CustomerName)
	r.CustomerEmail = strings.ToLower(strings.TrimSpace(r.CustomerEmail))
	r.ServiceType = strings.TrimSpace(r.ServiceType)
	r.Address = strings.TrimSpace(r.Address)
	r.Area = strings.TrimSpace(r.Area)
	r.AssignedCleaner = strings.TrimSpace(r.AssignedCleaner)
	r.Notes = strings.TrimSpace(r.Notes)
	r.ScheduledFor = strings.TrimSpace(r.ScheduledFor)
	r.Recurrence = strings.TrimSpace(r.Recurrence)

	if r.CustomerName == "" {
		return response.NewAPIError(400, "customerName is required")
	}
	if r.CustomerEmail != "" && !validate.Email(r.CustomerEmail) {
		return response.NewAPIError(400, "a valid customerEmail is required")
	}
	// Service types are admin-editable via Settings → Service catalog,
	// so any non-empty value is accepted (CHECK constraint removed).
	if r.ServiceType == "" {
		return response.NewAPIError(400, "serviceType is required")
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
	if r.IsRecurring && !IsValidRecurrence(r.Recurrence) {
		return response.NewAPIError(400, "recurrence must be weekly, biweekly or monthly when isRecurring is true")
	}
	if !r.IsRecurring && r.Recurrence != "" {
		return response.NewAPIError(400, "recurrence requires isRecurring to be true")
	}
	return nil
}

type UpdateBookingRequest struct {
	CustomerName    *string `json:"customerName"`
	CustomerEmail   *string `json:"customerEmail"`
	CustomerID      *int64  `json:"customerId"`
	SiteID          *int64  `json:"siteId"`
	ContractID      *int64  `json:"contractId"`
	ClearSiteID     *bool   `json:"clearSiteId"`
	ClearContractID *bool   `json:"clearContractId"`
	ServiceType     *string `json:"serviceType"`
	ScheduledFor    *string `json:"scheduledFor"`
	DurationMinutes *int    `json:"durationMinutes"`
	Address         *string `json:"address"`
	Area            *string `json:"area"`
	AssignedCleaner *string `json:"assignedCleaner"`
	Status          *string `json:"status"`
	Notes           *string `json:"notes"`
	// IsRecurring *false stops a series; Recurrence changes the frequency of an
	// active series. Both are independent so either can be updated alone.
	IsRecurring *bool   `json:"isRecurring"`
	Recurrence  *string `json:"recurrence"`
	// CleanerIDs replaces the whole assignment when present (nil = unchanged,
	// empty array = clear). First element becomes the primary cleaner.
	CleanerIDs []int64 `json:"cleanerIds"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateBookingRequest) Validate() error {
	if r.CustomerName != nil && strings.TrimSpace(*r.CustomerName) == "" {
		return response.NewAPIError(400, "customerName cannot be empty")
	}
	if r.CustomerEmail != nil {
		email := strings.ToLower(strings.TrimSpace(*r.CustomerEmail))
		if email != "" && !validate.Email(email) {
			return response.NewAPIError(400, "a valid customerEmail is required")
		}
		r.CustomerEmail = &email
	}
	if r.ServiceType != nil && strings.TrimSpace(*r.ServiceType) == "" {
		return response.NewAPIError(400, "serviceType cannot be empty")
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
	if r.Recurrence != nil {
		rec := strings.TrimSpace(*r.Recurrence)
		if rec != "" && !IsValidRecurrence(rec) {
			return response.NewAPIError(400, "recurrence must be weekly, biweekly or monthly")
		}
		r.Recurrence = &rec
	}
	return nil
}

func (r *UpdateBookingRequest) IsEmpty() bool {
	return r.CustomerName == nil && r.CustomerEmail == nil && r.CustomerID == nil &&
		r.SiteID == nil && r.ContractID == nil && r.ClearSiteID == nil && r.ClearContractID == nil &&
		r.ServiceType == nil && r.ScheduledFor == nil &&
		r.DurationMinutes == nil && r.Address == nil && r.Area == nil && r.AssignedCleaner == nil &&
		r.Status == nil && r.Notes == nil && r.CleanerIDs == nil &&
		r.IsRecurring == nil && r.Recurrence == nil
}
