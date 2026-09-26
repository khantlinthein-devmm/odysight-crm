package invoices

import "time"

type Status string

const (
	StatusDraft  Status = "draft"
	StatusIssued Status = "issued"
	StatusPaid   Status = "paid"
	StatusVoid   Status = "void"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusIssued, StatusPaid, StatusVoid:
		return true
	}
	return false
}

type Invoice struct {
	ID            int64
	InvoiceNumber string
	BookingID     int64
	BookingNumber string
	CustomerName  string
	CustomerEmail string
	Address       string
	ServiceType   string
	ServiceName   string
	Subtotal      float64
	TaxRate       float64
	TaxAmount     float64
	Total         float64
	Currency      string
	Status        Status
	// Tax-invoice snapshot taken at issue time.
	CustomerTaxID      string
	CustomerTaxBranch  string
	WithholdingRate    float64
	WithholdingAmount  float64
	ContractID         *int64
	IdempotencyKey     *string
	BillingPeriodStart *time.Time
	BillingPeriodEnd   *time.Time
	IssuedAt           time.Time
	PaidAt             *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// BookingSnapshot is the slice of a booking an invoice is drawn from.
type BookingSnapshot struct {
	ID                int64
	BookingNumber     string
	CustomerName      string
	CustomerEmail     string
	Address           string
	ServiceType       string
	DurationMinutes   int
	Status            string
	Completed         bool
	CustomerTaxID     string
	CustomerTaxBranch string
	WithholdingRate   float64
}

// NetPayable is what the customer actually transfers: the VAT-inclusive
// total less withholding tax, which they remit to the Revenue Department.
func (inv Invoice) NetPayable() float64 {
	return round2(inv.Total - inv.WithholdingAmount)
}
