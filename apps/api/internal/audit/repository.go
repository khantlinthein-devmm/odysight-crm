package audit

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

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Entry, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		args = append(args, like, like, like)
		conds = append(conds, fmt.Sprintf("(a.action ILIKE $%d OR a.resource ILIKE $%d OR u.email ILIKE $%d)", start, start+1, start+2))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + conds[0]
	}
	var total int
	countQ := `SELECT COUNT(*) FROM audit_logs a LEFT JOIN users u ON u.id = a.user_id ` + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT a.id, a.user_id, COALESCE(u.name, ''), COALESCE(u.email, ''),
		a.action, a.resource, a.resource_id, a.created_at
		FROM audit_logs a LEFT JOIN users u ON u.id = a.user_id ` + where +
		fmt.Sprintf(` ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	items := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserName, &e.UserEmail,
			&e.Action, &e.Resource, &e.ResourceID, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate audit logs: %w", err)
	}
	return items, total, nil
}
