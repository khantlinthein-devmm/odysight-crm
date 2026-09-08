package invoices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/internal/settings"
	"github.com/odysight/crm/pkg/mailer"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

// Emailer abstracts the SMTP sender so the service stays testable.
type Emailer interface {
	Enabled() bool
	Send(ctx context.Context, to, subject, bodyHTML string, att *mailer.Attachment) error
}

type Service struct {
	repo     *Repository
	settings *settings.Service
	mailer   func() Emailer
	notifier *notifications.Service
}

func NewService(repo *Repository, settingsSvc *settings.Service, newMailer func() Emailer, notifier *notifications.Service) *Service {
	return &Service{repo: repo, settings: settingsSvc, mailer: newMailer, notifier: notifier}
}

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Invoice, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (Invoice, error) {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Invoice{}, mapRepoError(err)
	}
	return inv, nil
}

// PDF renders the branded invoice PDF, incorporating company settings.
func (s *Service) PDF(ctx context.Context, id int64) (Invoice, []byte, error) {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Invoice{}, nil, mapRepoError(err)
	}
	pdfBytes, err := s.renderPDF(ctx, inv)
	if err != nil {
		return Invoice{}, nil, err
	}
	return inv, pdfBytes, nil
}

// renderPDF renders an invoice PDF with company branding from settings.
func (s *Service) renderPDF(ctx context.Context, inv Invoice) ([]byte, error) {
	var company settings.Company
	if raw, err := s.settings.GetAll(ctx); err == nil {
		if v, ok := raw[settings.KeyCompany]; ok {
			_ = json.Unmarshal(v, &company)
		}
	}
	return renderInvoicePDF(inv, company)
}

// Create bills a completed booking, pricing it from the service catalog.
func (s *Service) Create(ctx context.Context, req CreateInvoiceRequest) (Invoice, error) {
	if err := req.Validate(); err != nil {
		return Invoice{}, err
	}

	b, err := s.repo.BookingForInvoice(ctx, req.BookingID)
	if err != nil {
		return Invoice{}, mapRepoError(err)
	}
	if !b.Completed {
		return Invoice{}, response.NewAPIError(422, "only completed bookings can be invoiced")
	}

	catalog, taxRate, currency, err := s.pricing(ctx)
	if err != nil {
		return Invoice{}, err
	}

	serviceName := b.ServiceType
	basePrice := 0.0
	if item, ok := catalog[b.ServiceType]; ok {
		serviceName = item.Name
		basePrice = item.BasePrice
	}

	subtotal := basePrice
	if req.Subtotal != nil {
		subtotal = *req.Subtotal
	}
	taxAmount := round2(subtotal * taxRate / 100)

	inv := Invoice{
		BookingID:     b.ID,
		BookingNumber: b.BookingNumber,
		CustomerName:  b.CustomerName,
		CustomerEmail: b.CustomerEmail,
		Address:       b.Address,
		ServiceType:   b.ServiceType,
		ServiceName:   serviceName,
		Subtotal:      round2(subtotal),
		TaxRate:       taxRate,
		TaxAmount:     taxAmount,
		Total:         round2(subtotal + taxAmount),
		Currency:      currency,
		Status:        StatusIssued,
	}

	created, err := s.repo.Create(ctx, inv)
	if err != nil {
		return Invoice{}, mapRepoError(err)
	}
	s.deliver(ctx, created, "Invoice",
		fmt.Sprintf("Your invoice %s for booking %s is ready. Total: %s %s.",
			created.InvoiceNumber, created.BookingNumber, created.Currency,
			fmt.Sprintf("%0.2f", created.Total)))
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateInvoiceRequest) (Invoice, error) {
	if err := req.Validate(); err != nil {
		return Invoice{}, err
	}
	if req.IsEmpty() {
		return Invoice{}, response.NewAPIError(400, "at least one field must be provided")
	}

	next := Status(strings.TrimSpace(*req.Status))
	updated, err := s.repo.Update(ctx, id, next)
	if err != nil {
		return Invoice{}, mapUpdateError(err)
	}
	if updated.Status == StatusPaid {
		s.deliver(ctx, updated, "Receipt",
			fmt.Sprintf("Payment received for invoice %s (booking %s). Total: %s %s.",
				updated.InvoiceNumber, updated.BookingNumber, updated.Currency,
				fmt.Sprintf("%0.2f", updated.Total)))
	}
	return updated, nil
}

// SendEmail explicitly (re)sends the invoice/Pdf to the customer.
func (s *Service) SendEmail(ctx context.Context, id int64) error {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapRepoError(err)
	}
	if inv.Status == StatusVoid {
		return response.NewAPIError(422, "a void invoice cannot be emailed")
	}
	if strings.TrimSpace(inv.CustomerEmail) == "" {
		return response.NewAPIError(422, "invoice has no customer email")
	}
	mail := s.mailer()
	if mail == nil || !mail.Enabled() {
		return response.NewAPIError(422, "smtp is not configured")
	}
	pdf, err := s.renderPDF(ctx, inv)
	if err != nil {
		return fmt.Errorf("render invoice %d pdf: %w", inv.ID, err)
	}
	err = mail.Send(ctx, inv.CustomerEmail, "Invoice "+inv.InvoiceNumber,
		invoiceEmailBody(inv), &mailer.Attachment{FileName: inv.InvoiceNumber + ".pdf", Data: pdf})
	if err != nil {
		s.record(ctx, notifications.EventInvoiceIssued, inv, "failed", err.Error())
		return fmt.Errorf("send invoice %d email: %w", inv.ID, err)
	}
	s.record(ctx, notifications.EventInvoiceIssued, inv, "sent", "")
	return nil
}

// MarkPaidForBooking is invoked by the payments flow when a payment becomes paid.
// It settles the booking's invoice and emails the receipt. A payment recorded
// for a booking without an invoice is valid and is not an error.
func (s *Service) MarkPaidForBooking(ctx context.Context, bookingNumber string) error {
	if err := s.repo.MarkPaidForBooking(ctx, bookingNumber); err != nil {
		return err
	}
	inv, err := s.repo.GetByBookingNumber(ctx, bookingNumber)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if inv.Status == StatusPaid {
		s.deliver(ctx, inv, "Receipt",
			fmt.Sprintf("Payment received for invoice %s (booking %s). Total: %s %s.",
				inv.InvoiceNumber, inv.BookingNumber, inv.Currency,
				fmt.Sprintf("%0.2f", inv.Total)))
	}
	return nil
}

// deliver sends invoice/receipt emails best-effort: missing SMTP config or an
// absent customer email is logged (slog + notification log), never fatal to
// the request.
func (s *Service) deliver(ctx context.Context, inv Invoice, kind, body string) {
	if inv.Status == StatusVoid {
		return
	}
	eventType := notifications.EventInvoiceIssued
	subject := "Invoice " + inv.InvoiceNumber
	if kind == "Receipt" {
		eventType = notifications.EventPaymentReceived
		subject = "Payment received — Invoice " + inv.InvoiceNumber
	}

	if strings.TrimSpace(inv.CustomerEmail) == "" {
		slog.Debug("invoice email skipped: no customer email", "invoice", inv.InvoiceNumber)
		s.record(ctx, eventType, inv, "skipped", "no customer email")
		return
	}
	mail := s.mailer()
	if mail == nil || !mail.Enabled() {
		slog.Debug("invoice email skipped: smtp not configured", "invoice", inv.InvoiceNumber)
		s.record(ctx, eventType, inv, "skipped", "smtp not configured")
		return
	}
	pdf, err := s.renderPDF(ctx, inv)
	if err != nil {
		slog.Warn("invoice email skipped: pdf render failed", "invoice", inv.InvoiceNumber, "error", err)
		s.record(ctx, eventType, inv, "failed", "pdf render failed")
		return
	}
	if err := mail.Send(ctx, inv.CustomerEmail, subject, invoiceEmailBody(inv),
		&mailer.Attachment{FileName: inv.InvoiceNumber + ".pdf", Data: pdf}); err != nil {
		slog.Warn("invoice email failed", "invoice", inv.InvoiceNumber, "error", err)
		s.record(ctx, eventType, inv, "failed", err.Error())
		return
	}
	s.record(ctx, eventType, inv, "sent", "")
}

// record writes the email outcome to the notification log (best-effort).
func (s *Service) record(ctx context.Context, eventType string, inv Invoice, status, errMsg string) {
	if s.notifier == nil {
		return
	}
	subject := "Invoice " + inv.InvoiceNumber
	if eventType == notifications.EventPaymentReceived {
		subject = "Payment received — Invoice " + inv.InvoiceNumber
	}
	s.notifier.Record(ctx, notifications.ChannelEmail, eventType, inv.CustomerEmail, subject, status, errMsg)
}

func invoiceEmailBody(inv Invoice) string {
	return `<div style="font-family:Arial,sans-serif;color:#1e293b;max-width:480px;margin:0 auto;">` +
		`<h2 style="margin-bottom:4px;">` + htmlEscape(inv.InvoiceNumber) + `</h2>` +
		`<p style="color:#64748b;margin-top:0;">Booking ` + htmlEscape(inv.BookingNumber) + `</p>` +
		`<table style="width:100%;border:1px solid #e2e8f0;border-radius:8px;font-size:14px;">` +
		`<tr><td style="padding:8px 12px;">Service</td><td style="padding:8px 12px;font-weight:600;">` + htmlEscape(inv.ServiceName) + `</td></tr>` +
		`<tr style="background:#f8fafc;"><td style="padding:8px 12px;">Total</td><td style="padding:8px 12px;font-weight:700;">` + moneyHTML(inv.Currency, inv.Total) + `</td></tr>` +
		`<tr><td style="padding:8px 12px;">Status</td><td style="padding:8px 12px;">` + htmlEscape(string(inv.Status)) + `</td></tr>` +
		`</table>` +
		`<p style="color:#94a3b8;font-size:12px;margin-top:24px;">Thank you for your business!</p>` +
		`</div>`
}

func moneyHTML(currency string, amount float64) string {
	return fmt.Sprintf("%s %0.2f", htmlEscape(currency), amount)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}

// pricing reads the effective service catalog, tax rate, currency and SMTP config.
func (s *Service) pricing(ctx context.Context) (map[string]settings.ServiceItem, float64, string, error) {
	raw, err := s.settings.GetAll(ctx)
	if err != nil {
		return nil, 0, "", fmt.Errorf("load settings: %w", err)
	}
	catalog := map[string]settings.ServiceItem{}
	if items, ok := raw[settings.KeyServices]; ok {
		var list []settings.ServiceItem
		if err := json.Unmarshal(items, &list); err != nil {
			return nil, 0, "", fmt.Errorf("parse service catalog: %w", err)
		}
		for _, it := range list {
			catalog[it.ID] = it
		}
	}
	taxRate := 0.0
	if v, ok := raw[settings.KeyPayments]; ok {
		var p settings.PaymentSettings
		if err := json.Unmarshal(v, &p); err != nil {
			return nil, 0, "", fmt.Errorf("parse payment settings: %w", err)
		}
		taxRate = p.TaxRatePercent
	}
	currency := "THB"
	if v, ok := raw[settings.KeyLocalization]; ok {
		var loc settings.Localization
		if err := json.Unmarshal(v, &loc); err != nil {
			return nil, 0, "", fmt.Errorf("parse localization settings: %w", err)
		}
		currency = strings.ToUpper(loc.Currency)
	}
	return catalog, taxRate, currency, nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func mapRepoError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return response.NewAPIError(404, "invoice not found")
	case errors.Is(err, ErrBookingNotFound):
		return response.NewAPIError(404, "booking not found")
	case errors.Is(err, ErrBookingNotCompleted):
		return response.NewAPIError(422, "only completed bookings can be invoiced")
	case errors.Is(err, ErrActiveInvoiceExists):
		return response.NewAPIError(409, "booking already has an active invoice")
	default:
		return err
	}
}

func mapUpdateError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "invoice not found")
	}
	var apiErr *response.APIError
	if errors.As(err, &apiErr) {
		return err
	}
	return response.NewAPIError(400, err.Error())
}