package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/webpush"
)

const maxTextRunes = 4000

type Service struct {
	repo    *Repository
	files   *FileStore
	subject string
	events  *Broker

	pushMu sync.Mutex
	sender *webpush.Sender
}

// NewService wires chat. pushSubject is the contact URL/mailto sent to push
// services (the app's own https origin is fine).
func NewService(repo *Repository, files *FileStore, pushSubject string, events *Broker) *Service {
	return &Service{repo: repo, files: files, subject: pushSubject, events: events}
}

// Events exposes the broker for the SSE endpoint.
func (s *Service) Events() *Broker { return s.events }

func (s *Service) publish(ctx context.Context, to []int64, typ string, data any) {
	if s.events == nil {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	s.events.Publish(ctx, Event{To: to, Type: typ, Data: raw})
}

func (s *Service) Contacts(ctx context.Context, me auth.Identity) ([]Contact, error) {
	return s.repo.Contacts(ctx, me.UserID, IsOffice(me.Role))
}

func (s *Service) Conversations(ctx context.Context, me auth.Identity) ([]Conversation, error) {
	return s.repo.Conversations(ctx, me.UserID)
}

func (s *Service) UnreadTotal(ctx context.Context, me auth.Identity) (int, error) {
	return s.repo.UnreadTotal(ctx, me.UserID)
}

// Open returns the conversation with another user, creating it when the
// messaging rule allows.
func (s *Service) Open(ctx context.Context, me auth.Identity, otherID int64) (Conversation, error) {
	if otherID == me.UserID {
		return Conversation{}, response.NewAPIError(400, "you cannot message yourself")
	}
	if err := s.checkAllowed(ctx, me, otherID); err != nil {
		return Conversation{}, err
	}
	id, err := s.repo.OpenConversation(ctx, me.UserID, otherID)
	if err != nil {
		return Conversation{}, err
	}
	return s.repo.Conversation(ctx, id, me.UserID)
}

func (s *Service) checkAllowed(ctx context.Context, me auth.Identity, otherID int64) error {
	_, role, err := s.repo.UserRole(ctx, otherID)
	if err != nil {
		return mapErr(err)
	}
	share := false
	if !IsOffice(me.Role) && !IsOffice(auth.Role(role)) {
		if share, err = s.repo.ShareGroup(ctx, me.UserID, otherID); err != nil {
			return err
		}
	}
	if !CanMessage(me.Role, auth.Role(role), share) {
		return response.NewAPIError(403, "you can only message people in your chat group or the office")
	}
	return nil
}

func (s *Service) Messages(ctx context.Context, me auth.Identity, conversationID, after, before int64, limit int) ([]Message, error) {
	if _, err := s.repo.Thread(ctx, conversationID, me.UserID); err != nil {
		return nil, mapErr(err)
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.Messages(ctx, conversationID, after, before, limit)
}

// SendText posts a text message.
func (s *Service) SendText(ctx context.Context, me auth.Identity, conversationID int64, body string) (Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Message{}, response.NewAPIError(400, "message is empty")
	}
	if utf8.RuneCountInString(body) > maxTextRunes {
		return Message{}, response.NewAPIError(400, "message is too long")
	}
	return s.send(ctx, me, NewMessage{ConversationID: conversationID, SenderID: me.UserID, Kind: KindText, Body: body})
}

// SendFile posts a photo or voice message.
func (s *Service) SendFile(ctx context.Context, me auth.Identity, conversationID int64, kind string, r io.Reader, caption string, durationMs int) (Message, error) {
	if kind != KindVoice {
		return Message{}, response.NewAPIError(400, "only voice messages can be attached")
	}
	// Check membership and the messaging rule before writing to disk.
	if _, err := s.thread(ctx, me, conversationID); err != nil {
		return Message{}, err
	}
	name, mime, err := s.files.Save(kind, r)
	if err != nil {
		return Message{}, response.NewAPIError(422, err.Error())
	}
	if durationMs < 0 || durationMs > 5*60*1000 {
		durationMs = 0
	}
	caption = strings.TrimSpace(caption)
	if utf8.RuneCountInString(caption) > maxTextRunes {
		caption = string([]rune(caption)[:maxTextRunes])
	}
	return s.send(ctx, me, NewMessage{
		ConversationID: conversationID, SenderID: me.UserID, Kind: kind,
		Body: caption, FileName: name, MimeType: mime, DurationMs: durationMs,
	})
}

// thread loads a conversation the caller may post in: a member of the group
// for a group chat, or the messaging rule for a direct conversation (group
// membership can change after a conversation starts).
func (s *Service) thread(ctx context.Context, me auth.Identity, conversationID int64) (Thread, error) {
	t, err := s.repo.Thread(ctx, conversationID, me.UserID)
	if err != nil {
		return Thread{}, mapErr(err)
	}
	if !t.IsGroup() {
		if err := s.checkAllowed(ctx, me, t.Other); err != nil {
			return Thread{}, err
		}
	}
	return t, nil
}

func (s *Service) send(ctx context.Context, me auth.Identity, m NewMessage) (Message, error) {
	t, err := s.thread(ctx, me, m.ConversationID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.AddMessage(ctx, m)
	if err != nil {
		return Message{}, err
	}
	// Everyone in it sees it, the sender's other devices too.
	s.publish(ctx, append(t.Recipients(), me.UserID), "message", map[string]int64{
		"conversationId": msg.ConversationID, "messageId": msg.ID, "senderId": me.UserID,
	})
	title, prefix := msg.SenderName, ""
	if t.IsGroup() {
		title, prefix = t.GroupName, msg.SenderName+": "
	}
	for _, to := range t.Recipients() {
		go s.notify(to, title, prefix, msg)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, me auth.Identity, conversationID, messageID int64) error {
	t, err := s.repo.Thread(ctx, conversationID, me.UserID)
	if err != nil {
		return mapErr(err)
	}
	if err := s.repo.MarkRead(ctx, conversationID, me.UserID, messageID); err != nil {
		return err
	}
	// "Seen" for the sender of a direct message; the reader's other devices
	// clear their badge. Group chats show no "seen".
	to := []int64{me.UserID}
	if !t.IsGroup() {
		to = append(to, t.Other)
	}
	s.publish(ctx, to, "read", map[string]int64{
		"conversationId": conversationID, "messageId": messageID, "userId": me.UserID,
	})
	return nil
}

// Typing tells the others "… is typing". Nothing is stored.
func (s *Service) Typing(ctx context.Context, me auth.Identity, conversationID int64) error {
	t, err := s.repo.Thread(ctx, conversationID, me.UserID)
	if err != nil {
		return mapErr(err)
	}
	name, _, _ := s.repo.UserRole(ctx, me.UserID)
	s.publish(ctx, t.Recipients(), "typing", map[string]any{
		"conversationId": conversationID, "userId": me.UserID, "name": name,
	})
	return nil
}

// File returns the path and type of a chat file the user may see.
func (s *Service) File(ctx context.Context, me auth.Identity, name string) (string, string, error) {
	path, ct, ok := s.files.Path(name)
	if !ok {
		return "", "", response.NewAPIError(404, "file not found")
	}
	if _, err := s.repo.FileAccess(ctx, name, me.UserID); err != nil {
		return "", "", response.NewAPIError(404, "file not found")
	}
	return path, ct, nil
}

// Groups

func (s *Service) Groups(ctx context.Context) ([]Group, error) {
	return s.repo.Groups(ctx)
}

type SaveGroupRequest struct {
	Name      string  `json:"name"`
	MemberIDs []int64 `json:"memberIds"`
}

func (s *Service) SaveGroup(ctx context.Context, me auth.Identity, id int64, req SaveGroupRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return response.NewAPIError(400, "group name is required")
	}
	if utf8.RuneCountInString(req.Name) > 80 {
		return response.NewAPIError(400, "group name is too long")
	}
	_, err := s.repo.SaveGroup(ctx, id, req.Name, req.MemberIDs, me.UserID)
	return mapErr(err)
}

// DeleteGroup removes a group, its group chat and that chat's voice notes.
func (s *Service) DeleteGroup(ctx context.Context, id int64) error {
	files, err := s.repo.DeleteGroup(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	for _, f := range files {
		if err := s.files.Remove(f); err != nil {
			slog.Warn("remove group chat file", "file", f, "error", err)
		}
	}
	return nil
}

// Push

// pushSender loads (or on first use creates) the VAPID keys. A failure is
// not cached, so a database hiccup does not disable push until restart.
func (s *Service) pushSender(ctx context.Context) (*webpush.Sender, error) {
	s.pushMu.Lock()
	defer s.pushMu.Unlock()
	if s.sender != nil {
		return s.sender, nil
	}
	stored, err := s.repo.VAPIDKey(ctx, func() (string, error) {
		k, err := webpush.GenerateKeys()
		if err != nil {
			return "", err
		}
		return k.MarshalPrivate()
	})
	if err != nil {
		return nil, err
	}
	keys, err := webpush.ParseKeys(stored)
	if err != nil {
		return nil, err
	}
	s.sender = &webpush.Sender{Keys: keys, Subject: s.subject}
	return s.sender, nil
}

// PushPublicKey is the applicationServerKey browsers subscribe with.
func (s *Service) PushPublicKey(ctx context.Context) (string, error) {
	sender, err := s.pushSender(ctx)
	if err != nil {
		return "", err
	}
	return sender.Keys.PublicKey(), nil
}

type SubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (s *Service) Subscribe(ctx context.Context, me auth.Identity, req SubscribeRequest) error {
	if !strings.HasPrefix(req.Endpoint, "https://") || len(req.Endpoint) > 2000 ||
		req.Keys.P256dh == "" || req.Keys.Auth == "" || len(req.Keys.P256dh) > 200 || len(req.Keys.Auth) > 100 {
		return response.NewAPIError(400, "invalid push subscription")
	}
	return s.repo.SaveSubscription(ctx, me.UserID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth)
}

func (s *Service) Unsubscribe(ctx context.Context, me auth.Identity, endpoint string) error {
	return s.repo.DeleteSubscription(ctx, me.UserID, endpoint)
}

// notify pushes a new-message alert to every device of the recipient. It
// runs detached from the request, best-effort. title is the sender (or the
// group); prefix goes before the preview ("Noi: " in a group).
func (s *Service) notify(recipient int64, title, prefix string, msg Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	subs, err := s.repo.Subscriptions(ctx, recipient)
	if err != nil || len(subs) == 0 {
		return
	}
	sender, err := s.pushSender(ctx)
	if err != nil {
		slog.Warn("chat push disabled", "error", err)
		return
	}
	preview := msg.Body
	switch msg.Kind {
	case KindImage:
		preview = "📷 " + preview
	case KindVoice:
		preview = "🎤 Voice message"
	}
	preview = prefix + preview
	if r := []rune(preview); len(r) > 120 {
		preview = string(r[:120]) + "…"
	}
	payload, _ := json.Marshal(map[string]string{
		"title": title,
		"body":  strings.TrimSpace(preview),
		"url":   fmt.Sprintf("/chat?c=%d", msg.ConversationID),
		"tag":   fmt.Sprintf("chat-%d", msg.ConversationID),
	})
	for _, sub := range subs {
		err := sender.Send(ctx, webpush.Subscription{Endpoint: sub.Endpoint, P256dh: sub.P256dh, Auth: sub.Auth}, payload, 24*time.Hour)
		if errors.Is(err, webpush.ErrGone) {
			s.repo.DropSubscription(ctx, sub.ID)
		} else if err != nil {
			slog.Warn("chat push failed", "subscription", sub.ID, "error", err)
		}
	}
}

func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
		return response.NewAPIError(404, "not found")
	case errors.Is(err, ErrUserMissing):
		return response.NewAPIError(404, "user not found")
	case errors.Is(err, ErrNotAllowed):
		return response.NewAPIError(403, err.Error())
	}
	return err
}
