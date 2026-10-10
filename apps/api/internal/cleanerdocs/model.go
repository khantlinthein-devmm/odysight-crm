package cleanerdocs

import "time"

// DocType identifies the kind of identity / work-authorization document.
type DocType string

const (
	TypePassport   DocType = "passport"
	TypeVisa       DocType = "visa"
	TypeWorkPermit DocType = "work_permit"
	TypePinkCard   DocType = "pink_card"
	TypeIDCard     DocType = "id_card"
	TypeResume     DocType = "resume"
	TypeOther      DocType = "other"
)

var typeLabels = map[DocType]string{
	TypePassport:   "Passport",
	TypeVisa:       "Visa",
	TypeWorkPermit: "Work permit",
	TypePinkCard:   "Pink card",
	TypeIDCard:     "ID card",
	TypeResume:     "Resume",
	TypeOther:      "Document",
}

func (t DocType) Valid() bool {
	_, ok := typeLabels[t]
	return ok
}

func (t DocType) Label() string {
	if l, ok := typeLabels[t]; ok {
		return l
	}
	return string(t)
}

// Expiry status values exposed to the UI.
const (
	StatusNone     = "none"
	StatusValid    = "valid"
	StatusExpiring = "expiring"
	StatusExpired  = "expired"
)

// ExpiringWindowDays is how far ahead a document counts as "expiring" and
// when the first office reminder goes out.
const ExpiringWindowDays = 60

type Document struct {
	ID            int64
	CleanerID     int64
	Type          DocType
	NumberEnc     []byte
	IssueDate     *time.Time
	ExpiryDate    *time.Time
	Notes         string
	FileName      string
	OriginalName  string
	ContentType   string
	SizeBytes     int64
	UploadedBy    *int64
	ReminderStage int16
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ExpiringItem is a document joined with its cleaner's name for the
// office-wide "expiring soon" list and reminder emails.
type ExpiringItem struct {
	DocumentID    int64
	CleanerID     int64
	CleanerName   string
	Type          DocType
	ExpiryDate    time.Time
	ReminderStage int16
}

// today returns the current local calendar date as UTC midnight, matching how
// pgx decodes DATE columns, so day differences are exact.
func today(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DaysUntil returns whole days from now's calendar date to expiry (negative
// once expired).
func DaysUntil(expiry, now time.Time) int {
	e := time.Date(expiry.Year(), expiry.Month(), expiry.Day(), 0, 0, 0, 0, time.UTC)
	return int(e.Sub(today(now)).Hours() / 24)
}

func ExpiryStatus(expiry *time.Time, now time.Time) string {
	if expiry == nil {
		return StatusNone
	}
	days := DaysUntil(*expiry, now)
	switch {
	case days < 0:
		return StatusExpired
	case days <= ExpiringWindowDays:
		return StatusExpiring
	}
	return StatusValid
}

// reminderStage maps days-to-expiry onto the reminder ladder:
// 0 none, 1 within 60d, 2 within 30d, 3 within 7d, 4 expired.
func reminderStage(days int) int16 {
	switch {
	case days < 0:
		return 4
	case days <= 7:
		return 3
	case days <= 30:
		return 2
	case days <= ExpiringWindowDays:
		return 1
	}
	return 0
}
