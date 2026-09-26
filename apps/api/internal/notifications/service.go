package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/odysight/crm/internal/settings"
	"github.com/odysight/crm/pkg/mailer"
	"github.com/odysight/crm/pkg/pagination"
)

// EventType names used in the notification log.
const (
	EventBookingCreated   = "booking.created"
	EventBookingReminder  = "booking.reminder"
	EventInvoiceIssued    = "invoice.issued"
	EventInvoiceOverdue   = "invoice.overdue"
	EventPaymentReceived  = "payment.received"
	EventFeedbackRequest  = "feedback.request"
	EventBookingCompleted = "booking.completed"
)

// LinePusher sends a LINE text message to a user (see line.Client.Push).
type LinePusher interface {
	Push(ctx context.Context, to, text string) error
}

// Emailer abstracts the SMTP sender so the service stays testable.
type Emailer interface {
	Enabled() bool
	Send(ctx context.Context, to, subject, bodyHTML string, att *mailer.Attachment) error
}

// SMSClient sends an SMS / WhatsApp message via a generic ISO-20022 style
// webhook. The concrete HTTP implementation is in PackageNotifications.
type SMSClient interface {
	Send(ctx context.Context, channel, to, message string, cfg settings.SMSSettings) error
}

// Service orchestrates email and SMS/WhatsApp delivery with an audit log.
type Service struct {
	line     LinePusher
	repo     *Repository
	settings *settings.Service
	mailer   func() Emailer
	sms      *HTTPClient
	timeout  time.Duration
}

func NewService(repo *Repository, settingsSvc *settings.Service, newMailer func() Emailer) *Service {
	return &Service{
		repo:     repo,
		settings: settingsSvc,
		mailer:   newMailer,
		sms:      NewHTTPClient(10 * time.Second),
		timeout:  10 * time.Second,
	}
}

// WithLINE enables LINE pushes to customers. Without it EmitLINE logs the
// message as skipped.
func (s *Service) WithLINE(p LinePusher) *Service {
	s.line = p
	return s
}

// EmitLINE pushes a text message to a customer's LINE chat, best-effort, and
// records the outcome in the notification log. The recipient is logged as
// "LINE <name>" because raw LINE user ids mean nothing to staff.
func (s *Service) EmitLINE(ctx context.Context, eventType, lineUserID, displayName, message string) {
	if strings.TrimSpace(lineUserID) == "" {
		return
	}
	recipient := "LINE " + displayName
	log := LogEntry{Channel: ChannelLINE, EventType: eventType, Recipient: recipient, Subject: firstLine(message)}
	if s.line == nil {
		log.Status = StatusSkipped
		log.Error = "LINE not configured"
		_ = s.repo.Log(ctx, log)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.line.Push(ctx, lineUserID, message); err != nil {
		log.Status = StatusFailed
		log.Error = err.Error()
	} else {
		log.Status = StatusSent
	}
	_ = s.repo.Log(ctx, log)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if r := []rune(s); len(r) > 120 {
		s = string(r[:120])
	}
	return s
}

// List returns the notification-log page.
func (s *Service) List(ctx context.Context, params pagination.Params) ([]LogEntryDTO, int, error) {
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]LogEntryDTO, 0, len(items))
	for _, e := range items {
		dtos = append(dtos, toDTO(e))
	}
	return dtos, total, nil
}

// Record writes a delivery record to the notification log without sending.
// It is used by flows that deliver attachments (e.g. invoice PDFs) through
// their own sender but still want the outcome surfaced in the log.
func (s *Service) Record(ctx context.Context, channel, eventType, recipient, subject, status, errMsg string) {
	_ = s.repo.Log(ctx, LogEntry{
		Channel:   channel,
		EventType: eventType,
		Recipient: recipient,
		Subject:   subject,
		Status:    status,
		Error:     errMsg,
	})
}

// Emit emails a message, best-effort honoring the email settings.
func (s *Service) EmitEmail(ctx context.Context, eventType, recipient, subject, bodyHTML string) {
	log := LogEntry{Channel: ChannelEmail, EventType: eventType, Recipient: recipient, Subject: subject}
	m := s.mailer()
	if m == nil || !m.Enabled() {
		log.Status = StatusSkipped
		log.Error = "smtp not configured"
		_ = s.repo.Log(ctx, log)
		return
	}
	if err := m.Send(ctx, recipient, subject, bodyHTML, nil); err != nil {
		log.Status = StatusFailed
		log.Error = err.Error()
	} else {
		log.Status = StatusSent
	}
	_ = s.repo.Log(ctx, log)
}

// NotifyOffice emails every configured notification recipient (Settings →
// Notifications → recipients), best-effort. Every attempt is written to the
// notification log, so the office sees the event in the notification center
// even when SMTP is not configured (logged as skipped) or no recipients are
// set. Used for customer self-service events like portal bookings that
// otherwise arrive silently.
func (s *Service) NotifyOffice(ctx context.Context, eventType, subject, bodyHTML string) {
	recipients := s.officeRecipients(ctx)
	if len(recipients) == 0 {
		s.Record(ctx, ChannelEmail, eventType, "(office recipients)", subject, StatusSkipped, "no notification recipients configured")
		return
	}
	for _, to := range recipients {
		s.EmitEmail(ctx, eventType, to, subject, bodyHTML)
	}
}

func (s *Service) officeRecipients(ctx context.Context) []string {
	all, err := s.settings.GetAll(ctx)
	if err != nil {
		return nil
	}
	raw, ok := all[settings.KeyNotifications]
	if !ok {
		return nil
	}
	var n settings.NotificationSettings
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil
	}
	out := []string{}
	for _, e := range n.Recipients {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// EmitSMS sends an SMS or WhatsApp message through the configured provider.
func (s *Service) EmitSMS(ctx context.Context, channel, eventType, recipient, message string) {
	var key string
	switch channel {
	case ChannelSMS:
		key = settings.KeySMS
	case ChannelWhatsApp:
		key = settings.KeyWhatsApp
	default:
		return
	}
	cfg, err := s.settings.SMSConfig(ctx, key)
	if err != nil {
		cfg = settings.SMSSettings{}
	}
	log := LogEntry{Channel: channel, EventType: eventType, Recipient: recipient, Subject: message}
	if !cfg.Configured() {
		log.Status = StatusSkipped
		log.Error = "provider not configured"
		_ = s.repo.Log(ctx, log)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.sms.Send(ctx, channel, recipient, message, cfg); err != nil {
		log.Status = StatusFailed
		log.Error = err.Error()
	} else {
		log.Status = StatusSent
	}
	_ = s.repo.Log(ctx, log)
}

// HTTPClient posts text messages to the configured webhook URL.
type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{client: &http.Client{Timeout: timeout}}
}

func (c *HTTPClient) Send(ctx context.Context, channel, to, message string, cfg settings.SMSSettings) error {
	type payload struct {
		Channel string `json:"channel"`
		To      string `json:"to"`
		From    string `json:"from"`
		Message string `json:"message"`
	}
	body, err := json.Marshal(payload{Channel: channel, To: to, From: cfg.FromNumber, Message: message})
	if err != nil {
		return fmt.Errorf("encode sms payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build sms request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send %s: %w", channel, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("provider returned status %d", resp.StatusCode)
	}
	return nil
}
