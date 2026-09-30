// Package chat is the team's direct messaging: one-to-one conversations
// between staff, with text, photo and voice messages and phone push
// notifications.
package chat

import (
	"time"

	"github.com/odysight/crm/internal/auth"
)

const (
	KindText = "text"
	// KindImage is no longer accepted; older photo messages still display.
	KindImage = "image"
	KindVoice = "voice"
)

// FileRetention is how long chat voice notes are kept on disk.
const FileRetention = 30 * 24 * time.Hour

// IsOffice reports whether a role belongs to the office team, which may
// message anyone. Everyone else needs a shared chat group.
func IsOffice(role auth.Role) bool {
	switch role {
	case auth.RoleSuperAdmin, auth.RoleAdmin, auth.RoleManager, auth.RoleDispatch:
		return true
	}
	return false
}

// CanMessage is the messaging rule: office staff talk to anyone (and anyone
// can reach them); other members need at least one group in common.
func CanMessage(a, b auth.Role, shareGroup bool) bool {
	return IsOffice(a) || IsOffice(b) || shareGroup
}

type Contact struct {
	UserID int64  `json:"userId"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

type Conversation struct {
	ID            int64      `json:"id"`
	Other         Contact    `json:"other"`
	LastMessageAt *time.Time `json:"lastMessageAt"`
	LastKind      string     `json:"lastKind"`
	LastBody      string     `json:"lastBody"`
	LastFromMe    bool       `json:"lastFromMe"`
	Unread        int        `json:"unread"`
	// OtherLastRead lets the sender show "seen" on their messages.
	OtherLastRead int64 `json:"otherLastRead"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversationId"`
	SenderID       *int64    `json:"senderId"`
	Kind           string    `json:"kind"`
	Body           string    `json:"body"`
	FileURL        string    `json:"fileUrl,omitempty"`
	MimeType       string    `json:"mimeType,omitempty"`
	DurationMs     int       `json:"durationMs,omitempty"`
	// Expired: the voice note was removed by the 30-day cleanup.
	Expired bool `json:"expired,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Group struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Members   []Contact `json:"members"`
	CreatedAt time.Time `json:"createdAt"`
}
