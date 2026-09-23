package line

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/odysight/crm/internal/leads"
)

// LeadStore is the slice of lead persistence the LINE flow needs. The live
// implementation is *leads.Repository; tests use a fake.
type LeadStore interface {
	FindByLineUserID(ctx context.Context, lineUserID string) (leads.Lead, error)
	CreateLineLead(ctx context.Context, l leads.Lead) (leads.Lead, error)
	RefreshLineIdentity(ctx context.Context, id int64, firstName, lastName, pictureURL string) (leads.Lead, error)
}

// Event is a normalized LINE webhook event (follow or message from a user).
type Event struct {
	Type       string
	ReplyToken string
	UserID     string
}

type Service struct {
	store     LeadStore
	profiles  ProfileFetcher
	replier   Replier
	autoReply string
}

func NewService(store LeadStore, profiles ProfileFetcher, replier Replier, autoReply string) *Service {
	return &Service{store: store, profiles: profiles, replier: replier, autoReply: strings.TrimSpace(autoReply)}
}

// VerifySignature validates the X-Line-Signature header: Base64(HMAC-SHA256
// (rawBody, channelSecret)). Every webhook delivery must pass this check;
// unsigned requests are rejected by the handler before any processing.
func VerifySignature(channelSecret string, body []byte, signature string) bool {
	if channelSecret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// validUserID rejects group/room IDs and malformed values. LINE user IDs
// start with "U".
func validUserID(id string) bool {
	return len(id) >= 2 && len(id) <= 64 && strings.HasPrefix(id, "U")
}

// splitName keeps the full display name visible: first token becomes the
// first name (required by the leads schema), the rest becomes last name.
func splitName(display string) (first, last string) {
	parts := strings.Fields(strings.TrimSpace(display))
	if len(parts) == 0 {
		return "", ""
	}
	first = parts[0]
	if len(first) > 100 {
		first = first[:100]
	}
	if len(parts) > 1 {
		last = strings.Join(parts[1:], " ")
		if len(last) > 100 {
			last = last[:100]
		}
	}
	return first, last
}

// HandleEvents processes follow/message events: fetch the LINE profile,
// create the lead once (idempotent), refresh the stored identity when the
// profile changed, and send the optional auto-reply. Unknown users or
// profile failures fail that event only; other events still process.
func (s *Service) HandleEvents(ctx context.Context, events []Event) (created int, err error) {
	for _, e := range events {
		if e.Type != "follow" && e.Type != "message" {
			continue
		}
		if !validUserID(e.UserID) {
			continue
		}
		profile, perr := s.profiles.GetProfile(ctx, e.UserID)
		if perr != nil {
			err = perr
			continue
		}
		first, last := splitName(profile.DisplayName)
		if first == "" {
			err = fmt.Errorf("line profile %s has no display name", e.UserID)
			continue
		}
		existing, ferr := s.store.FindByLineUserID(ctx, e.UserID)
		if ferr != nil && !errors.Is(ferr, leads.ErrNotFound) {
			err = ferr
			continue
		}
		if ferr == nil {
			// Known user: refresh the name/avatar when they changed, then
			// reply. Never creates a second lead.
			if existing.FirstName != first || existing.LastName != last ||
				existing.LinePictureURL != profile.PictureURL {
				if _, rerr := s.store.RefreshLineIdentity(ctx, existing.ID, first, last, profile.PictureURL); rerr != nil {
					err = rerr
					continue
				}
			}
		} else {
			if _, cerr := s.store.CreateLineLead(ctx, leads.Lead{
				FirstName:      first,
				LastName:       last,
				Phone:          "", // LINE shares no phone; staff collect it on first contact.
				Status:         leads.StatusNew,
				Source:         leads.SourceLine,
				LineUserID:     e.UserID,
				LinePictureURL: profile.PictureURL,
			}); cerr != nil {
				err = cerr
				continue
			}
			created++
		}
		if s.autoReply != "" && e.ReplyToken != "" && s.replier != nil {
			if rerr := s.replier.Reply(ctx, e.ReplyToken, s.autoReply); rerr != nil {
				err = rerr
			}
		}
	}
	return created, err
}
