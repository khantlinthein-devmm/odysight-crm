package attendance

import "time"

// PersonType distinguishes the two workforces attendance is kept for: field
// cleaners (cleaners table) and office/team staff (users table).
type PersonType string

const (
	PersonCleaner PersonType = "cleaner"
	PersonStaff   PersonType = "staff"
)

func (p PersonType) Valid() bool {
	switch p {
	case PersonCleaner, PersonStaff:
		return true
	}
	return false
}

// Person identifies whose attendance a row belongs to.
type Person struct {
	Type PersonType
	ID   int64
}

// CleanerID returns the cleaners.id to store, or nil for a staff row.
func (p Person) CleanerID() any {
	if p.Type == PersonCleaner {
		return p.ID
	}
	return nil
}

// UserID returns the users.id to store, or nil for a cleaner row.
func (p Person) UserID() any {
	if p.Type == PersonStaff {
		return p.ID
	}
	return nil
}

// Record is one person's attendance row for a single work date.
type Record struct {
	ID         int64
	PersonType PersonType
	PersonID   int64
	PersonName string
	WorkDate   string // YYYY-MM-DD
	CheckInAt  *time.Time
	CheckOutAt *time.Time
	Note       string
	CreatedAt  time.Time
}
