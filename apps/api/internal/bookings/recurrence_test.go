package bookings

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestNextOccurrence(t *testing.T) {
	base := time.Date(2026, 3, 2, 9, 30, 0, 0, time.UTC)
	cases := []struct {
		recurrence string
		want       time.Time
		ok         bool
	}{
		{RecurWeekly, base.AddDate(0, 0, 7), true},
		{RecurBiweekly, base.AddDate(0, 0, 14), true},
		{RecurMonthly, base.AddDate(0, 1, 0), true},
		{"yearly", time.Time{}, false},
		{"", time.Time{}, false},
	}
	for _, c := range cases {
		got, ok := nextOccurrence(base, c.recurrence)
		if ok != c.ok {
			t.Fatalf("%q: ok = %v, want %v", c.recurrence, ok, c.ok)
		}
		if ok && !got.Equal(c.want) {
			t.Fatalf("%q: got %v, want %v", c.recurrence, got, c.want)
		}
	}
}

func TestNextOccurrencePreservesWallClock(t *testing.T) {
	base := time.Date(2026, 3, 2, 9, 30, 0, 0, time.UTC)
	got, ok := nextOccurrence(base, RecurWeekly)
	if !ok {
		t.Fatal("weekly recurrence rejected")
	}
	if got.Hour() != 9 || got.Minute() != 30 {
		t.Fatalf("wall clock drifted: got %v", got)
	}
}

func TestIsDuplicateSeriesSlot(t *testing.T) {
	dup := &pgconn.PgError{Code: "23505", ConstraintName: "uq_bookings_series_slot"}
	if !isDuplicateSeriesSlot(dup) {
		t.Fatal("series slot violation not recognised")
	}
	// Wrapped the way the repository returns it.
	if !isDuplicateSeriesSlot(errors.Join(errors.New("create booking"), dup)) {
		t.Fatal("wrapped series slot violation not recognised")
	}
	// A different unique index must not be mistaken for it.
	other := &pgconn.PgError{Code: "23505", ConstraintName: "bookings_booking_number_key"}
	if isDuplicateSeriesSlot(other) {
		t.Fatal("unrelated unique violation treated as a duplicate slot")
	}
	if isDuplicateSeriesSlot(errors.New("boom")) {
		t.Fatal("plain error treated as a duplicate slot")
	}
}
