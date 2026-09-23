package quotes

import "time"

type Status string

const (
	StatusDraft    Status = "draft"
	StatusSent     Status = "sent"
	StatusAccepted Status = "accepted"
	StatusRejected Status = "rejected"
	StatusExpired  Status = "expired"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusSent, StatusAccepted, StatusRejected, StatusExpired:
		return true
	}
	return false
}

type QuoteItem struct {
	ID          int64
	QuoteID     int64
	ServiceName string
	Description string
	Quantity    float64
	UnitPrice   float64
	LineTotal   float64
	SortOrder   int
}

type Quote struct {
	ID                 int64
	QuoteNumber        string
	CustomerID         int64
	SiteID             *int64
	Status             Status
	ValidUntil         *time.Time
	Subtotal           float64
	TaxRate            float64
	Total              float64
	Currency           string
	Notes              string
	Version            int
	AcceptedAt         *time.Time
	RejectedAt         *time.Time
	ConvertedBookingID *int64
	ConvertedContractID *int64
	Items              []QuoteItem
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
