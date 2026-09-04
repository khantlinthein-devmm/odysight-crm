package payments

import (
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusPaid     Status = "paid"
	StatusFailed   Status = "failed"
	StatusRefunded Status = "refunded"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusPaid, StatusFailed, StatusRefunded:
		return true
	}
	return false
}

type Payment struct {
	ID            int64
	InvoiceNumber string
	PayerName     string
	Amount        float64
	Currency      string
	Method        string
	Status        Status
	CreatedAt     time.Time
}
