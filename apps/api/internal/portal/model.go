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

type MeResponse struct {
	Customer Customer `json:"customer"`
}