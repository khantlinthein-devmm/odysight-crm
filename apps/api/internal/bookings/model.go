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

type Booking struct {
	ID              int64
	BookingNumber   string
	CustomerName    string
	ServiceType     ServiceType
	ScheduledFor    time.Time
	DurationMinutes int
	Address         string
	AssignedCleaner string
	Status          Status
	Notes           string
	CreatedAt       time.Time
}
