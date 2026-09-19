package bookings

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

// These exercise the recurring generator against a real database, because the
// duplication bug they cover lives in the transaction boundary rather than in
// any pure function. CI has no Postgres service, so they skip unless
// TEST_DATABASE_URL is set:
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/bookings/ -run DB
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return pool
}

// seedRecurring creates one overdue recurring booking and removes the whole
// series afterwards.
func seedRecurring(t *testing.T, repo *Repository, series string) Booking {
	t.Helper()
	ctx := context.Background()
	src, err := repo.Create(ctx, Booking{
		CustomerName:    "Recurring Test Co",
		ServiceType:     "office",
		ScheduledFor:    time.Now().Add(-72 * time.Hour),
		DurationMinutes: 180,
		Address:         "1 Test Road",
		Status:          StatusConfirmed,
		IsRecurring:     true,
		Recurrence:      RecurWeekly,
		SeriesID:        &series,
	})
	if err != nil {
		t.Fatalf("seed recurring booking: %v", err)
	}
	t.Cleanup(func() {
		_, _ = repo.pool.Exec(context.Background(),
			`DELETE FROM bookings WHERE series_id = $1`, series)
	})
	return src
}

func countSeries(t *testing.T, repo *Repository, series string) int {
	t.Helper()
	var n int
	if err := repo.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM bookings WHERE series_id = $1`, series).Scan(&n); err != nil {
		t.Fatalf("count series: %v", err)
	}
	return n
}

func isRecurring(t *testing.T, repo *Repository, id int64) bool {
	t.Helper()
	var flag bool
	if err := repo.pool.QueryRow(context.Background(),
		`SELECT is_recurring FROM bookings WHERE id = $1`, id).Scan(&flag); err != nil {
		t.Fatalf("read is_recurring: %v", err)
	}
	return flag
}

// A second generator pass must not create a second successor.
func TestDBRecurrenceGeneratesOnceOnly(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	series := "test-once-" + time.Now().Format("150405.000000")
	src := seedRecurring(t, repo, series)

	runner := NewRecurrenceRunner(repo, time.Hour)
	ctx := context.Background()
	if err := runner.generate(ctx); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	if got := countSeries(t, repo, series); got != 2 {
		t.Fatalf("after first run: %d bookings in series, want 2", got)
	}
	if isRecurring(t, repo, src.ID) {
		t.Fatal("source still marked recurring after roll-forward")
	}

	if err := runner.generate(ctx); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if got := countSeries(t, repo, series); got != 2 {
		t.Fatalf("after second run: %d bookings in series, want 2", got)
	}
}

// The original bug: the successor was written but retiring the source failed,
// so every later run produced another copy of the same occurrence. Re-arming
// the source reproduces that state; the generator must converge instead.
func TestDBRecurrenceRecoversFromFailedRetire(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	series := "test-retry-" + time.Now().Format("150405.000000")
	src := seedRecurring(t, repo, series)

	runner := NewRecurrenceRunner(repo, time.Hour)
	ctx := context.Background()
	if err := runner.generate(ctx); err != nil {
		t.Fatalf("first generate: %v", err)
	}

	// Simulate the retirement having failed after the successor was written.
	if _, err := pool.Exec(ctx,
		`UPDATE bookings SET is_recurring = true, recurrence = $2 WHERE id = $1`,
		src.ID, RecurWeekly); err != nil {
		t.Fatalf("re-arm source: %v", err)
	}

	if err := runner.generate(ctx); err != nil {
		t.Fatalf("recovery generate: %v", err)
	}
	if got := countSeries(t, repo, series); got != 2 {
		t.Fatalf("duplicate occurrence created: %d bookings in series, want 2", got)
	}
	if isRecurring(t, repo, src.ID) {
		t.Fatal("source still recurring: generator would loop forever")
	}
}

// List loads every booking's crew in one batched query; each booking must end
// up with its own cleaners and not another row's.
func TestDBListLoadsAssignmentsPerBooking(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	var ids []int64
	rows, err := pool.Query(ctx, `SELECT id FROM cleaners ORDER BY id LIMIT 2`)
	if err != nil {
		t.Fatalf("query cleaners: %v", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan cleaner: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) < 2 {
		t.Skip("needs at least two cleaners in the database")
	}

	marker := "Batch Test " + time.Now().Format("150405.000000")
	made := map[int64]int64{} // booking id -> expected cleaner id
	for _, cleanerID := range ids {
		crew, err := repo.CleanerNames(ctx, []int64{cleanerID})
		if err != nil {
			t.Fatalf("resolve cleaner %d: %v", cleanerID, err)
		}
		b, err := repo.Create(ctx, Booking{
			CustomerName:    marker,
			ServiceType:     "office",
			ScheduledFor:    time.Now().Add(48 * time.Hour),
			DurationMinutes: 120,
			Status:          StatusPending,
			Cleaners:        []CleanerBrief{{ID: crew[0].ID, Name: crew[0].Name, Role: "primary"}},
		})
		if err != nil {
			t.Fatalf("create booking for cleaner %d: %v", cleanerID, err)
		}
		made[b.ID] = cleanerID
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM bookings WHERE customer_name = $1`, marker)
	})

	items, _, err := repo.List(ctx, pagination.Params{Limit: 200, Search: marker})
	if err != nil {
		t.Fatalf("list bookings: %v", err)
	}
	seen := 0
	for _, b := range items {
		want, ok := made[b.ID]
		if !ok {
			continue
		}
		seen++
		if len(b.Cleaners) != 1 {
			t.Fatalf("booking %d: got %d cleaners, want 1", b.ID, len(b.Cleaners))
		}
		if b.Cleaners[0].ID != want {
			t.Fatalf("booking %d: got cleaner %d, want %d", b.ID, b.Cleaners[0].ID, want)
		}
	}
	if seen != len(made) {
		t.Fatalf("listed %d of %d seeded bookings", seen, len(made))
	}
}

// RollForwardRecurring must leave the source due for retry when the successor
// cannot be written, rather than retiring it and dropping the series.
func TestDBRollForwardKeepsSourceOnInsertFailure(t *testing.T) {
	pool := testPool(t)
	repo := NewRepository(pool)
	series := "test-fail-" + time.Now().Format("150405.000000")
	src := seedRecurring(t, repo, series)

	ctx := context.Background()
	child := src
	child.ID = 0
	child.BookingNumber = ""
	child.CreatedAt = time.Time{}
	// An unknown cleaner makes the insert fail inside the transaction.
	child.Cleaners = []CleanerBrief{{ID: 0, Name: "nobody", Role: "primary"}}
	child.ScheduledFor = src.ScheduledFor.AddDate(0, 0, 7)

	if _, _, err := repo.RollForwardRecurring(ctx, src.ID, child); err == nil {
		t.Fatal("expected roll-forward to fail on a bad assignment")
	}
	if got := countSeries(t, repo, series); got != 1 {
		t.Fatalf("partial successor left behind: %d bookings in series, want 1", got)
	}
	if !isRecurring(t, repo, src.ID) {
		t.Fatal("source retired despite the successor failing: the series would stop")
	}
}
