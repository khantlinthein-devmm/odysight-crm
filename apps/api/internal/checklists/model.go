package checklists

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted:
		return true
	}
	return false
}

type TemplateItem struct {
	ID        int64
	Label     string
	SortOrder int
}

type Template struct {
	ID          int64
	Name        string
	ServiceType string
	IsActive    bool
	Items       []TemplateItem
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ChecklistItem struct {
	ID            int64
	ChecklistID   int64
	Label         string
	IsCompleted   bool
	CompletedBy   *string
	CompletedAt   *time.Time
	Notes         string
	BeforePhotoURL *string
	AfterPhotoURL  *string
	SortOrder     int
}

type Checklist struct {
	ID                int64
	BookingID         int64
	TemplateID        *int64
	Status            Status
	ClientSignature   *string
	ClientConfirmedAt *time.Time
	Items             []ChecklistItem
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
