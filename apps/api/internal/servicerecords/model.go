package servicerecords

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusCompleted  Status = "completed"
	StatusRescheduled Status = "rescheduled"
	StatusCancelled  Status = "cancelled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusCompleted, StatusRescheduled, StatusCancelled:
		return true
	}
	return false
}

type ServiceRecord struct {
	ID            int64
	BookingNumber string
	CleanerName   string
	ServiceType   string
	Rating        *int
	Status        Status
	Notes         string
	CompletedAt   *time.Time
	CreatedAt     time.Time
}
