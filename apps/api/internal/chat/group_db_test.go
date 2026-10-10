package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/odysight/crm/internal/auth"
)

// TestDBGroupChat runs against a migrated database and skips unless
// TEST_DATABASE_URL is set.
func TestDBGroupChat(t *testing.T) {
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
	svc := NewService(repo, files, "", nil)

	stamp := time.Now().UnixNano()
	ids := make([]int64, 4) // a, b, c are members; outsider is not
	for i := range ids {
		if err := pool.QueryRow(ctx,
			`INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, 'x', 'CLEANER') RETURNING id`,
			fmt.Sprintf("Group %d", i), fmt.Sprintf("group-%d-%d@test.local", stamp, i)).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	a, b, c, outsider := ids[0], ids[1], ids[2], ids[3]
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, ids) })
	me := func(id int64) auth.Identity { return auth.Identity{UserID: id, Role: auth.RoleCleaner} }

	group, err := repo.SaveGroup(ctx, 0, fmt.Sprintf("Team %d", stamp), []int64{a, b}, a)
	if err != nil {
		t.Fatal(err)
	}
	find := func(user int64) *Conversation {
		list, err := repo.Conversations(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		for i := range list {
			if list[i].Group != nil && list[i].Group.ID == group {
				return &list[i]
			}
		}
		return nil
	}
	conv := find(a)
	if conv == nil || conv.Group.Members != 2 {
		t.Fatalf("new group chat not listed for a member: %+v", conv)
	}
	if find(outsider) != nil {
		t.Fatal("group chat listed for a non-member")
	}

	if _, err := svc.SendText(ctx, me(a), conv.ID, "hello team"); err != nil {
		t.Fatal(err)
	}
	if got := find(b); got == nil || got.Unread != 1 || got.LastSender != "Group 0" {
		t.Fatalf("b should have 1 unread from Group 0: %+v", got)
	}
	if got := find(a); got.Unread != 0 {
		t.Fatalf("sender has %d unread", got.Unread)
	}
	if _, err := svc.SendText(ctx, me(outsider), conv.ID, "let me in"); err == nil {
		t.Fatal("non-member posted to the group")
	}
	if _, err := svc.Messages(ctx, me(outsider), conv.ID, 0, 0, 10); err == nil {
		t.Fatal("non-member read the group")
	}

	// c joins later: history counts as read.
	if _, err := repo.SaveGroup(ctx, group, "Team", []int64{a, b, c}, a); err != nil {
		t.Fatal(err)
	}
	if got := find(c); got == nil || got.Unread != 0 || got.Group.Members != 3 {
		t.Fatalf("late joiner: %+v", got)
	}
	msgs, err := svc.Messages(ctx, me(c), conv.ID, 0, 0, 10)
	if err != nil || len(msgs) != 1 || msgs[0].SenderName != "Group 0" {
		t.Fatalf("late joiner reads history: %v %+v", err, msgs)
	}
	if err := svc.MarkRead(ctx, me(b), conv.ID, msgs[0].ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := repo.UnreadTotal(ctx, b); n != 0 {
		t.Fatalf("b unread after reading: %d", n)
	}

	webm := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0}, 100)...)
	voice, err := svc.SendFile(ctx, me(c), conv.ID, KindVoice, bytes.NewReader(webm), "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(voice.FileURL)
	if _, _, err := svc.File(ctx, me(b), name); err != nil {
		t.Fatalf("member cannot play group voice note: %v", err)
	}
	if _, _, err := svc.File(ctx, me(outsider), name); err == nil {
		t.Fatal("non-member played group voice note")
	}

	// b leaves: no longer in the chat.
	if _, err := repo.SaveGroup(ctx, group, "Team", []int64{a, c}, a); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendText(ctx, me(b), conv.ID, "still here?"); err == nil {
		t.Fatal("removed member posted")
	}

	if err := svc.DeleteGroup(ctx, group); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Thread(ctx, conv.ID, a); !errors.Is(err, ErrNotFound) {
		t.Fatalf("group chat survived its group: %v", err)
	}
	if _, err := os.Stat(filepath.Join(files.dir, name)); !os.IsNotExist(err) {
		t.Fatal("group voice note left on disk")
	}
}
