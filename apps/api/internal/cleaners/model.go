package cleaners

import "time"

type Status string

const (
	StatusAvailable Status = "available"
	StatusAssigned  Status = "assigned"
	StatusOnLeave   Status = "on_leave"
	StatusInactive  Status = "inactive"
)

func (s Status) Valid() bool {
	switch s {
	case StatusAvailable, StatusAssigned, StatusOnLeave, StatusInactive:
		return true
	}
	return false
}

type Cleaner struct {
	ID        int64
	FirstName string
	LastName  string
	Phone     string
	Email     string
	Skills    string
	Status    Status
	CreatedAt time.Time
}
