package line

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/odysight/crm/internal/leads"
)

type fakeStore struct {
	byLine map[string]leads.Lead
	nextID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{byLine: map[string]leads.Lead{}, nextID: 100}
}

func (f *fakeStore) FindByLineUserID(_ context.Context, id string) (leads.Lead, error) {
	l, ok := f.byLine[id]
	if !ok {
		return leads.Lead{}, leads.ErrNotFound
	}
	return l, nil
}

func (f *fakeStore) CreateLineLead(_ context.Context, l leads.Lead) (leads.Lead, error) {
	if _, ok := f.byLine[l.LineUserID]; ok {
		return f.byLine[l.LineUserID], nil
	}
	f.nextID++
	l.ID = f.nextID
	f.byLine[l.LineUserID] = l
	return l, nil
}

func (f *fakeStore) RefreshLineIdentity(_ context.Context, id int64, first, last, picture string) (leads.Lead, error) {
	for k, l := range f.byLine {
		if l.ID == id {
			l.FirstName, l.LastName, l.LinePictureURL = first, last, picture
			f.byLine[k] = l
			return l, nil
		}
	}
	return leads.Lead{}, leads.ErrNotFound
}

type fakeProfiles struct {
	names map[string]Profile
	err   error
}

func (f *fakeProfiles) GetProfile(_ context.Context, userID string) (Profile, error) {
	if f.err != nil {
		return Profile{}, f.err
	}
	if p, ok := f.names[userID]; ok {
		return p, nil
	}
	return Profile{UserID: userID, DisplayName: "Test User"}, nil
}

type fakeReplier struct {
	sent []string
	err  error
}

func (f *fakeReplier) Reply(_ context.Context, replyToken, text string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, replyToken+":"+text)
	return nil
}

func hmacForTest(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	secret := "test-channel-secret"
	body := []byte(`{"destination":"U123","events":[]}`)
	mac := hmacForTest(secret, body)
	if !VerifySignature(secret, body, mac) {
		t.Fatal("valid signature rejected")
	}
	if VerifySignature(secret, body, mac+"x") {
		t.Fatal("tampered signature accepted")
	}
	if VerifySignature(secret, append(body, ' '), mac) {
		t.Fatal("tampered body accepted")
	}
	if VerifySignature("other-secret", body, mac) {
		t.Fatal("wrong secret accepted")
	}
	if VerifySignature("", body, mac) {
		t.Fatal("empty secret accepted")
	}
	if VerifySignature(secret, body, "") {
		t.Fatal("empty signature accepted")
	}
}

func TestFollowCreatesLineLead(t *testing.T) {
	store := newFakeStore()
	profiles := &fakeProfiles{names: map[string]Profile{
		"U111": {UserID: "U111", DisplayName: "Somsak J"},
	}}
	replier := &fakeReplier{}
	svc := NewService(store, profiles, replier, "hello")

	created, err := svc.HandleEvents(context.Background(), []Event{
		{Type: "follow", ReplyToken: "rt1", UserID: "U111"},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}
	lead, err := store.FindByLineUserID(context.Background(), "U111")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if lead.FirstName != "Somsak" || lead.LastName != "J" {
		t.Fatalf("name split = %q %q", lead.FirstName, lead.LastName)
	}
	if lead.Source != leads.SourceLine || lead.Status != leads.StatusNew {
		t.Fatalf("source/status = %q/%q", lead.Source, lead.Status)
	}
	if lead.Phone != "" {
		t.Fatalf("LINE lead must start with empty phone, got %q", lead.Phone)
	}
	if len(replier.sent) != 1 {
		t.Fatalf("auto-reply not sent: %v", replier.sent)
	}
}

func TestDuplicateFollowNeverDuplicatesLead(t *testing.T) {
	store := newFakeStore()
	profiles := &fakeProfiles{names: map[string]Profile{
		"U222": {UserID: "U222", DisplayName: "Mali"},
	}}
	svc := NewService(store, profiles, &fakeReplier{}, "")

	for i := 0; i < 3; i++ {
		created, err := svc.HandleEvents(context.Background(), []Event{
			{Type: "follow", UserID: "U222"},
			{Type: "message", UserID: "U222"},
		})
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if i == 0 && created != 1 {
			t.Fatalf("first delivery created = %d, want 1", created)
		}
		if i > 0 && created != 0 {
			t.Fatalf("retry %d created = %d, want 0", i, created)
		}
	}
	if len(store.byLine) != 1 {
		t.Fatalf("leads = %d, want 1", len(store.byLine))
	}
}

func TestChangedProfileRefreshesIdentity(t *testing.T) {
	store := newFakeStore()
	profiles := &fakeProfiles{names: map[string]Profile{
		"U333": {UserID: "U333", DisplayName: "Old Name", PictureURL: "http://x/old.jpg"},
	}}
	svc := NewService(store, profiles, &fakeReplier{}, "")

	if _, err := svc.HandleEvents(context.Background(), []Event{{Type: "follow", UserID: "U333"}}); err != nil {
		t.Fatal(err)
	}
	profiles.names["U333"] = Profile{UserID: "U333", DisplayName: "New Name", PictureURL: "http://x/new.jpg"}
	if _, err := svc.HandleEvents(context.Background(), []Event{{Type: "message", UserID: "U333"}}); err != nil {
		t.Fatal(err)
	}
	lead, _ := store.FindByLineUserID(context.Background(), "U333")
	if lead.FirstName != "New" || lead.LinePictureURL != "http://x/new.jpg" {
		t.Fatalf("identity not refreshed: %+v", lead)
	}
	if len(store.byLine) != 1 {
		t.Fatal("refresh must not create a second lead")
	}
}

func TestNonUserAndUnknownEventsSkipped(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, &fakeProfiles{}, &fakeReplier{}, "")
	created, err := svc.HandleEvents(context.Background(), []Event{
		{Type: "unfollow", UserID: "U444"},
		{Type: "join", UserID: "U444"},
		{Type: "message", UserID: "Cgroup1"}, // group ID, not a user
		{Type: "message", UserID: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created != 0 || len(store.byLine) != 0 {
		t.Fatal("non-user events must not create leads")
	}
}

func TestProfileFailureFailsEventOnly(t *testing.T) {
	store := newFakeStore()
	profiles := &fakeProfiles{err: errors.New("line down")}
	svc := NewService(store, profiles, &fakeReplier{}, "")
	_, err := svc.HandleEvents(context.Background(), []Event{{Type: "follow", UserID: "U555"}})
	if err == nil {
		t.Fatal("expected profile error")
	}
	if len(store.byLine) != 0 {
		t.Fatal("failed profile must not create a lead")
	}
}

func TestSplitName(t *testing.T) {
	first, last := splitName("Somsak Jaidee")
	if first != "Somsak" || last != "Jaidee" {
		t.Fatalf("got %q %q", first, last)
	}
	first, last = splitName("Mali")
	if first != "Mali" || last != "" {
		t.Fatalf("got %q %q", first, last)
	}
	first, _ = splitName("   ")
	if first != "" {
		t.Fatalf("blank display name must yield empty first, got %q", first)
	}
}
