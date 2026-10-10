package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Real-time delivery. Events go through PostgreSQL NOTIFY on one channel and
// every API instance LISTENs, then fans them out to its own connected
// browsers (Server-Sent Events). One instance today, several tomorrow, no
// extra infrastructure either way.
//
// Events are small hints ("conversation 12 has message 345"); browsers fetch
// the content through the normal, permission-checked endpoints.

const notifyChannel = "chat_events"

// Event is delivered to the users listed in To.
type Event struct {
	To   []int64         `json:"to"`
	Type string          `json:"type"` // message | read | typing | presence
	Data json.RawMessage `json:"data"`
}

type Broker struct {
	pool *pgxpool.Pool

	mu   sync.RWMutex
	subs map[int64]map[chan Event]struct{}

	// done closes when the server shuts down, ending every open stream so
	// browsers reconnect to the new process instead of hanging on this one
	// (graceful shutdown would otherwise wait on streams that never end).
	done     chan struct{}
	stopOnce sync.Once

	// Presence: someone is online while they have the app open (an event
	// stream is connected) and for presenceGrace after it closes, so phones
	// hopping networks do not flicker offline. Tracked per API instance.
	lastSeen map[int64]time.Time
	onChange func(userID int64, online bool)
}

// presenceGrace is a var so tests can shorten it.
var presenceGrace = 20 * time.Second

// OnPresence sets the callback run (in its own goroutine) when a user comes
// online or goes offline. Set it before serving requests.
func (b *Broker) OnPresence(fn func(userID int64, online bool)) { b.onChange = fn }

// Online reports whether the user has the app open.
func (b *Broker) Online(userID int64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.onlineLocked(userID)
}

func (b *Broker) onlineLocked(userID int64) bool {
	if len(b.subs[userID]) > 0 {
		return true
	}
	seen, ok := b.lastSeen[userID]
	return ok && time.Since(seen) < presenceGrace
}

func (b *Broker) changed(userID int64, online bool) {
	if b.onChange != nil {
		go b.onChange(userID, online)
	}
}

func NewBroker(pool *pgxpool.Pool) *Broker {
	return &Broker{
		pool: pool, subs: map[int64]map[chan Event]struct{}{}, done: make(chan struct{}),
		lastSeen: map[int64]time.Time{},
	}
}

// Done is closed once the broker stops.
func (b *Broker) Done() <-chan struct{} { return b.done }

// Publish sends an event to its recipients on every instance.
func (b *Broker) Publish(ctx context.Context, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	if _, err := b.pool.Exec(ctx, `SELECT pg_notify($1, $2)`, notifyChannel, string(payload)); err != nil {
		// Fall back to local delivery so a single instance still works.
		slog.Warn("chat notify failed; delivering locally", "error", err)
		b.deliver(ev)
	}
}

// Subscribe registers a browser connection for a user. The returned cancel
// must be called when the connection closes.
func (b *Broker) Subscribe(userID int64) (<-chan Event, func()) {
	ch := make(chan Event, 32)
	b.mu.Lock()
	cameOnline := !b.onlineLocked(userID)
	if b.subs[userID] == nil {
		b.subs[userID] = map[chan Event]struct{}{}
	}
	b.subs[userID][ch] = struct{}{}
	delete(b.lastSeen, userID)
	b.mu.Unlock()
	if cameOnline {
		b.changed(userID, true)
	}
	return ch, func() {
		b.mu.Lock()
		delete(b.subs[userID], ch)
		last := len(b.subs[userID]) == 0
		if last {
			delete(b.subs, userID)
			closedAt := time.Now()
			b.lastSeen[userID] = closedAt
			// Offline only if nothing reconnected during the grace period.
			time.AfterFunc(presenceGrace, func() {
				b.mu.Lock()
				gone := len(b.subs[userID]) == 0 && b.lastSeen[userID].Equal(closedAt)
				if gone {
					delete(b.lastSeen, userID)
				}
				b.mu.Unlock()
				if gone {
					b.changed(userID, false)
				}
			})
		}
		b.mu.Unlock()
	}
}

func (b *Broker) deliver(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, uid := range ev.To {
		for ch := range b.subs[uid] {
			select {
			case ch <- ev:
			default: // a stalled connection drops hints; it re-syncs on reconnect
			}
		}
	}
}

// Run LISTENs for events until ctx ends, reconnecting after failures.
func (b *Broker) Run(ctx context.Context) {
	defer b.stopOnce.Do(func() { close(b.done) })
	backoff := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		err := b.listen(ctx)
		if time.Since(started) > time.Minute {
			backoff = time.Second // it was healthy; retry quickly
		}
		if ctx.Err() != nil {
			return
		}
		slog.Warn("chat event listener stopped; retrying", "error", err, "in", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (b *Broker) listen(ctx context.Context) error {
	pooled, err := b.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	// The connection carries LISTEN state: take it out of the pool for good
	// and close it when listening stops.
	conn := pooled.Hijack()
	defer conn.Close(context.Background())
	if _, err := conn.Exec(ctx, "LISTEN "+notifyChannel); err != nil {
		return err
	}
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var ev Event
		if json.Unmarshal([]byte(n.Payload), &ev) == nil {
			b.deliver(ev)
		}
	}
}
