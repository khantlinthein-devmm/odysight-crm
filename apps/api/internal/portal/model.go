package portal

import "time"

// portalAudience is the JWT audience for customer-portal sessions.
const portalAudience = "odysight-portal"

// Customer is the subset of a customer record exposed to the portal.
type Customer struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Area      string `json:"area"`
}

// Booking is a customer's own booking as shown in the portal.
type Booking struct {
	ID              int64     `json:"id"`
	BookingNumber   string    `json:"bookingNumber"`
	ServiceType     string    `json:"serviceType"`
	ScheduledFor    time.Time `json:"scheduledFor"`
	DurationMinutes int       `json:"durationMinutes"`
	Address         string    `json:"address"`
	Assignee        string    `json:"assignee"`
	Status          string    `json:"status"`
	Notes           string    `json:"notes"`
}

// Site is a customer's service location as exposed to the portal.
type Site struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	IsDefault bool   `json:"isDefault"`
}

// ServiceItem is one bookable service from the workspace catalog.
type ServiceItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	DurationMinutes int     `json:"durationMinutes"`
	BasePrice       float64 `json:"basePrice"`
	Active          bool    `json:"active"`
}

// CreateBookingRequest is the portal self-booking payload. Everything is
// scoped to the logged-in customer: no customerId, no contract, no cleaner
// selection. SiteID is optional but must belong to the customer when given.
type CreateBookingRequest struct {
	ServiceType     string `json:"serviceType"`
	ScheduledFor    string `json:"scheduledFor"`
	DurationMinutes int    `json:"durationMinutes"`
	SiteID          *int64 `json:"siteId"`
	Address         string `json:"address"`
	Notes           string `json:"notes"`
}

type MeResponse struct {
	Customer Customer `json:"customer"`
}