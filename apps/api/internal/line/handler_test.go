package line

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func signForTest(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func testHandler() (*Handler, *fakeStore, *fakeReplier) {
	store := newFakeStore()
	profiles := &fakeProfiles{names: map[string]Profile{
		"U999": {UserID: "U999", DisplayName: "Webhook Nok"},
	}}
	replier := &fakeReplier{}
	svc := NewService(store, profiles, replier, "")
	return NewHandler(svc, "s3cr3t"), store, replier
}

func postWebhook(t *testing.T, h *Handler, body []byte, signature string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/line/webhook", bytes.NewReader(body))
	if signature != "" {
		req.Header.Set("X-Line-Signature", signature)
	}
	rec := httptest.NewRecorder()
	h.Webhook(rec, req)
	return rec
}

func TestWebhookCreatesLeadOnFollow(t *testing.T) {
	h, store, _ := testHandler()
	body := []byte(`{"destination":"Uxxx","events":[{"type":"follow","replyToken":"rt","source":{"type":"user","userId":"U999"}}]}`)
	rec := postWebhook(t, h, body, signForTest("s3cr3t", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", rec.Code, rec.Body.String())
	}
	if _, err := store.FindByLineUserID(context.Background(), "U999"); err != nil {
		t.Fatalf("lead not created: %v", err)
	}
}

func TestWebhookRejectsBadSignature(t *testing.T) {
	h, store, _ := testHandler()
	body := []byte(`{"destination":"Uxxx","events":[{"type":"follow","source":{"type":"user","userId":"U999"}}]}`)
	rec := postWebhook(t, h, body, "bogus")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
	if len(store.byLine) != 0 {
		t.Fatal("unsigned delivery must not create a lead")
	}
}

func TestWebhookRejectsMissingSignature(t *testing.T) {
	h, _, _ := testHandler()
	body := []byte(`{"events":[]}`)
	rec := postWebhook(t, h, body, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestWebhookDisabledWithoutSecret(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, &fakeProfiles{}, &fakeReplier{}, "")
	h := NewHandler(svc, "")
	body := []byte(`{"events":[]}`)
	rec := postWebhook(t, h, body, signForTest("anything", body))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", rec.Code)
	}
}

func TestWebhookRejectsInvalidJSON(t *testing.T) {
	h, _, _ := testHandler()
	body := []byte(`{not json`)
	rec := postWebhook(t, h, body, signForTest("s3cr3t", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}

func TestClientProfileAndReply(t *testing.T) {
	var gotAuth, gotReplyAuth, gotReplyBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/bot/profile/U123":
			gotAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"userId": "U123", "displayName": "Stub Malee", "pictureUrl": "http://x/p.jpg",
			})
		case "/v2/bot/message/reply":
			gotReplyAuth = r.Header.Get("Authorization")
			var payload struct {
				ReplyToken string `json:"replyToken"`
				Messages   []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if len(payload.Messages) == 1 {
				gotReplyBody = payload.Messages[0].Text
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClientWithBase("tok123", srv.URL)
	p, err := c.GetProfile(context.Background(), "U123")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if p.DisplayName != "Stub Malee" || p.PictureURL != "http://x/p.jpg" {
		t.Fatalf("profile = %+v", p)
	}
	if gotAuth != "Bearer tok123" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if err := c.Reply(context.Background(), "rt9", "hi"); err != nil {
		t.Fatalf("reply: %v", err)
	}
	if gotReplyAuth != "Bearer tok123" || gotReplyBody != "hi" {
		t.Fatalf("reply auth=%q body=%q", gotReplyAuth, gotReplyBody)
	}
}
