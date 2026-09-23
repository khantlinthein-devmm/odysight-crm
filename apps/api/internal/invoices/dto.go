package invoices

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type InvoiceDTO struct {
	ID            int64      `json:"id"`
	InvoiceNumber string     `json:"invoiceNumber"`
	BookingID     int64      `json:"bookingId"`
	BookingNumber string     `json:"bookingNumber"`
	CustomerName  string     `json:"customerName"`
	CustomerEmail string     `json:"customerEmail"`
	Address       string     `json:"address"`
	ServiceType   string     `json:"serviceType"`
	ServiceName   string     `json:"serviceName"`
	Subtotal      float64    `json:"subtotal"`
	TaxRate       float64    `json:"taxRate"`
	TaxAmount     float64    `json:"taxAmount"`
	Total         float64    `json:"total"`
	Currency      string     `json:"currency"`
	Status        Status     `json:"status"`
	ContractID    *int64     `json:"contractId"`
	IdempotencyKey *string   `json:"idempotencyKey"`
	BillingPeriodStart *string `json:"billingPeriodStart"`
	BillingPeriodEnd   *string `json:"billingPeriodEnd"`
	IssuedAt      time.Time  `json:"issuedAt"`
	PaidAt        *time.Time `json:"paidAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func toDTO(inv Invoice) InvoiceDTO {
	dto := InvoiceDTO{
		ID:            inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		BookingID:     inv.BookingID,
		BookingNumber: inv.BookingNumber,
		CustomerName:  inv.CustomerName,
		CustomerEmail: inv.CustomerEmail,
		Address:       inv.Address,
		ServiceType:   inv.ServiceType,
		ServiceName:   inv.ServiceName,
		Subtotal:      inv.Subtotal,
		TaxRate:       inv.TaxRate,
		TaxAmount:     inv.TaxAmount,
		Total:         inv.Total,
		Currency:      inv.Currency,
		Status:        inv.Status,
		ContractID:    inv.ContractID,
		IdempotencyKey: inv.IdempotencyKey,
		IssuedAt:      inv.IssuedAt,
		PaidAt:        inv.PaidAt,
		CreatedAt:     inv.CreatedAt,
	}
	if inv.BillingPeriodStart != nil {
		v := inv.BillingPeriodStart.Format("2006-01-02")
		dto.BillingPeriodStart = &v
	}
	if inv.BillingPeriodEnd != nil {
		v := inv.BillingPeriodEnd.Format("2006-01-02")
		dto.BillingPeriodEnd = &v
	}
	return dto
}

type CreateInvoiceRequest struct {
	BookingID int64    `json:"bookingId"`
	Subtotal  *float64 `json:"subtotal,omitempty"`
	// Optional contract billing: when provided, the invoice is linked to the
	// contract and the idempotency key guarantees a retried scheduler run
	// returns the existing invoice instead of billing the period twice.
	ContractID         *int64  `json:"contractId"`
	IdempotencyKey     *string `json:"idempotencyKey"`
	BillingPeriodStart *string `json:"billingPeriodStart"`
	BillingPeriodEnd   *string `json:"billingPeriodEnd"`
}

func (r *CreateInvoiceRequest) Validate() error {
	if r.BookingID < 1 {
		return response.NewAPIError(400, "bookingId is required")
	}
	if r.Subtotal != nil && *r.Subtotal < 0 {
		return response.NewAPIError(400, "subtotal must be zero or greater")
	}
	if r.IdempotencyKey != nil {
		key := strings.TrimSpace(*r.IdempotencyKey)
		if key == "" {
			return response.NewAPIError(400, "idempotencyKey cannot be blank")
		}
		if len(key) > 128 {
			return response.NewAPIError(400, "idempotencyKey is too long")
		}
		r.IdempotencyKey = &key
	}
	if r.BillingPeriodStart != nil {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*r.BillingPeriodStart)); err != nil {
			return response.NewAPIError(400, "billingPeriodStart must be YYYY-MM-DD")
		}
	}
	if r.BillingPeriodEnd != nil {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*r.BillingPeriodEnd)); err != nil {
			return response.NewAPIError(400, "billingPeriodEnd must be YYYY-MM-DD")
		}
	}
	return nil
}

type UpdateInvoiceRequest struct {
	Status *string `json:"status"`
}

func (r *UpdateInvoiceRequest) Validate() error {
	if r.Status != nil && !Status(strings.TrimSpace(*r.Status)).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	return nil
}

func (r *UpdateInvoiceRequest) IsEmpty() bool {
	return r.Status == nil
}