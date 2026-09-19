package expenses

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("expense not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const expenseColumns = `e.id, e.spent_on, e.category, e.amount, COALESCE(e.note, ''),
	COALESCE(e.created_by, 0), e.created_at`

func scanExpense(row pgx.Row) (Expense, error) {
	var e Expense
	var spentOn time.Time
	err := row.Scan(&e.ID, &spentOn, &e.Category, &e.Amount, &e.Note, &e.CreatedBy, &e.CreatedAt)
	if err != nil {
		return Expense{}, err
	}
	e.SpentOn = spentOn.Format(dateFormat)
	return e, nil
}

// Filters narrows a List query. Empty fields are ignored.
type Filters struct {
	From     string // YYYY-MM-DD inclusive
	To       string // YYYY-MM-DD inclusive
	Category string
}

func (r *Repository) List(ctx context.Context, f Filters, params pagination.Params) ([]Expense, int, error) {
	args := []any{}
	conds := []string{}
	if f.From != "" {
		args = append(args, f.From)
		conds = append(conds, "e.spent_on >= $"+itoa(len(args))+"::date")
	}
	if f.To != "" {
		args = append(args, f.To)
		conds = append(conds, "e.spent_on <= $"+itoa(len(args))+"::date")
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conds = append(conds, "e.category = $"+itoa(len(args)))
	}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		args = append(args, like, like)
		conds = append(conds, "(e.category ILIKE $"+itoa(start)+" OR e.note ILIKE $"+itoa(start+1)+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM expenses e `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expenses: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + expenseColumns + ` FROM expenses e ` + where +
		` ORDER BY e.spent_on DESC, e.id DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query expenses: %w", err)
	}
	defer rows.Close()

	items := []Expense{}
	for rows.Next() {
		item, err := scanExpense(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan expense: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Expense, error) {
	e, err := scanExpense(r.pool.QueryRow(ctx,
		`SELECT `+expenseColumns+` FROM expenses e WHERE e.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Expense{}, ErrNotFound
	}
	if err != nil {
		return Expense{}, fmt.Errorf("get expense %d: %w", id, err)
	}
	return e, nil
}

// Create inserts one expense. createdBy may be 0 for an unattributed row.
func (r *Repository) Create(ctx context.Context, e Expense) (Expense, error) {
	var createdBy any
	if e.CreatedBy > 0 {
		createdBy = e.CreatedBy
	}
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO expenses (spent_on, category, amount, note, created_by)
		 VALUES ($1::date, $2, $3, $4, $5)
		 RETURNING id`,
		e.SpentOn, e.Category, e.Amount, e.Note, createdBy).Scan(&id)
	if err != nil {
		return Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateExpenseRequest) (Expense, error) {
	var spentOn, category, note, amount any
	if req.SpentOn != nil {
		spentOn = *req.SpentOn
	}
	if req.Category != nil {
		category = *req.Category
	}
	if req.Note != nil {
		note = *req.Note
	}
	if req.Amount != nil {
		amount = *req.Amount
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE expenses SET
			spent_on = COALESCE($2::date, spent_on),
			category = COALESCE($3, category),
			amount   = COALESCE($4, amount),
			note     = COALESCE($5, note)
		 WHERE id = $1`,
		id, spentOn, category, amount, note)
	if err != nil {
		return Expense{}, fmt.Errorf("update expense %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return Expense{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete expense %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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
