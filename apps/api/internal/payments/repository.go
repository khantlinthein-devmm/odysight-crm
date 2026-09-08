package payments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("payment not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const paymentColumns = `id, invoice_number, customer_name, booking_number, amount, currency, method, status, created_at`

// invoiceNumberExpr resolves the booking's real invoice number, falling back
// to a PAY- reference derived from the payment's own id when no invoice exists
// yet (keeps the column non-null, unique, and collision-free vs. INV- numbers).
const invoiceNumberExpr = `COALESCE(
		(SELECT invoice_number FROM invoices WHERE booking_number = $2 ORDER BY id DESC LIMIT 1),
		'PAY-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0')
	)`

func scanPayment(row pgx.Row) (Payment, error) {
	var p Payment
	err := row.Scan(&p.ID, &p.InvoiceNumber, &p.CustomerName, &p.BookingNumber, &p.Amount, &p.Currency, &p.Method, &p.Status, &p.CreatedAt)
	return p, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Payment, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"invoice_number", "customer_name", "booking_number"} {
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
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM payments `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count payments: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT `+paymentColumns+` FROM payments ` + where + ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query payments: %w", err)
	}
	defer rows.Close()

	items := []Payment{}
	for rows.Next() {
		item, err := scanPayment(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan payment: %w", err)
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

func (r *Repository) GetByID(ctx context.Context, id int64) (Payment, error) {
	p, err := scanPayment(r.pool.QueryRow(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, fmt.Errorf("get payment %d: %w", id, err)
	}
	return p, nil
}

// Create inserts the payment and assigns the booking's invoice number.
func (r *Repository) Create(ctx context.Context, p Payment) (Payment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Payment{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	temp := "TMP-" + fmt.Sprintf("%d", time.Now().UnixNano())
	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO payments (invoice_number, customer_name, booking_number, amount, currency, method, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		temp, p.CustomerName, p.BookingNumber, p.Amount, p.Currency, p.Method, p.Status).Scan(&id); err != nil {
		return Payment{}, fmt.Errorf("create payment: %w", err)
	}

	created, err := scanPayment(tx.QueryRow(ctx,
		`UPDATE payments SET invoice_number = `+invoiceNumberExpr+`
		 WHERE id = $1
		 RETURNING `+paymentColumns,
		id, p.BookingNumber))
	if err != nil {
		return Payment{}, fmt.Errorf("assign invoice number to payment %d: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Payment{}, fmt.Errorf("commit payment %d: %w", id, err)
	}
	return created, nil
}

// Patch carries only the fields that should change; nil means "leave as is".
type Patch struct {
	Status *Status
	Method *Method
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Payment, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanPayment(r.pool.QueryRow(ctx,
		`UPDATE payments SET
			status = COALESCE($2, status),
			method = COALESCE($3, method)
		 WHERE id = $1
		 RETURNING `+paymentColumns,
		id, status, p.Method))
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, fmt.Errorf("update payment %d: %w", id, err)
	}
	return updated, nil
}
