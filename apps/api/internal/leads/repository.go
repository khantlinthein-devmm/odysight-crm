package leads

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("lead not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const leadColumns = `id, first_name, last_name, email, phone, status, source, created_at`

func scanLead(row pgx.Row) (Lead, error) {
	var l Lead
	err := row.Scan(&l.ID, &l.FirstName, &l.LastName, &l.Email, &l.Phone, &l.Status, &l.Source, &l.CreatedAt)
	return l, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Lead, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"first_name", "last_name", "email", "phone"} {
			orParts = append(orParts, col+" ILIKE $"+itoa(start+i))
			args = append(args, like)
			_ = i
		}
		conds = append(conds, "("+joinOr(orParts)+")")
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
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM leads `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count leads: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT `+leadColumns+` FROM leads ` + where + ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query leads: %w", err)
	}
	defer rows.Close()

	items := []Lead{}
	for rows.Next() {
		item, err := scanLead(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan lead: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }
func joinOr(parts []string) string { return joinWith(parts, " OR ") }
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

func (r *Repository) GetByID(ctx context.Context, id int64) (Lead, error) {
	l, err := scanLead(r.pool.QueryRow(ctx,
		`SELECT `+leadColumns+` FROM leads WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Lead{}, ErrNotFound
	}
	if err != nil {
		return Lead{}, fmt.Errorf("get lead %d: %w", id, err)
	}
	return l, nil
}

func (r *Repository) Create(ctx context.Context, l Lead) (Lead, error) {
	created, err := scanLead(r.pool.QueryRow(ctx,
		`INSERT INTO leads (first_name, last_name, email, phone, status, source)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+leadColumns,
		l.FirstName, l.LastName, l.Email, l.Phone, l.Status, l.Source))
	if err != nil {
		return Lead{}, fmt.Errorf("create lead: %w", err)
	}
	return created, nil
}

// Patch carries only the fields that should change; nil means "leave as is".
type Patch struct {
	FirstName *string
	LastName  *string
	Email     *string
	Phone     *string
	Status    *Status
	Source    *Source
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Lead, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}
	var source any
	if p.Source != nil {
		source = *p.Source
	}

	updated, err := scanLead(r.pool.QueryRow(ctx,
		`UPDATE leads SET
			first_name = COALESCE($2, first_name),
			last_name  = COALESCE($3, last_name),
			email      = COALESCE($4, email),
			phone      = COALESCE($5, phone),
			status     = COALESCE($6, status),
			source     = COALESCE($7, source)
		 WHERE id = $1
		 RETURNING `+leadColumns,
		id, p.FirstName, p.LastName, p.Email, p.Phone, status, source))
	if errors.Is(err, pgx.ErrNoRows) {
		return Lead{}, ErrNotFound
	}
	if err != nil {
		return Lead{}, fmt.Errorf("update lead %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM leads WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete lead %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
