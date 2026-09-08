package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Defaults returns the factory settings used to seed fresh databases.
func Defaults() map[string]string {
	return map[string]string{
		KeyCompany:       `{"name":"Smile Clean","phone":"","address":"","invoiceFooter":"Thank you for your business!","logoUrl":""}`,
		KeyLocalization:  `{"timezone":"Asia/Bangkok","dateFormat":"DD/MM/YYYY","language":"en","currency":"THB"}`,
		KeyBooking:       `{"defaultDurationMinutes":120,"bufferMinutes":30,"workStart":"08:00","workEnd":"18:00","holidays":[]}`,
		KeyPayments:      `{"taxRatePercent":7,"methods":["cash","bank_transfer","promptpay","credit_card","line_pay","online_wallet"]}`,
		KeyServices:      `[{"id":"house_cleaning","name":"House Cleaning","durationMinutes":180,"basePrice":1500,"active":true},{"id":"condo_cleaning","name":"Condo Cleaning","durationMinutes":120,"basePrice":1200,"active":true},{"id":"deep_cleaning","name":"Deep Cleaning","durationMinutes":240,"basePrice":2500,"active":true},{"id":"move_in_out","name":"Move In/Out","durationMinutes":300,"basePrice":3000,"active":true},{"id":"after_renovation","name":"After Renovation","durationMinutes":240,"basePrice":2800,"active":true},{"id":"office_cleaning","name":"Office Cleaning","durationMinutes":210,"basePrice":2200,"active":true},{"id":"junk_removal","name":"Junk Removal","durationMinutes":120,"basePrice":1000,"active":true},{"id":"aircon_service","name":"Aircon Service","durationMinutes":90,"basePrice":800,"active":true}]`,
		KeyNotifications: `{"newLeadEmail":true,"bookingChangeEmail":true,"paymentFailEmail":true,"overdueReminderEmail":true,"overdueReminderDays":7,"recipients":[]}`,
		KeySMTP:          `{"enabled":false,"host":"","port":587,"username":"","password":"","fromEmail":"","fromName":"","encryption":"starttls"}`,
		KeySMS:           `{"enabled":false,"provider":"","webhookUrl":"","apiKey":"","fromNumber":""}`,
		KeyWhatsApp:      `{"enabled":false,"provider":"","webhookUrl":"","apiKey":"","fromNumber":""}`,
	}
}

// GetAll returns effective settings (stored values overlaid on defaults).
// The SMTP password is masked (write-only) so secrets never leave the API.
func (s *Service) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	stored, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]json.RawMessage, len(KnownKeys()))
	for _, k := range KnownKeys() {
		if v, ok := stored[k]; ok {
			out[k] = v
			continue
		}
		out[k] = json.RawMessage(Defaults()[k])
	}
	s.backfill(out)
	return s.maskSMTP(s.maskSMS(out)), nil
}

// backfill overlays factory defaults for keys that older stored JSON lacks
// (e.g. settings rows written before a setting was introduced). Stored values
// are preserved; only genuinely missing fields change.
func (s *Service) backfill(out map[string]json.RawMessage) {
	raw, ok := out[KeyNotifications]
	if !ok {
		return
	}
	var storedMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &storedMap); err != nil {
		return
	}
	var defaultMap map[string]json.RawMessage
	if err := json.Unmarshal(json.RawMessage(Defaults()[KeyNotifications]), &defaultMap); err != nil {
		return
	}
	changed := false
	for k, dv := range defaultMap {
		if _, ok := storedMap[k]; !ok {
			storedMap[k] = dv
			changed = true
		}
	}
	if changed {
		if encoded, err := json.Marshal(storedMap); err == nil {
			out[KeyNotifications] = encoded
		}
	}
}

// maskSMTP blanks the stored SMTP password in a response payload.
func (s *Service) maskSMTP(out map[string]json.RawMessage) map[string]json.RawMessage {
	raw, ok := out[KeySMTP]
	if !ok {
		return out
	}
	var cfg SMTPSettings
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return out
	}
	cfg.Password = ""
	encoded, _ := json.Marshal(cfg)
	out[KeySMTP] = encoded
	return out
}

// maskSMS blanks the stored SMS/WhatsApp API key in a response payload.
func (s *Service) maskSMS(out map[string]json.RawMessage) map[string]json.RawMessage {
	for _, key := range []string{KeySMS, KeyWhatsApp} {
		raw, ok := out[key]
		if !ok {
			continue
		}
		var cfg SMSSettings
		if err := json.Unmarshal(raw, &cfg); err != nil {
			continue
		}
		if cfg.APIKey != "" {
			cfg.APIKey = "••••••••" // write-only placeholder
			encoded, _ := json.Marshal(cfg)
			out[key] = encoded
		}
	}
	return out
}

// SMTPConfig returns the stored SMTP configuration including its real password.
// Only callers authorized to send mail (not the public settings reader) use this.
func (s *Service) SMTPConfig(ctx context.Context) (SMTPSettings, error) {
	stored, err := s.repo.GetAll(ctx)
	if err != nil {
		return SMTPSettings{}, err
	}
	raw, ok := stored[KeySMTP]
	if !ok {
		raw = json.RawMessage(Defaults()[KeySMTP])
	}
	var cfg SMTPSettings
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return SMTPSettings{}, fmt.Errorf("parse smtp settings: %w", err)
	}
	return cfg, nil
}

// SMSConfig returns the stored SMS provider configuration including the real
// API key. Only internal senders (the notification service) use this.
func (s *Service) SMSConfig(ctx context.Context, key string) (SMSSettings, error) {
	if key != KeySMS && key != KeyWhatsApp {
		return SMSSettings{}, fmt.Errorf("unknown sms key: %s", key)
	}
	stored, err := s.repo.GetAll(ctx)
	if err != nil {
		return SMSSettings{}, err
	}
	raw, ok := stored[key]
	if !ok {
		raw = json.RawMessage(Defaults()[key])
	}
	var cfg SMSSettings
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return SMSSettings{}, fmt.Errorf("parse %s settings: %w", key, err)
	}
	return cfg, nil
}

// Update validates and stores the provided keys, then returns effective settings.
func (s *Service) Update(ctx context.Context, raw map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, response.NewAPIError(400, "no settings provided")
	}

	// Write-only SMTP password & SMS keys: blank/masked values keep the stored one.
	for _, key := range []string{KeySMTP, KeySMS, KeyWhatsApp} {
		v, ok := raw[key]
		if !ok {
			continue
		}
		if key == KeySMTP {
			var incoming SMTPSettings
			if err := json.Unmarshal(v, &incoming); err != nil {
				return nil, response.NewAPIError(400, "smtp: invalid JSON")
			}
			if incoming.Password == "" {
				stored, err := s.repo.GetAll(ctx)
				if err != nil {
					return nil, err
				}
				if cur, ok := stored[KeySMTP]; ok {
					var old SMTPSettings
					if err := json.Unmarshal(cur, &old); err == nil {
						incoming.Password = old.Password
						encoded, _ := json.Marshal(incoming)
						raw[KeySMTP] = encoded
					}
				}
			}
			continue
		}
		var incoming SMSSettings
		if err := json.Unmarshal(v, &incoming); err != nil {
			return nil, response.NewAPIError(400, key+": invalid JSON")
		}
		if incoming.APIKey == "" || incoming.APIKey == "••••••••" {
			stored, err := s.repo.GetAll(ctx)
			if err != nil {
				return nil, err
			}
			if cur, ok := stored[key]; ok {
				var old SMSSettings
				if err := json.Unmarshal(cur, &old); err == nil {
					incoming.APIKey = old.APIKey
					encoded, _ := json.Marshal(incoming)
					raw[key] = encoded
				}
			}
		}
	}

	for k, v := range raw {
		if !IsKnownKey(k) {
			return nil, response.NewAPIError(400, "unknown setting: "+k)
		}
		if err := validateKey(k, v); err != nil {
			return nil, err
		}
	}
	for k, v := range raw {
		if err := s.repo.Upsert(ctx, k, v); err != nil {
			return nil, err
		}
	}
	return s.GetAll(ctx)
}

func validateKey(key string, raw json.RawMessage) error {
	switch key {
	case KeyCompany:
		var v Company
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "company: invalid JSON")
		}
		return validateCompany(v)
	case KeyLocalization:
		var v Localization
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "localization: invalid JSON")
		}
		return validateLocalization(v)
	case KeyBooking:
		var v BookingDefaults
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "booking: invalid JSON")
		}
		if v.Holidays == nil {
			v.Holidays = []string{}
		}
		return validateBooking(v)
	case KeyPayments:
		var v PaymentSettings
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "payments: invalid JSON")
		}
		return validatePayments(v)
	case KeyServices:
		var items []ServiceItem
		if err := json.Unmarshal(raw, &items); err != nil {
			return response.NewAPIError(400, "services: invalid JSON")
		}
		if len(items) == 0 {
			return response.NewAPIError(400, "services: needs at least one item")
		}
		seen := map[string]bool{}
		for _, it := range items {
			if err := it.validate(); err != nil {
				return err
			}
			if seen[it.ID] {
				return response.NewAPIError(400, "services: duplicate id "+it.ID)
			}
			seen[it.ID] = true
		}
		return nil
	case KeyNotifications:
		var v NotificationSettings
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "notifications: invalid JSON")
		}
		return validateNotifications(v)
	case KeySMTP:
		var v SMTPSettings
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "smtp: invalid JSON")
		}
		return validateSMTP(v)
	case KeySMS:
		var v SMSSettings
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "sms: invalid JSON")
		}
		return validateSMS(v)
	case KeyWhatsApp:
		var v SMSSettings
		if err := json.Unmarshal(raw, &v); err != nil {
			return response.NewAPIError(400, "whatsapp: invalid JSON")
		}
		return validateSMS(v)
	}
	return response.NewAPIError(400, "unknown setting: "+key)
}
