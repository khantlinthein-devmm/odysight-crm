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

// ItemType says how a cleaner answers an item.
type ItemType string

const (
	ItemCheck    ItemType = "check"     // tick when done
	ItemScore    ItemType = "score"     // rate 1-5
	ItemPassFail ItemType = "pass_fail" // pass / fail / n/a
)

func (t ItemType) Valid() bool {
	switch t {
	case ItemCheck, ItemScore, ItemPassFail:
		return true
	}
	return false
}

// Result values for pass_fail items.
const (
	ResultPass = "pass"
	ResultFail = "fail"
	ResultNA   = "na"
)

// Review outcomes set by the office on a submitted checklist.
const (
	ReviewNone     = ""
	ReviewApproved = "approved"
	ReviewRework   = "rework"
)

type TemplateItem struct {
	ID           int64
	Label        string
	ItemType     ItemType
	RequirePhoto bool
	SortOrder    int
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
	ID              int64
	ChecklistID     int64
	Label           string
	ItemType        ItemType
	RequirePhoto    bool
	IsCompleted     bool
	Score           *int
	Result          *string
	CompletedBy     *string
	CompletedByUser *int64
	CompletedAt     *time.Time
	Notes           string
	BeforePhotoURL  *string
	AfterPhotoURL   *string
	SortOrder       int
}

type Checklist struct {
	ID                int64
	BookingID         int64
	TemplateID        *int64
	Status            Status
	ClientSignature   *string
	ClientConfirmedAt *time.Time
	SubmittedAt       *time.Time
	SubmittedBy       *string
	ReviewStatus      string
	ReviewNote        string
	ReviewedBy        *string
	ReviewedAt        *time.Time
	Items             []ChecklistItem
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Locked reports whether cleaners can no longer change the checklist: it was
// submitted and not sent back for rework.
func (c Checklist) Locked() bool {
	return c.SubmittedAt != nil
}

// Summary is one row of the office checklist list.
type Summary struct {
	ID            int64
	BookingID     int64
	BookingNumber string
	CustomerName  string
	ServiceType   string
	ScheduledFor  time.Time
	Cleaners      string
	Status        Status
	SubmittedAt   *time.Time
	ReviewStatus  string
	ItemsTotal    int
	ItemsDone     int
	AvgScore      *float64
	FailCount     int
	PhotoCount    int
}

// ListFilter narrows the office checklist list.
type ListFilter struct {
	State     string // "", pending, in_progress, submitted, approved, rework
	CleanerID int64
	From      *time.Time
	To        *time.Time
	Search    string
	Limit     int
	Offset    int
}
