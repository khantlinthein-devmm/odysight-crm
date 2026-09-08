package audit

import "time"

type Entry struct {
	ID         int64
	UserID     *int64
	UserName   string
	UserEmail  string
	Action     string
	Resource   string
	ResourceID *int64
	CreatedAt  time.Time
}
