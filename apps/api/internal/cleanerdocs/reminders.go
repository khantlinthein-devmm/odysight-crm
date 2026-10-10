package cleanerdocs

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"time"

	"github.com/odysight/crm/internal/notifications"
)

// ReminderRunner emails the office (Settings → Notifications → recipients)
// as each document crosses 60, 30 and 7 days before expiry and once more when
// it expires. reminder_stage makes runs idempotent across restarts; it is
// reset when the expiry date is edited. Single-instance only, like the
// invoice reminder runner. Each run also sweeps orphaned encrypted files.
type ReminderRunner struct {
	repo     *Repository
	store    *Store
	notifier *notifications.Service
	interval time.Duration
	now      func() time.Time
}

func NewReminderRunner(repo *Repository, store *Store, notifier *notifications.Service, interval time.Duration) *ReminderRunner {
	return &ReminderRunner{repo: repo, store: store, notifier: notifier, interval: interval, now: time.Now}
}

func (r *ReminderRunner) Run(ctx context.Context) {
	r.fire(ctx)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("document expiry runner stopping")
			return
		case <-ticker.C:
			r.fire(ctx)
		}
	}
}

func (r *ReminderRunner) fire(ctx context.Context) {
	if err := r.sendReminders(ctx); err != nil {
		slog.Warn("document expiry run failed", "error", err)
	}
	if r.store != nil {
		r.sweep(ctx)
	}
}

func (r *ReminderRunner) sendReminders(ctx context.Context) error {
	now := r.now()
	items, err := r.repo.Expiring(ctx, today(now).AddDate(0, 0, ExpiringWindowDays))
	if err != nil {
		return err
	}
	sent := 0
	for _, e := range items {
		days := DaysUntil(e.ExpiryDate, now)
		stage := reminderStage(days)
		if stage <= e.ReminderStage {
			continue
		}
		if r.notifier != nil {
			r.notifier.NotifyOffice(ctx, notifications.EventCleanerDocExpiring, reminderSubject(e, days), reminderBody(e, days))
		}
		if err := r.repo.SetReminderStage(ctx, e.DocumentID, stage); err != nil {
			slog.Warn("document expiry stamp failed", "document", e.DocumentID, "error", err)
			continue
		}
		sent++
	}
	if sent > 0 {
		slog.Info("document expiry scan complete", "candidates", len(items), "notified", sent)
	}
	return nil
}

func (r *ReminderRunner) sweep(ctx context.Context) {
	keep, err := r.repo.FileNames(ctx)
	if err != nil {
		slog.Warn("document sweep skipped", "error", err)
		return
	}
	if n, err := r.store.Sweep(keep, time.Hour); err != nil {
		slog.Warn("document sweep failed", "error", err)
	} else if n > 0 {
		slog.Info("removed orphaned document files", "count", n)
	}
}

func reminderSubject(e ExpiringItem, days int) string {
	if days < 0 {
		return fmt.Sprintf("%s of %s has expired", e.Type.Label(), e.CleanerName)
	}
	return fmt.Sprintf("%s of %s expires in %d days", e.Type.Label(), e.CleanerName, days)
}

func reminderBody(e ExpiringItem, days int) string {
	when := fmt.Sprintf("expires in <strong>%d days</strong>", days)
	if days < 0 {
		when = "<strong>has expired</strong>"
	}
	return `<div style="font-family:Arial,sans-serif;color:#1e293b;max-width:480px;margin:0 auto;">` +
		`<h2 style="margin-bottom:4px;">Document renewal needed</h2>` +
		`<p>The ` + html.EscapeString(e.Type.Label()) + ` of <strong>` + html.EscapeString(e.CleanerName) + `</strong> ` + when +
		` (` + e.ExpiryDate.Format("02/01/2006") + `).</p>` +
		`<p style="color:#475569;font-size:13px;">Update the new expiry date on the cleaner's Documents panel once it is renewed.</p>` +
		`</div>`
}
