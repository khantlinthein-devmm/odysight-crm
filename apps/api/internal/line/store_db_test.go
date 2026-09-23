package line

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/internal/leads"
)

// Exercises the LINE lead upsert against a real database, because the
// retry-safety lives in the partial unique index + ON CONFLICT boundary.
// Skips unless TEST_DATABASE_URL is set:
//
//	TEST_DATABASE_URL=postgres://odysight:secret@localhost:5433/odysight_crm go test ./internal/line/ -run DB -v
func testRepo(t *testing.T) (*leads.Repository, func()) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)
	repo := leads.NewRepository(pool)
	uid := "Utestdb000000000000000000000001"
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM leads WHERE line_user_id = $1`, uid)
	})
	_, _ = pool.Exec(ctx, `DELETE FROM leads WHERE line_user_id = $1`, uid)
	return repo, func() {}
}

func TestDBCreateLineLeadIdempotent(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	uid := "Utestdb000000000000000000000001"

	first, err := repo.CreateLineLead(ctx, leads.Lead{
		FirstName: "DB", LastName: "Test", Status: leads.StatusNew,
		Source: leads.SourceLine, LineUserID: uid,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Simulated LINE retry: same user again must return the same row.
	second, err := repo.CreateLineLead(ctx, leads.Lead{
		FirstName: "DB", LastName: "Test", Status: leads.StatusNew,
		Source: leads.SourceLine, LineUserID: uid,
	})
	if err != nil {
		t.Fatalf("retry create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("retry created a second lead: %d vs %d", first.ID, second.ID)
	}

	found, err := repo.FindByLineUserID(ctx, uid)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.ID != first.ID {
		t.Fatalf("find returned %d, want %d", found.ID, first.ID)
	}

	updated, err := repo.RefreshLineIdentity(ctx, first.ID, "DB2", "Test2", "http://x/p.jpg")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if updated.FirstName != "DB2" || updated.LinePictureURL != "http://x/p.jpg" {
		t.Fatalf("refresh = %+v", updated)
	}
}
