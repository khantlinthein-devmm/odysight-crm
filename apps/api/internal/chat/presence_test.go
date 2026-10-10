package chat

import (
	"testing"
	"time"
)

func TestPresence(t *testing.T) {
	old := presenceGrace
	presenceGrace = 50 * time.Millisecond
	t.Cleanup(func() { presenceGrace = old })

	b := NewBroker(nil)
	changes := make(chan bool, 10)
	b.OnPresence(func(id int64, online bool) {
		if id == 7 {
			changes <- online
		}
	})
	expect := func(want bool) {
		t.Helper()
		select {
		case got := <-changes:
			if got != want {
				t.Fatalf("presence change: got %v, want %v", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("no presence change (want %v)", want)
		}
	}
	quiet := func() {
		t.Helper()
		select {
		case got := <-changes:
			t.Fatalf("unexpected presence change %v", got)
		case <-time.After(120 * time.Millisecond):
		}
	}

	if b.Online(7) {
		t.Fatal("online before connecting")
	}
	_, closeA := b.Subscribe(7)
	expect(true)
	_, closeB := b.Subscribe(7) // a second tab: no new announcement
	quiet()
	closeA()
	if !b.Online(7) {
		t.Fatal("offline while another tab is open")
	}
	closeB()
	if !b.Online(7) {
		t.Fatal("offline immediately after closing (should wait out the grace period)")
	}
	// Reconnecting within the grace period (a phone switching networks)
	// announces nothing either way.
	_, closeC := b.Subscribe(7)
	quiet()
	closeC()
	expect(false)
	if b.Online(7) {
		t.Fatal("still online after the grace period")
	}
}
