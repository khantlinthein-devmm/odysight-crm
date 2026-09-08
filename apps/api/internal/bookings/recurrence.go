package bookings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"
)

// RecurrenceRunner generates the next occurrence for each due recurring
// booking. A recurring booking keeps ONE current occurrence; once its
// scheduled time is more than a day in the past, the runner creates the next
// occurrence (same series_id, shifted schedule) and marks the old row as
// non-recurring so it can never fire again. This yields an open-ended forward
// chain and is fully idempotent across restarts.
type RecurrenceRunner struct {
	repo     *Repository
	interval time.Duration
}

func NewRecurrenceRunner(repo *Repository, interval time.Duration) *RecurrenceRunner {
	return &RecurrenceRunner{repo: repo, interval: interval}
}

// Run blocks until ctx is cancelled, firing once immediately then every interval.
func (r *RecurrenceRunner) Run(ctx context.Context) {
	r.fire(ctx)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("recurring booking runner stopping")
			return
		case <-ticker.C:
			r.fire(ctx)
		}
	}
}

func (r *RecurrenceRunner) fire(ctx context.Context) {
	if err := r.generate(ctx); err != nil {
		slog.Warn("recurring booking run failed", "error", err)
	}
}

func (r *RecurrenceRunner) generate(ctx context.Context) error {
	// Wait a full day after the scheduled slot before spawning the successor
	// so short delays (or status edits) never create duplicates mid-day.
	cutoff := time.Now().Add(-24 * time.Hour)
	due, err := r.repo.DueRecurring(ctx, cutoff)
	if err != nil {
		return err
	}
	if len(due) == 0 {
		return nil
	}

	created := 0
	for _, src := range due {
		next, ok := nextOccurrence(src.ScheduledFor, src.Recurrence)
		if !ok {
			slog.Warn("recurring booking has unknown frequency", "id", src.ID, "recurrence", src.Recurrence)
			if err := r.repo.DisableRecurring(ctx, src.ID); err != nil {
				slog.Warn("disable invalid recurring booking failed", "id", src.ID, "error", err)
			}
			continue
		}
		child := src
		child.ID = 0
		child.BookingNumber = ""
		child.ScheduledFor = next
		child.Status = StatusPending
		child.CreatedAt = time.Time{}
		child.Cleaners = append([]CleanerBrief(nil), src.Cleaners...)

		if _, err := r.repo.Create(ctx, child); err != nil {
			slog.Warn("recurring successor creation failed", "series", *src.SeriesID, "error", err)
			continue
		}
		if err := r.repo.DisableRecurring(ctx, src.ID); err != nil {
			slog.Warn("recurring source disable failed", "id", src.ID, "error", err)
			continue
		}
		created++
	}
	if created > 0 {
		slog.Info("recurring booking generation complete", "due", len(due), "created", created)
	}
	return nil
}

// nextOccurrence shifts a time forward by the recurrence interval, preserving
// the wall-clock time. It reports false for unknown frequencies.
func nextOccurrence(t time.Time, recurrence string) (time.Time, bool) {
	switch recurrence {
	case RecurWeekly:
		return t.AddDate(0, 0, 7), true
	case RecurBiweekly:
		return t.AddDate(0, 0, 14), true
	case RecurMonthly:
		// AddDate(0, 1, 0) clamps e.g. Jan 31 -> Mar 3; acceptable for
		// monthly cleaning schedules and keeps the schedule strictly forward.
		return t.AddDate(0, 1, 0), true
	}
	return time.Time{}, false
}

// newSeriesID returns a random hex identifier linking every occurrence of a
// recurring schedule together.
func newSeriesID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(buf[:])
}