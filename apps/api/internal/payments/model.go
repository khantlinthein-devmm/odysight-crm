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

type Method string

const (
	MethodCash          Method = "cash"
	MethodBankTransfer  Method = "bank_transfer"
	MethodPromptPay     Method = "promptpay"
	MethodCreditCard    Method = "credit_card"
	MethodLinePay       Method = "line_pay"
	MethodOnlineWallet  Method = "online_wallet"
)

func (m Method) Valid() bool {
	switch m {
	case MethodCash, MethodBankTransfer, MethodPromptPay, MethodCreditCard, MethodLinePay, MethodOnlineWallet:
		return true
	}
	return false
}

type Payment struct {
	ID            int64
	InvoiceNumber string
	CustomerName  string
	BookingNumber string
	Amount        float64
	Currency      string
	Method        Method
	Status        Status
	CreatedAt     time.Time
}
