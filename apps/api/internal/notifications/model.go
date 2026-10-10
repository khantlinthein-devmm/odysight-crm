package notifications

import "time"

// Channel identifiers for notification delivery.
const (
	ChannelEmail    = "email"
	ChannelSMS      = "sms"
	ChannelWhatsApp = "whatsapp"
	ChannelLINE     = "line"
)

// Status values for the notification log.
const (
	StatusSent    = "sent"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)

type LogEntry struct {
	ID        int64
	Channel   string
	EventType string
	Recipient string
	Subject   string
	Status    string
	Error     string
	CreatedAt time.Time
}

type LogEntryDTO struct {
	ID        int64     `json:"id"`
	Channel   string    `json:"channel"`
	EventType string    `json:"eventType"`
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(e LogEntry) LogEntryDTO {
	return LogEntryDTO{
		ID:        e.ID,
		Channel:   e.Channel,
		EventType: e.EventType,
		Recipient: e.Recipient,
		Subject:   e.Subject,
		Status:    e.Status,
		Error:     e.Error,
		CreatedAt: e.CreatedAt,
	}
}
