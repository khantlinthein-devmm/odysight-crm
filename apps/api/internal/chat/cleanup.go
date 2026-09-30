package chat

import (
	"context"
	"log/slog"
	"time"
)

// Cleaner deletes chat voice notes (and any older photos) once they are
// past FileRetention, so chat never fills the server's disk. It checks once
// a day; each file lives about 30 days.
type Cleaner struct {
	repo     *Repository
	files    *FileStore
	interval time.Duration
	now      func() time.Time
}

func NewCleaner(repo *Repository, files *FileStore) *Cleaner {
	return &Cleaner{repo: repo, files: files, interval: 24 * time.Hour, now: time.Now}
}

func (c *Cleaner) Run(ctx context.Context) {
	c.Sweep(ctx)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.Sweep(ctx)
		}
	}
}

// Sweep removes every expired file, in batches, and returns how many went.
func (c *Cleaner) Sweep(ctx context.Context) int {
	cutoff := c.now().Add(-FileRetention)
	removed := 0
	for {
		batch, err := c.repo.FilesOlderThan(ctx, cutoff, 200)
		if err != nil {
			slog.Warn("chat cleanup: list failed", "error", err)
			return removed
		}
		if len(batch) == 0 {
			break
		}
		for _, f := range batch {
			if err := c.files.Remove(f.FileName); err != nil {
				// Leave the row pointing at the file so the next run retries.
				slog.Warn("chat cleanup: delete failed", "file", f.FileName, "error", err)
				return removed
			}
			if err := c.repo.MarkFileExpired(ctx, f.MessageID); err != nil {
				slog.Warn("chat cleanup: mark failed", "message", f.MessageID, "error", err)
				return removed
			}
			removed++
		}
	}
	if removed > 0 {
		slog.Info("chat cleanup: expired voice notes removed", "count", removed)
	}
	return removed
}
