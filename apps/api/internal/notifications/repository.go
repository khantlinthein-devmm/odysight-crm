package notifications

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Log(ctx context.Context, entry LogEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_logs (channel, event_type, recipient, subject, status, error)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		entry.Channel, entry.EventType, entry.Recipient, entry.Subject, entry.Status, entry.Error)
	if err != nil {
		return fmt.Errorf("log notification: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]LogEntry, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		args = append(args, like, like, like)
		conds = append(conds, "(recipient ILIKE $"+itoa(start)+" OR subject ILIKE $"+itoa(start+1)+" OR event_type ILIKE $"+itoa(start+2)+")")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "status = $"+itoa(len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_logs `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notification logs: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT id, channel, event_type, recipient, subject, status, error, created_at
		FROM notification_logs ` + where +
		` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query notification logs: %w", err)
	}
	defer rows.Close()

	items := []LogEntry{}
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.Channel, &e.EventType, &e.Recipient, &e.Subject, &e.Status, &e.Error, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan notification log: %w", err)
		}
		items = append(items, e)
	}
	return items, total, rows.Err()
}

func itoa(i int) string             { return fmt.Sprintf("%d", i) }
func joinAnd(parts []string) string { return joinWith(parts, " AND ") }
func joinWith(parts []string, sep string) string {
	out := ""
	for i, s := range parts {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}