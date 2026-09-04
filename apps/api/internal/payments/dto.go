package payments

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type PaymentDTO struct {
	ID            int64     `json:"id"`
	InvoiceNumber string    `json:"invoiceNumber"`
	PayerName     string    `json:"payerName"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Method        string    `json:"method"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func toDTO(p Payment) PaymentDTO {
	return PaymentDTO{
		ID:            p.ID,
		InvoiceNumber: p.InvoiceNumber,
		PayerName:     p.PayerName,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Method:        p.Method,
		Status:        string(p.Status),
		CreatedAt:     p.CreatedAt,
	}
}

type CreatePaymentRequest struct {
	PayerName string  `json:"payerName"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Method    string  `json:"method"`
	Status    string  `json:"status"`
}

func (r *CreatePaymentRequest) Validate() error {
	r.PayerName = strings.TrimSpace(r.PayerName)
	r.Currency = strings.ToUpper(strings.TrimSpace(r.Currency))
	r.Method = strings.TrimSpace(r.Method)

	if r.PayerName == "" {
		return response.NewAPIError(400, "payerName is required")
	}
	if r.Amount <= 0 {
		return response.NewAPIError(400, "amount must be greater than zero")
	}
	if len(r.Currency) != 3 {
		return response.NewAPIError(400, "a valid 3-letter currency code is required")
	}
	if r.Method == "" {
		return response.NewAPIError(400, "method is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

type UpdatePaymentRequest struct {
	Status *string `json:"status"`
	Method *string `json:"method"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdatePaymentRequest) Validate() error {
	if r.Method != nil && strings.TrimSpace(*r.Method) == "" {
		return response.NewAPIError(400, "method cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdatePaymentRequest) IsEmpty() bool {
	return r.Status == nil && r.Method == nil
}
