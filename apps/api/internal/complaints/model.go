// Package complaints tracks customer complaints against a service-level
// target and books free re-clean visits to resolve them.
package complaints

import (
	"strings"
	"time"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusResolved, StatusClosed:
		return true
	}
	return false
}

// Done reports whether the complaint no longer counts against the SLA.
func (s Status) Done() bool { return s == StatusResolved || s == StatusClosed }

var categories = map[string]bool{"quality": true, "missed_area": true, "late": true, "no_show": true,
	"damage": true, "staff_behaviour": true, "other": true}
var channels = map[string]bool{"phone": true, "line": true, "email": true, "portal": true, "in_person": true}

// slaHours is how long the office has to resolve a complaint.
var slaHours = map[string]int{"high": 24, "medium": 48, "low": 72}

// DueAt is the SLA deadline for a complaint raised at t.
func DueAt(severity string, t time.Time) time.Time {
	h, ok := slaHours[strings.TrimSpace(severity)]
	if !ok {
		h = slaHours["medium"]
	}
	return t.Add(time.Duration(h) * time.Hour)
}

type Complaint struct {
	ID               int64
	Number           string
	CustomerID       int64
	CustomerName     string
	SiteID           *int64
	SiteName         string
	BookingID        *int64
	BookingNumber    string
	Category         string
	Severity         string
	Channel          string
	Description      string
	Status           Status
	DueAt            time.Time
	Resolution       string
	ResolvedAt       *time.Time
	RecleanBookingID *int64
	RecleanNumber    string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Overdue reports whether an unresolved complaint is past its SLA.
func (c Complaint) Overdue(now time.Time) bool {
	return !c.Status.Done() && now.After(c.DueAt)
}
