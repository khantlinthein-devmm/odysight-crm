package invoices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/internal/settings"
	"github.com/odysight/crm/pkg/mailer"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/promptpay"
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
	company, pay := s.billingSettings(ctx)
	return renderInvoicePDF(inv, company, pay, s.dueDays(ctx))
}

// dueDays is the payment term printed on invoices: the number of days after
// which an unpaid invoice counts as overdue (Settings → Notifications).
func (s *Service) dueDays(ctx context.Context) int {
	days := settings.DefaultOverdueReminderDays
	if raw, err := s.settings.GetAll(ctx); err == nil {
		var n settings.NotificationSettings
		if v, ok := raw[settings.KeyNotifications]; ok && json.Unmarshal(v, &n) == nil && n.OverdueReminderDays >= 1 {
			days = n.OverdueReminderDays
		}
	}
	return days
}

// billingSettings loads the company identity and payment details printed on
// invoices. Missing or unreadable settings fall back to zero values.
func (s *Service) billingSettings(ctx context.Context) (settings.Company, settings.PaymentSettings) {
	var company settings.Company
	var pay settings.PaymentSettings
	if raw, err := s.settings.GetAll(ctx); err == nil {
		if v, ok := raw[settings.KeyCompany]; ok {
			_ = json.Unmarshal(v, &company)
		}
		if v, ok := raw[settings.KeyPayments]; ok {
			_ = json.Unmarshal(v, &pay)
		}
	}
	return company, pay
}

// PromptPayQR renders the PromptPay QR (PNG) for an invoice's balance due. It
// fails when no PromptPay ID is configured.
func (s *Service) PromptPayQR(ctx context.Context, id int64) ([]byte, error) {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoError(err)
	}
	_, pay := s.billingSettings(ctx)
	if strings.TrimSpace(pay.PromptPayID) == "" {
		return nil, response.NewAPIError(404, "PromptPay is not configured; set it under Settings → Payments")
	}
	if inv.BalanceDue() <= 0 {
		return nil, response.NewAPIError(404, "nothing is due on this invoice")
	}
	payload, err := promptpay.Payload(pay.PromptPayID, inv.BalanceDue())
	if err != nil {
		return nil, response.NewAPIError(422, err.Error())
	}
	return promptpay.PNG(payload, 512)
}

// Create bills a booking, pricing it from the service catalog. It can be
// issued before the job so the customer can pay a deposit against it.
// When req.IdempotencyKey is set, an existing invoice for that key is
// returned instead of creating a duplicate (contract billing retries).
func (s *Service) Create(ctx context.Context, req CreateInvoiceRequest) (Invoice, error) {
	return s.create(ctx, req, true)
}

// create bills the booking; notify=false skips the invoice email/LINE (used
// when the customer pays on the spot and only needs the receipt).
func (s *Service) create(ctx context.Context, req CreateInvoiceRequest, notify bool) (Invoice, error) {
	if err := req.Validate(); err != nil {
		return Invoice{}, err
	}

	if req.IdempotencyKey != nil {
		if existing, err := s.repo.GetByIdempotencyKey(ctx, *req.IdempotencyKey); err == nil {
			return existing, nil
		}
	}

	b, err := s.repo.BookingForInvoice(ctx, req.BookingID)
	if err != nil {
		return Invoice{}, mapRepoError(err)
	}
	if !b.Billable {
		return Invoice{}, response.NewAPIError(422, "a cancelled or no-show booking cannot be invoiced")
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

	// Price: an explicit amount, else the booking's agreed price, else the
	// catalog base price.
	subtotal := basePrice
	if b.Price != nil {
		subtotal = *b.Price
	}
	if req.Subtotal != nil {
		subtotal = *req.Subtotal
	}
	if round2(subtotal) <= 0 {
		return Invoice{}, response.NewAPIError(422,
			"this booking has no price: set a price on the booking or enter an amount")
	}
	taxAmount := round2(subtotal * taxRate / 100)
	// Withholding tax is computed on the pre-VAT amount (Thai WHT rules).
	whtRate := b.WithholdingRate
	if req.WithholdingRate != nil {
		whtRate = *req.WithholdingRate
	}

	inv := Invoice{
		BookingID:         b.ID,
		BookingNumber:     b.BookingNumber,
		CustomerName:      b.CustomerName,
		CustomerEmail:     b.CustomerEmail,
		Address:           b.Address,
		ServiceType:       b.ServiceType,
		ServiceName:       serviceName,
		Subtotal:          round2(subtotal),
		TaxRate:           taxRate,
		TaxAmount:         taxAmount,
		Total:             round2(subtotal + taxAmount),
		Currency:          currency,
		Status:            StatusIssued,
		CustomerTaxID:     b.CustomerTaxID,
		CustomerTaxBranch: b.CustomerTaxBranch,
		WithholdingRate:   whtRate,
		WithholdingAmount: round2(round2(subtotal) * whtRate / 100),
		ContractID:        req.ContractID,
		IdempotencyKey:    req.IdempotencyKey,
	}
	if req.BillingPeriodStart != nil {
		if v, err := time.Parse("2006-01-02", strings.TrimSpace(*req.BillingPeriodStart)); err == nil {
			inv.BillingPeriodStart = &v
		}
	}
	if req.BillingPeriodEnd != nil {
		if v, err := time.Parse("2006-01-02", strings.TrimSpace(*req.BillingPeriodEnd)); err == nil {
			inv.BillingPeriodEnd = &v
		}
	}

	created, err := s.repo.Create(ctx, inv)
	if err != nil {
		// A concurrent retry may have won the insert race: return the winner.
		if req.IdempotencyKey != nil && strings.Contains(err.Error(), "duplicate idempotency key") {
			if existing, gerr := s.repo.GetByIdempotencyKey(ctx, *req.IdempotencyKey); gerr == nil {
				return existing, nil
			}
		}
		return Invoice{}, mapRepoError(err)
	}
	if notify {
		s.deliver(ctx, created)
	}
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
	if next == StatusPaid || next == StatusPartiallyPaid {
		return Invoice{}, response.NewAPIError(422, "record a payment to mark an invoice paid")
	}
	updated, err := s.repo.Update(ctx, id, next)
	if err != nil {
		return Invoice{}, mapUpdateError(err)
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
		s.record(ctx, notifications.EventInvoiceIssued, inv.CustomerEmail, "Invoice "+inv.InvoiceNumber, "failed", err.Error())
		return fmt.Errorf("send invoice %d email: %w", inv.ID, err)
	}
	s.record(ctx, notifications.EventInvoiceIssued, inv.CustomerEmail, "Invoice "+inv.InvoiceNumber, "sent", "")
	return nil
}

// deliver sends the invoice by email and LINE, best-effort: missing SMTP
// config or an absent customer email is logged (slog + notification log),
// never fatal to the request.
func (s *Service) deliver(ctx context.Context, inv Invoice) {
	if inv.Status == StatusVoid {
		return
	}
	s.deliverLINE(ctx, inv)
	subject := "Invoice " + inv.InvoiceNumber
	s.sendMail(ctx, notifications.EventInvoiceIssued, inv.CustomerEmail, subject, invoiceEmailBody(inv),
		inv.InvoiceNumber, func() ([]byte, error) { return s.renderPDF(ctx, inv) })
}

// sendMail emails one document with its PDF attached and logs the outcome.
func (s *Service) sendMail(ctx context.Context, eventType, to, subject, body, docNumber string, render func() ([]byte, error)) {
	if strings.TrimSpace(to) == "" {
		slog.Debug("document email skipped: no customer email", "document", docNumber)
		s.record(ctx, eventType, to, subject, "skipped", "no customer email")
		return
	}
	mail := s.mailer()
	if mail == nil || !mail.Enabled() {
		slog.Debug("document email skipped: smtp not configured", "document", docNumber)
		s.record(ctx, eventType, to, subject, "skipped", "smtp not configured")
		return
	}
	pdf, err := render()
	if err != nil {
		slog.Warn("document email skipped: pdf render failed", "document", docNumber, "error", err)
		s.record(ctx, eventType, to, subject, "failed", "pdf render failed")
		return
	}
	if err := mail.Send(ctx, to, subject, body, &mailer.Attachment{FileName: docNumber + ".pdf", Data: pdf}); err != nil {
		slog.Warn("document email failed", "document", docNumber, "error", err)
		s.record(ctx, eventType, to, subject, "failed", err.Error())
		return
	}
	s.record(ctx, eventType, to, subject, "sent", "")
}

// deliverLINE pushes the invoice summary to the customer's LINE chat when
// they have one, including the PromptPay ID to pay to.
func (s *Service) deliverLINE(ctx context.Context, inv Invoice) {
	if s.notifier == nil {
		return
	}
	name, lineID, err := s.repo.LineContactForBooking(ctx, inv.BookingID)
	if err != nil || lineID == "" {
		return
	}
	_, pay := s.billingSettings(ctx)
	text := fmt.Sprintf("🧾 ใบแจ้งหนี้ %s\nงาน: %s (%s)\nยอดชำระ: %s %s",
		inv.InvoiceNumber, inv.ServiceName, inv.BookingNumber, inv.Currency, amount(inv.BalanceDue()))
	if inv.WithholdingAmount > 0 {
		text += fmt.Sprintf("\n(หักภาษี ณ ที่จ่าย %s%% แล้ว)", trimRate(inv.WithholdingRate))
	}
	if id, ok := promptpay.Normalize(pay.PromptPayID); ok {
		text += "\n\nชำระผ่าน PromptPay: " + id
	}
	if b := strings.TrimSpace(pay.BankAccount); b != "" {
		text += "\nหรือโอนเข้าบัญชี: " + b
	}
	s.notifier.EmitLINE(ctx, notifications.EventInvoiceIssued, lineID, name, text)
}

// record writes the email outcome to the notification log (best-effort).
func (s *Service) record(ctx context.Context, eventType, to, subject, status, errMsg string) {
	if s.notifier == nil {
		return
	}
	s.notifier.Record(ctx, notifications.ChannelEmail, eventType, to, subject, status, errMsg)
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
	// VAT is only charged by a VAT-registered company (Settings → Company).
	var company settings.Company
	if v, ok := raw[settings.KeyCompany]; ok {
		_ = json.Unmarshal(v, &company)
	}
	taxRate := 0.0
	if v, ok := raw[settings.KeyPayments]; ok && company.VATRegistered {
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
	case errors.Is(err, ErrBookingNotBillable):
		return response.NewAPIError(422, "a cancelled or no-show booking cannot be invoiced")
	case errors.Is(err, ErrActiveInvoiceExists):
		return response.NewAPIError(409, "booking already has an active invoice")
	case errors.Is(err, ErrReceiptNotFound):
		return response.NewAPIError(404, "receipt not found")
	case errors.Is(err, ErrPaymentNotFound):
		return response.NewAPIError(404, "payment not found")
	case errors.Is(err, ErrInvoiceFullyPaid):
		return response.NewAPIError(422, "invoice is already fully paid")
	case errors.Is(err, ErrInvoiceNotOpen):
		return response.NewAPIError(422, "a void invoice cannot take payments")
	case errors.Is(err, ErrOverpayment):
		return response.NewAPIError(422, err.Error())
	case errors.Is(err, ErrInvoiceHasPaid):
		return response.NewAPIError(422, "this invoice has payments; refund them before voiding it")
	default:
		return err
	}
}

func mapUpdateError(err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvoiceHasPaid) {
		return mapRepoError(err)
	}
	var apiErr *response.APIError
	if errors.As(err, &apiErr) {
		return err
	}
	return response.NewAPIError(400, err.Error())
}
