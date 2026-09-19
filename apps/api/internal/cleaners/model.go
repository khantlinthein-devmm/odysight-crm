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
	// LineID is the cleaner's LINE messenger handle, the usual contact
	// channel in Thailand. Blank when not collected.
	LineID string
	Skills string
	Status Status
	// Area is the dispatch zone (e.g. "Bang Na"). Used to match cleaners
	// with bookings in the same area for the mobile app.
	Area string
	// UserID links the cleaner profile to a login user (role CLEANER).
	// Nil means the profile is not linked yet (matched by email instead).
	UserID *int64
	// Lat/Lng hold the last reported GPS position from the mobile app.
	Lat *float64
	Lng *float64
	// IsOnline is true while the cleaner app has location sharing on.
	IsOnline bool
	// LastSeenAt is the last location-ping time. Nil when never reported.
	LastSeenAt *time.Time
	CreatedAt  time.Time
}

// PhoneNumber is an extra labeled phone number for a cleaner
// (cleaner_phones table).
type PhoneNumber struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	Phone string `json:"phone"`
}
