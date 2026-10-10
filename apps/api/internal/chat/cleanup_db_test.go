package chat

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestDBCleanupRemovesOldVoiceNotes runs against a migrated database and
// skips unless TEST_DATABASE_URL is set.
func TestDBCleanupRemovesOldVoiceNotes(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := NewRepository(pool)
	files := NewFileStore(t.TempDir())

	stamp := time.Now().UnixNano()
	var a, b int64
	for i, id := range []*int64{&a, &b} {
		if err := pool.QueryRow(ctx,
			`INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, 'x', 'CLEANER') RETURNING id`,
			fmt.Sprintf("Cleanup %d", i), fmt.Sprintf("cleanup-%d-%d@test.local", stamp, i)).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, a, b) })
	conv, err := repo.OpenConversation(ctx, a, b)
	if err != nil {
		t.Fatal(err)
	}

	webm := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0}, 100)...)
	store := func() string {
		name, _, err := files.Save(KindVoice, bytes.NewReader(webm))
		if err != nil {
			t.Fatal(err)
		}
		return name
	}
	oldName, newName := store(), store()
	old, err := repo.AddMessage(ctx, NewMessage{ConversationID: conv, SenderID: a, Kind: KindVoice, FileName: oldName, MimeType: "audio/webm"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddMessage(ctx, NewMessage{ConversationID: conv, SenderID: a, Kind: KindVoice, FileName: newName, MimeType: "audio/webm"}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE chat_messages SET created_at = now() - interval '31 days' WHERE id = $1`, old.ID); err != nil {
		t.Fatal(err)
	}

	if _, _, err := files.Save(KindImage, bytes.NewReader([]byte("\x89PNG\r\n\x1a\n"))); err == nil {
		t.Fatal("photos must be refused")
	}

	c := NewCleaner(repo, files)
	if n := c.Sweep(ctx); n < 1 {
		t.Fatalf("swept %d files, want at least the old one", n)
	}
	if _, err := os.Stat(filepath.Join(files.dir, oldName)); !os.IsNotExist(err) {
		t.Fatal("old voice note still on disk")
	}
	if _, err := os.Stat(filepath.Join(files.dir, newName)); err != nil {
		t.Fatal("recent voice note was deleted")
	}
	msgs, err := repo.Messages(ctx, conv, 0, 0, 10)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("messages: %v %d", err, len(msgs))
	}
	if !msgs[0].Expired || msgs[0].FileURL != "" || msgs[1].Expired || msgs[1].FileURL == "" {
		t.Fatalf("expired flags wrong: %+v", msgs)
	}
}
