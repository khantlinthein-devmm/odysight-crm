// Package rowno gives records a gap-free display number: 1…N in the order
// they were created, recounted after a delete. Database ids never change or
// get reused (links, bookings and invoices point at them), so screens show
// this number instead.
package rowno

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Numbers returns the display number of each id in table. table must be a
// trusted constant, never user input.
func Numbers(ctx context.Context, q Querier, table string, ids []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx,
		`SELECT id, n FROM (SELECT id, row_number() OVER (ORDER BY id) AS n FROM `+table+`) t
		  WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
