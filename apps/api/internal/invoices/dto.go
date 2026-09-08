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
	IssuedAt      time.Time  `json:"issuedAt"`
	PaidAt        *time.Time `json:"paidAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func toDTO(inv Invoice) InvoiceDTO {
	return InvoiceDTO{
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
		IssuedAt:      inv.IssuedAt,
		PaidAt:        inv.PaidAt,
		CreatedAt:     inv.CreatedAt,
	}
}

type CreateInvoiceRequest struct {
	BookingID int64    `json:"bookingId"`
	Subtotal  *float64 `json:"subtotal,omitempty"`
}

func (r *CreateInvoiceRequest) Validate() error {
	if r.BookingID < 1 {
		return response.NewAPIError(400, "bookingId is required")
	}
	if r.Subtotal != nil && *r.Subtotal < 0 {
		return response.NewAPIError(400, "subtotal must be zero or greater")
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