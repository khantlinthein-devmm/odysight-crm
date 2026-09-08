package invoices

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/internal/settings"
	"github.com/odysight/crm/pkg/mailer"
)

// ReminderRunner scans for overdue issued invoices and emails each customer a
// reminder with the branded PDF attached. It is safe for a single API instance;
// the reminder_sent_at stamp makes every run idempotent across restarts.
type ReminderRunner struct {
	repo     *Repository
	settings *settings.Service
	mailer   func() Emailer
	notifier *notifications.Service
	interval time.Duration
}

func NewReminderRunner(repo *Repository, settingsSvc *settings.Service, newMailer func() Emailer, notifier *notifications.Service, interval time.Duration) *ReminderRunner {
	return &ReminderRunner{repo: repo, settings: settingsSvc, mailer: newMailer, notifier: notifier, interval: interval}
}

// Run blocks until ctx is cancelled, firing once immediately then every interval.
func (r *ReminderRunner) Run(ctx context.Context) {
	r.fire(ctx)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("overdue reminder runner stopping")
			return
		case <-ticker.C:
			r.fire(ctx)
		}
	}
}

func (r *ReminderRunner) fire(ctx context.Context) {
	if err := r.sendReminders(ctx); err != nil {
		slog.Warn("overdue reminder run failed", "error", err)
	}
}

func (r *ReminderRunner) sendReminders(ctx context.Context) error {
	all, err := r.settings.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	var n settings.NotificationSettings
	if v, ok := all[settings.KeyNotifications]; ok {
		if err := json.Unmarshal(v, &n); err != nil {
			return fmt.Errorf("parse notification settings: %w", err)
		}
	}
	if !n.OverdueReminderEmail {
		return nil
	}

	mails := r.mailer()
	if mails == nil || !mails.Enabled() {
		slog.Debug("overdue reminders skipped: smtp not configured")
		return nil
	}

	grace := settings.DefaultOverdueReminderDays
	if n.OverdueReminderDays >= 1 {
		grace = n.OverdueReminderDays
	}
	threshold := time.Now().Add(-time.Duration(grace) * 24 * time.Hour)

	overdue, err := r.repo.OverdueIssued(ctx, threshold)
	if err != nil {
		return err
	}
	if len(overdue) == 0 {
		return nil
	}

	var company settings.Company
	if v, ok := all[settings.KeyCompany]; ok {
		_ = json.Unmarshal(v, &company)
	}

	sent, noEmail := 0, 0
	for _, inv := range overdue {
		if strings.TrimSpace(inv.CustomerEmail) == "" {
			r.record(ctx, inv, "skipped", "no customer email")
			noEmail++
			continue
		}
		pdf, err := renderInvoicePDF(inv, company)
		if err != nil {
			slog.Warn("overdue reminder pdf render failed", "invoice", inv.InvoiceNumber, "error", err)
			r.record(ctx, inv, "failed", "pdf render failed")
			continue
		}
		subject := "Reminder: invoice " + inv.InvoiceNumber + " is overdue"
		if err := mails.Send(ctx, inv.CustomerEmail, subject,
			reminderEmailBody(inv), &mailer.Attachment{FileName: inv.InvoiceNumber + ".pdf", Data: pdf}); err != nil {
			slog.Warn("overdue reminder email failed", "invoice", inv.InvoiceNumber, "error", err)
			r.record(ctx, inv, "failed", err.Error())
			continue
		}
		r.record(ctx, inv, "sent", "")
		if err := r.repo.MarkReminderSent(ctx, inv.ID); err != nil {
			slog.Warn("overdue reminder stamp failed", "invoice", inv.InvoiceNumber, "error", err)
			continue
		}
		sent++
	}
	slog.Info("overdue reminder scan complete", "overdue", len(overdue), "sent", sent, "no_email", noEmail)
	return nil
}

// record writes an overdue-reminder outcome to the notification log (best-effort).
func (r *ReminderRunner) record(ctx context.Context, inv Invoice, status, errMsg string) {
	if r.notifier == nil {
		return
	}
	r.notifier.Record(ctx, notifications.ChannelEmail, notifications.EventInvoiceOverdue,
		inv.CustomerEmail, "Reminder: invoice "+inv.InvoiceNumber+" is overdue", status, errMsg)
}

func reminderEmailBody(inv Invoice) string {
	return `<div style="font-family:Arial,sans-serif;color:#1e293b;max-width:480px;margin:0 auto;">` +
		`<h2 style="margin-bottom:4px;">Invoice ` + htmlEscape(inv.InvoiceNumber) + ` is overdue</h2>` +
		`<p style="color:#64748b;margin-top:0;">Booking ` + htmlEscape(inv.BookingNumber) + ` · Total due ` + moneyHTML(inv.Currency, inv.Total) + `</p>` +
		`<table style="width:100%;border:1px solid #e2e8f0;border-radius:8px;font-size:14px;">` +
		`<tr><td style="padding:8px 12px;">Invoice date</td><td style="padding:8px 12px;font-weight:600;">` + inv.IssuedAt.Format("02/01/2006") + `</td></tr>` +
		`<tr style="background:#f8fafc;"><td style="padding:8px 12px;">Total due</td><td style="padding:8px 12px;font-weight:700;">` + moneyHTML(inv.Currency, inv.Total) + `</td></tr>` +
		`<tr><td style="padding:8px 12px;">Status</td><td style="padding:8px 12px;">Unpaid</td></tr>` +
		`</table>` +
		`<p style="color:#475569;font-size:13px;margin-top:16px;">Please arrange payment at your earliest convenience. If you have already paid, please ignore this reminder.</p>` +
		`</div>`
}