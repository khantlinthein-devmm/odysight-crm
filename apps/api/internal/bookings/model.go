package bookings

import "time"

type Status string

const (
	StatusPending     Status = "pending"
	StatusConfirmed   Status = "confirmed"
	StatusInProgress  Status = "in_progress"
	StatusCompleted   Status = "completed"
	StatusCancelled   Status = "cancelled"
	StatusNoShow      Status = "no_show"
	StatusRescheduled Status = "rescheduled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusConfirmed, StatusInProgress, StatusCompleted,
		StatusCancelled, StatusNoShow, StatusRescheduled:
		return true
	}
	return false
}

type ServiceType string

const (
	SvcHouseCleaning   ServiceType = "house_cleaning"
	SvcCondoCleaning   ServiceType = "condo_cleaning"
	SvcDeepCleaning    ServiceType = "deep_cleaning"
	SvcMoveInOut       ServiceType = "move_in_out"
	SvcAfterRenovation ServiceType = "after_renovation"
	SvcOfficeCleaning  ServiceType = "office_cleaning"
	SvcJunkRemoval     ServiceType = "junk_removal"
	SvcAirconService   ServiceType = "aircon_service"
)

func (s ServiceType) Valid() bool {
	switch s {
	case SvcHouseCleaning, SvcCondoCleaning, SvcDeepCleaning, SvcMoveInOut,
		SvcAfterRenovation, SvcOfficeCleaning, SvcJunkRemoval, SvcAirconService:
		return true
	}
	return false
}

// CleanerBrief links a booking to a real cleaner from the cleaners table.
type CleanerBrief struct {
	ID   int64
	Name string
	Role string // "primary" | "crew"
}

type Booking struct {
	ID              int64
	BookingNumber   string
	CustomerName    string
	CustomerEmail   string
	CustomerID      *int64
	ServiceType     ServiceType
	ScheduledFor    time.Time
	DurationMinutes int
	Address         string
	// Area is the dispatch zone (e.g. "Bang Na"), copied from the customer
	// when the booking is created. Cleaners filter the available pool by it.
	Area            string
	AssignedCleaner string // display: primary cleaner's full name
	Status          Status
	Notes           string
	CreatedAt       time.Time
	IsRecurring     bool
	Recurrence      string // "weekly" | "biweekly" | "monthly" | "" (never recurring)
	SeriesID        *string
	Cleaners        []CleanerBrief
}

// Recurrence frequencies for recurring bookings.
const (
	RecurWeekly   = "weekly"
	RecurBiweekly = "biweekly"
	RecurMonthly  = "monthly"
)

func IsValidRecurrence(r string) bool {
	switch r {
	case RecurWeekly, RecurBiweekly, RecurMonthly:
		return true
	}
	return false
}
