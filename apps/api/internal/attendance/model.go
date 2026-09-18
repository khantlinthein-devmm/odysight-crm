package attendance

import "time"

// Record is one cleaner's attendance row for a single work date.
type Record struct {
	ID          int64
	CleanerID   int64
	CleanerName string
	WorkDate    string // YYYY-MM-DD
	CheckInAt   *time.Time
	CheckOutAt  *time.Time
	Note        string
	CreatedAt   time.Time
}
