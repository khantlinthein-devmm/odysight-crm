package settings

import (
	"strings"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

// Known settings keys.
const (
	KeyCompany       = "company"
	KeyLocalization  = "localization"
	KeyBooking       = "booking"
	KeyPayments      = "payments"
	KeyServices      = "services"
	KeyNotifications = "notifications"
	KeySMTP          = "smtp"
	KeySMS           = "sms"
	KeyWhatsApp      = "whatsapp"
)

func KnownKeys() []string {
	return []string{KeyCompany, KeyLocalization, KeyBooking, KeyPayments, KeyServices, KeyNotifications, KeySMTP, KeySMS, KeyWhatsApp}
}

func IsKnownKey(k string) bool {
	switch k {
	case KeyCompany, KeyLocalization, KeyBooking, KeyPayments, KeyServices, KeyNotifications, KeySMTP, KeySMS, KeyWhatsApp:
		return true
	}
	return false
}

type Company struct {
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	InvoiceFooter string `json:"invoiceFooter"`
	LogoURL       string `json:"logoUrl"`
}

type Localization struct {
	Timezone   string `json:"timezone"`
	DateFormat string `json:"dateFormat"`
	Language   string `json:"language"`
	Currency   string `json:"currency"`
}

type BookingDefaults struct {
	DefaultDurationMinutes int      `json:"defaultDurationMinutes"`
	BufferMinutes          int      `json:"bufferMinutes"`
	WorkStart              string   `json:"workStart"`
	WorkEnd                string   `json:"workEnd"`
	Holidays               []string `json:"holidays"`
}

type PaymentSettings struct {
	TaxRatePercent float64  `json:"taxRatePercent"`
	Methods        []string `json:"methods"`
}

type ServiceItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	DurationMinutes int     `json:"durationMinutes"`
	BasePrice       float64 `json:"basePrice"`
	Active          bool    `json:"active"`
}

type NotificationSettings struct {
	NewLeadEmail           bool     `json:"newLeadEmail"`
	BookingChangeEmail     bool     `json:"bookingChangeEmail"`
	PaymentFailEmail       bool     `json:"paymentFailEmail"`
	OverdueReminderEmail   bool     `json:"overdueReminderEmail"`
	OverdueReminderDays    int      `json:"overdueReminderDays"`
	Recipients             []string `json:"recipients"`
}

// SMSProvider describes a supported SMS delivery channel.
const (
	SMSProviderHTTP   = "http"   // generic webhook (Twilio-style APIs)
	SMSProviderDisabled = ""     // channel turned off
)

// SMSSettings configures the SMS/WhatsApp provider. provider is "http" when
// enabled. WebhookURL and APIKey target a Twilio/SMG/PromptAPI-style endpoint.
type SMSSettings struct {
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider"`
	WebhookURL string `json:"webhookUrl"`
	APIKey     string `json:"apiKey"`
	FromNumber string `json:"fromNumber"`
}

func (v SMSSettings) Configured() bool {
	return v.Enabled && v.Provider != "" && v.WebhookURL != ""
}

func validateSMS(v SMSSettings) error {
	if !v.Enabled {
		return nil
	}
	if v.Provider == "" {
		return response.NewAPIError(400, "sms.provider is required when enabled")
	}
	if v.WebhookURL == "" {
		return response.NewAPIError(400, "sms.webhookUrl is required when enabled")
	}
	return nil
}

// DefaultOverdueReminderDays is the number of days after an invoice is issued
// before it counts as overdue. It is also the backfill value for databases
// that existed before the setting was introduced.
const DefaultOverdueReminderDays = 7

// SMTPSettings configures the mail server used to deliver invoice emails.
// Encryption is one of: "none", "starttls", "ssl".
type SMTPSettings struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"fromEmail"`
	FromName   string `json:"fromName"`
	Encryption string `json:"encryption"`
}

func (s ServiceItem) validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return response.NewAPIError(400, "services: each item needs an id")
	}
	if strings.TrimSpace(s.Name) == "" {
		return response.NewAPIError(400, "services: each item needs a name")
	}
	if s.DurationMinutes < 15 || s.DurationMinutes > 24*60 {
		return response.NewAPIError(400, "services: durationMinutes must be 15..1440")
	}
	if s.BasePrice < 0 {
		return response.NewAPIError(400, "services: basePrice must be >= 0")
	}
	return nil
}

var allowedCurrencies = map[string]bool{"THB": true, "USD": true, "EUR": true, "GBP": true, "JPY": true, "SGD": true, "MYR": true}
var allowedLanguages = map[string]bool{"en": true, "th": true}
var allowedDateFormats = map[string]bool{"DD/MM/YYYY": true, "MM/DD/YYYY": true, "YYYY-MM-DD": true}

func validateCompany(v Company) error {
	if strings.TrimSpace(v.Name) == "" {
		return response.NewAPIError(400, "company.name is required")
	}
	return nil
}

func validateLocalization(v Localization) error {
	if strings.TrimSpace(v.Timezone) == "" {
		return response.NewAPIError(400, "localization.timezone is required")
	}
	if !allowedDateFormats[v.DateFormat] {
		return response.NewAPIError(400, "localization.dateFormat must be DD/MM/YYYY, MM/DD/YYYY or YYYY-MM-DD")
	}
	if !allowedLanguages[v.Language] {
		return response.NewAPIError(400, "localization.language must be en or th")
	}
	if !allowedCurrencies[v.Currency] {
		return response.NewAPIError(400, "localization.currency is not supported")
	}
	return nil
}

func validateBooking(v BookingDefaults) error {
	if v.DefaultDurationMinutes < 15 || v.DefaultDurationMinutes > 24*60 {
		return response.NewAPIError(400, "booking.defaultDurationMinutes must be 15..1440")
	}
	if v.BufferMinutes < 0 || v.BufferMinutes > 24*60 {
		return response.NewAPIError(400, "booking.bufferMinutes must be 0..1440")
	}
	if !isHHMM(v.WorkStart) || !isHHMM(v.WorkEnd) {
		return response.NewAPIError(400, "booking.workStart/workEnd must be HH:MM")
	}
	return nil
}

func isHHMM(s string) bool {
	if len(s) != 5 || s[2] != ':' {
		return false
	}
	h, m := s[0:2], s[3:5]
	if h < "00" || h > "23" || m < "00" || m > "59" {
		return false
	}
	return true
}

func validatePayments(v PaymentSettings) error {
	if v.TaxRatePercent < 0 || v.TaxRatePercent > 100 {
		return response.NewAPIError(400, "payments.taxRatePercent must be 0..100")
	}
	if len(v.Methods) == 0 {
		return response.NewAPIError(400, "payments.methods needs at least one method")
	}
	for _, m := range v.Methods {
		if strings.TrimSpace(m) == "" {
			return response.NewAPIError(400, "payments.methods must not contain blanks")
		}
	}
	return nil
}

func validateNotifications(v NotificationSettings) error {
	if v.OverdueReminderDays < 1 || v.OverdueReminderDays > 365 {
		return response.NewAPIError(400, "notifications.overdueReminderDays must be 1..365")
	}
	for _, e := range v.Recipients {
		if !validate.Email(strings.ToLower(strings.TrimSpace(e))) {
			return response.NewAPIError(400, "notifications.recipients must be valid emails")
		}
	}
	return nil
}

func validateSMTP(v SMTPSettings) error {
	if !v.Enabled {
		return nil
	}
	if strings.TrimSpace(v.Host) == "" {
		return response.NewAPIError(400, "smtp.host is required when enabled")
	}
	if v.Port < 1 || v.Port > 65535 {
		return response.NewAPIError(400, "smtp.port must be 1..65535")
	}
	switch v.Encryption {
	case "none", "starttls", "ssl":
	default:
		return response.NewAPIError(400, "smtp.encryption must be none, starttls or ssl")
	}
	if !validate.Email(strings.ToLower(strings.TrimSpace(v.FromEmail))) {
		return response.NewAPIError(400, "smtp.fromEmail must be a valid email")
	}
	return nil
}
