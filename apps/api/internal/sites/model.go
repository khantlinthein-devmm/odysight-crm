package sites

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusInactive:
		return true
	}
	return false
}

type Site struct {
	ID          int64
	CustomerID  int64
	Name        string
	Address     string
	ContactName string
	Phone       string
	Email       string
	Notes       string
	Status      Status
	Latitude    *float64
	Longitude   *float64
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
