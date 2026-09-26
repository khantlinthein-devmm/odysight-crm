package line

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LINE Messaging API endpoints. Base URLs are fields (not constants) so tests
// can point the client at a local stub server.
type Profile struct {
	UserID      string
	DisplayName string
	PictureURL  string
}

type ProfileFetcher interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
}

type Replier interface {
	Reply(ctx context.Context, replyToken, text string) error
}

type Client struct {
	http       *http.Client
	apiBase    string
	token      string
	maxPicture int
}

func NewClient(channelAccessToken string) *Client {
	return NewClientWithBase(channelAccessToken, "https://api.line.me")
}

func NewClientWithBase(channelAccessToken, apiBase string) *Client {
	return &Client{
		http:    &http.Client{Timeout: 10 * time.Second},
		apiBase: apiBase,
		token:   channelAccessToken,
	}
}

// GetProfile fetches the LINE display name (and avatar) for a user ID.
// LINE never shares phone numbers: the lead starts with phone ” and staff
// collect it on first contact.
func (c *Client) GetProfile(ctx context.Context, userID string) (Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.apiBase+"/v2/bot/profile/"+userID, nil)
	if err != nil {
		return Profile{}, fmt.Errorf("line profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	res, err := c.http.Do(req)
	if err != nil {
		return Profile{}, fmt.Errorf("line profile call: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if res.StatusCode != http.StatusOK {
		return Profile{}, fmt.Errorf("line profile status %d", res.StatusCode)
	}
	var out struct {
		UserID      string `json:"userId"`
		DisplayName string `json:"displayName"`
		PictureURL  string `json:"pictureUrl"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return Profile{}, fmt.Errorf("line profile decode: %w", err)
	}
	return Profile{UserID: out.UserID, DisplayName: out.DisplayName, PictureURL: out.PictureURL}, nil
}

// Reply sends a text message using the event's one-time reply token.
func (c *Client) Reply(ctx context.Context, replyToken, text string) error {
	payload, err := json.Marshal(map[string]any{
		"replyToken": replyToken,
		"messages":   []any{map[string]any{"type": "text", "text": text}},
	})
	if err != nil {
		return fmt.Errorf("line reply encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.apiBase+"/v2/bot/message/reply", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("line reply request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("line reply call: %w", err)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 16<<10))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("line reply status %d", res.StatusCode)
	}
	return nil
}

// Push sends a text message to a user outside a reply window. It counts
// against the channel's monthly message quota.
func (c *Client) Push(ctx context.Context, to, text string) error {
	if c.token == "" {
		return fmt.Errorf("line push: channel access token not configured")
	}
	payload, err := json.Marshal(map[string]any{
		"to":       to,
		"messages": []any{map[string]any{"type": "text", "text": text}},
	})
	if err != nil {
		return fmt.Errorf("line push encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.apiBase+"/v2/bot/message/push", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("line push request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("line push call: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 16<<10))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("line push status %d: %s", res.StatusCode, bytes.TrimSpace(body))
	}
	return nil
}
