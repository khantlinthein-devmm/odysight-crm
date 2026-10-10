package chat

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBrokerDeliversOnlyToRecipients(t *testing.T) {
	b := NewBroker(nil)
	a, cancelA := b.Subscribe(1)
	defer cancelA()
	other, cancelOther := b.Subscribe(2)
	defer cancelOther()

	b.deliver(Event{To: []int64{1}, Type: "message", Data: []byte(`{"conversationId":7}`)})
	select {
	case ev := <-a:
		if ev.Type != "message" {
			t.Fatalf("got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("recipient got nothing")
	}
	select {
	case ev := <-other:
		t.Fatalf("non-recipient got %v", ev)
	default:
	}

	cancelA()
	b.deliver(Event{To: []int64{1}, Type: "message"}) // no subscriber: must not block or panic
}

// TestDBBrokerNotifyRoundTrip publishes through PostgreSQL NOTIFY and
// receives via LISTEN, as separate API instances would.
func TestDBBrokerNotifyRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	listener, sender := NewBroker(pool), NewBroker(pool)
	go listener.Run(ctx)
	ch, unsub := listener.Subscribe(42)
	defer unsub()
	time.Sleep(300 * time.Millisecond) // let LISTEN start

	sender.Publish(ctx, Event{To: []int64{42}, Type: "typing", Data: []byte(`{"conversationId":1}`)})
	select {
	case ev := <-ch:
		if ev.Type != "typing" {
			t.Fatalf("got %q", ev.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("event did not cross NOTIFY/LISTEN")
	}

	cancel()
	select {
	case <-listener.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("Done not closed after shutdown")
	}
}
