package invoices

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/pkg/mailer"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

// RecordPayment takes a payment against an invoice (full or partial) and
// issues the receipt for it, then sends the receipt to the customer.
func (s *Service) RecordPayment(ctx context.Context, invoiceID int64, req RecordPaymentRequest) (Invoice, Receipt, error) {
	if err := req.Validate(); err != nil {
		return Invoice{}, Receipt{}, err
	}
	inv, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return Invoice{}, Receipt{}, mapRepoError(err)
	}
	amt := inv.BalanceDue()
	if req.Amount != nil {
		amt = round2(*req.Amount)
	}
	if amt <= 0 {
		return Invoice{}, Receipt{}, mapRepoError(checkPayable(inv, 0.01))
	}
	company, _ := s.billingSettings(ctx)
	updated, rc, err := s.repo.RecordPayment(ctx, invoiceID, PaymentInput{
		Amount: amt, Method: req.Method, Reference: req.Reference,
		PaidAt: req.paidAt(), VATRegistered: company.VATRegistered,
	})
	if err != nil {
		return Invoice{}, Receipt{}, mapRepoError(err)
	}
	s.deliverReceipt(ctx, rc, updated)
	return updated, rc, nil
}

// Collect is the pay-on-completion flow for one-time customers: it bills the
// booking (reusing its open invoice if there is one) and records
// the payment straight away, so the customer only receives the receipt.
func (s *Service) Collect(ctx context.Context, req CollectRequest) (Invoice, Receipt, error) {
	if err := req.Validate(); err != nil {
		return Invoice{}, Receipt{}, err
	}
	inv, err := s.repo.GetActiveByBookingID(ctx, req.BookingID)
	switch {
	case errors.Is(err, ErrNotFound):
		inv, err = s.create(ctx, CreateInvoiceRequest{
			BookingID: req.BookingID, Subtotal: req.Subtotal, WithholdingRate: req.WithholdingRate,
		}, false)
		if err != nil {
			return Invoice{}, Receipt{}, err
		}
	case err != nil:
		return Invoice{}, Receipt{}, mapRepoError(err)
	case inv.Status == StatusPaid:
		return Invoice{}, Receipt{}, response.NewAPIError(409, "this booking is already paid ("+inv.InvoiceNumber+")")
	}
	return s.RecordPayment(ctx, inv.ID, req.RecordPaymentRequest)
}

// PayBooking records a paid payment made from the Payments page against the
// booking's open invoice. handled=false means the booking has no open
// invoice, and the caller records a stand-alone payment instead.
func (s *Service) PayBooking(ctx context.Context, bookingNumber string, amount float64, method, reference string) (int64, bool, error) {
	if strings.TrimSpace(bookingNumber) == "" {
		return 0, false, nil
	}
	id, err := s.repo.OpenInvoiceIDForBooking(ctx, bookingNumber)
	if errors.Is(err, ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	_, rc, err := s.RecordPayment(ctx, id, RecordPaymentRequest{Amount: &amount, Method: method, Reference: reference})
	if err != nil {
		return 0, true, err
	}
	return rc.PaymentID, true, nil
}

// SettlePayment is called when a pending payment becomes paid: the invoice
// it belongs to is credited and a receipt is issued.
func (s *Service) SettlePayment(ctx context.Context, paymentID int64) error {
	company, _ := s.billingSettings(ctx)
	out, err := s.repo.SettlePayment(ctx, paymentID, company.VATRegistered)
	if err != nil {
		return mapRepoError(err)
	}
	if out.Receipt != nil && out.Invoice != nil {
		s.deliverReceipt(ctx, *out.Receipt, *out.Invoice)
	}
	return nil
}

// RefundPayment cancels the payment's receipt and re-opens its invoice.
func (s *Service) RefundPayment(ctx context.Context, paymentID int64) error {
	if _, err := s.repo.RefundPayment(ctx, paymentID); err != nil {
		return mapRepoError(err)
	}
	return nil
}

func (s *Service) ListReceipts(ctx context.Context, params pagination.Params, invoiceID int64) ([]Receipt, int, error) {
	return s.repo.ListReceipts(ctx, params, invoiceID)
}

func (s *Service) GetReceipt(ctx context.Context, id int64) (Receipt, error) {
	rc, err := s.repo.GetReceipt(ctx, id)
	if err != nil {
		return Receipt{}, mapRepoError(err)
	}
	return rc, nil
}

// ReceiptPDF renders a receipt with the current company letterhead.
func (s *Service) ReceiptPDF(ctx context.Context, id int64) (Receipt, []byte, error) {
	rc, err := s.repo.GetReceipt(ctx, id)
	if err != nil {
		return Receipt{}, nil, mapRepoError(err)
	}
	inv, err := s.repo.GetByID(ctx, rc.InvoiceID)
	if err != nil {
		return Receipt{}, nil, mapRepoError(err)
	}
	company, pay := s.billingSettings(ctx)
	inv.SiteName = s.repo.SiteNameForBooking(ctx, inv.BookingID)
	pdf, err := renderReceiptPDF(rc, inv, company, pay)
	if err != nil {
		return Receipt{}, nil, fmt.Errorf("render receipt %d: %w", id, err)
	}
	return rc, pdf, nil
}

// SendReceipt (re)sends a receipt to the customer by email.
func (s *Service) SendReceipt(ctx context.Context, id int64) error {
	rc, pdf, err := s.ReceiptPDF(ctx, id)
	if err != nil {
		return err
	}
	if rc.Status != ReceiptValid {
		return response.NewAPIError(422, "a cancelled receipt cannot be emailed")
	}
	if strings.TrimSpace(rc.CustomerEmail) == "" {
		return response.NewAPIError(422, "invoice has no customer email")
	}
	mail := s.mailer()
	if mail == nil || !mail.Enabled() {
		return response.NewAPIError(422, "smtp is not configured")
	}
	subject := receiptSubject(rc)
	if err := mail.Send(ctx, rc.CustomerEmail, subject, receiptEmailBody(rc),
		&mailer.Attachment{FileName: rc.ReceiptNumber + ".pdf", Data: pdf}); err != nil {
		s.record(ctx, notifications.EventPaymentReceived, rc.CustomerEmail, subject, "failed", err.Error())
		return fmt.Errorf("send receipt %d email: %w", rc.ID, err)
	}
	s.record(ctx, notifications.EventPaymentReceived, rc.CustomerEmail, subject, "sent", "")
	return nil
}

// deliverReceipt sends a new receipt by LINE and email, best-effort.
func (s *Service) deliverReceipt(ctx context.Context, rc Receipt, inv Invoice) {
	if s.notifier != nil {
		if name, lineID, err := s.repo.LineContactForBooking(ctx, rc.BookingID); err == nil && lineID != "" {
			text := fmt.Sprintf("🧾 ได้รับชำระเงินเรียบร้อยแล้ว ขอบคุณค่ะ\nใบเสร็จ: %s\nใบแจ้งหนี้: %s\nยอดชำระ: %s %s",
				rc.ReceiptNumber, rc.InvoiceNumber, rc.Currency, amount(rc.Amount))
			if bal := inv.BalanceDue(); bal > 0 {
				text += fmt.Sprintf("\nยอดคงค้าง: %s %s", rc.Currency, amount(bal))
			}
			s.notifier.EmitLINE(ctx, notifications.EventPaymentReceived, lineID, name, text)
		}
	}
	company, pay := s.billingSettings(ctx)
	inv.SiteName = s.repo.SiteNameForBooking(ctx, inv.BookingID)
	s.sendMail(ctx, notifications.EventPaymentReceived, rc.CustomerEmail, receiptSubject(rc), receiptEmailBody(rc),
		rc.ReceiptNumber, func() ([]byte, error) { return renderReceiptPDF(rc, inv, company, pay) })
}

func receiptSubject(rc Receipt) string {
	return "Receipt " + rc.ReceiptNumber + " — Invoice " + rc.InvoiceNumber
}

func receiptEmailBody(rc Receipt) string {
	return `<div style="font-family:Arial,sans-serif;color:#1e293b;max-width:480px;margin:0 auto;">` +
		`<h2 style="margin-bottom:4px;">Receipt ` + htmlEscape(rc.ReceiptNumber) + `</h2>` +
		`<p style="color:#64748b;margin-top:0;">Invoice ` + htmlEscape(rc.InvoiceNumber) + ` · Booking ` + htmlEscape(rc.BookingNumber) + `</p>` +
		`<table style="width:100%;border:1px solid #e2e8f0;border-radius:8px;font-size:14px;">` +
		`<tr><td style="padding:8px 12px;">Service</td><td style="padding:8px 12px;font-weight:600;">` + htmlEscape(rc.ServiceName) + `</td></tr>` +
		`<tr style="background:#f8fafc;"><td style="padding:8px 12px;">Amount received</td><td style="padding:8px 12px;font-weight:700;">` + moneyHTML(rc.Currency, rc.Amount) + `</td></tr>` +
		`<tr><td style="padding:8px 12px;">Date</td><td style="padding:8px 12px;">` + rc.PaidAt.Format("02/01/2006") + `</td></tr>` +
		`</table>` +
		`<p style="color:#94a3b8;font-size:12px;margin-top:24px;">Thank you for your payment!</p>` +
		`</div>`
}

// RecordPaymentRequest is a payment received against an invoice. Amount
// defaults to the full balance due.
type RecordPaymentRequest struct {
	Amount    *float64 `json:"amount"`
	Method    string   `json:"method"`
	Reference string   `json:"reference"`
	PaidAt    *string  `json:"paidAt"`
}

func (r *RecordPaymentRequest) Validate() error {
	r.Method = strings.TrimSpace(r.Method)
	r.Reference = strings.TrimSpace(r.Reference)
	if r.Method == "" {
		return response.NewAPIError(400, "method is required")
	}
	if r.Amount != nil && *r.Amount <= 0 {
		return response.NewAPIError(400, "amount must be greater than zero")
	}
	if len(r.Reference) > 200 {
		return response.NewAPIError(400, "reference is too long")
	}
	if r.PaidAt != nil && strings.TrimSpace(*r.PaidAt) != "" {
		if _, ok := parsePaidAt(*r.PaidAt); !ok {
			return response.NewAPIError(400, "paidAt must be YYYY-MM-DD or an RFC3339 timestamp")
		}
	}
	return nil
}

func (r RecordPaymentRequest) paidAt() time.Time {
	if r.PaidAt != nil {
		if t, ok := parsePaidAt(*r.PaidAt); ok {
			return t
		}
	}
	return time.Now()
}

// parsePaidAt accepts a date (the day the money arrived) or a timestamp. A
// bare date today means "now", so receipts recorded today sort correctly.
func parsePaidAt(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, true
	}
	d, err := time.ParseInLocation("2006-01-02", v, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	now := time.Now()
	if d.Year() == now.Year() && d.YearDay() == now.YearDay() {
		return now, true
	}
	if d.After(now) {
		return time.Time{}, false
	}
	return d.Add(12 * time.Hour), true
}

// CollectRequest bills a booking and records its payment at once.
type CollectRequest struct {
	BookingID       int64    `json:"bookingId"`
	Subtotal        *float64 `json:"subtotal"`
	WithholdingRate *float64 `json:"withholdingRate"`
	RecordPaymentRequest
}

func (r *CollectRequest) Validate() error {
	if r.BookingID < 1 {
		return response.NewAPIError(400, "bookingId is required")
	}
	if r.Subtotal != nil && *r.Subtotal <= 0 {
		return response.NewAPIError(400, "subtotal must be greater than zero")
	}
	if r.WithholdingRate != nil && (*r.WithholdingRate < 0 || *r.WithholdingRate > 15) {
		return response.NewAPIError(400, "withholdingRate must be between 0 and 15 percent")
	}
	return r.RecordPaymentRequest.Validate()
}
