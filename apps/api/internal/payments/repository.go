package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("payment not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const paymentColumns = `id, invoice_number, payer_name, amount, currency, method, status, created_at`

const invoiceNumberExpr = `'INV-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0')`

func scanPayment(row pgx.Row) (Payment, error) {
	var p Payment
	err := row.Scan(&p.ID, &p.InvoiceNumber, &p.PayerName, &p.Amount, &p.Currency, &p.Method, &p.Status, &p.CreatedAt)
	return p, err
}

func (r *Repository) List(ctx context.Context) ([]Payment, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+paymentColumns+` FROM payments ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query payments: %w", err)
	}
	defer rows.Close()

	payments := []Payment{}
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
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

// Create inserts the payment and assigns the invoice number from the new row's id.
func (r *Repository) Create(ctx context.Context, p Payment) (Payment, error) {
	var id int64
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO payments (invoice_number, payer_name, amount, currency, method, status)
		 VALUES ('', $1, $2, $3, $4, $5)
		 RETURNING id`,
		p.PayerName, p.Amount, p.Currency, p.Method, p.Status).Scan(&id); err != nil {
		return Payment{}, fmt.Errorf("create payment: %w", err)
	}

	created, err := scanPayment(r.pool.QueryRow(ctx,
		`UPDATE payments SET invoice_number = `+invoiceNumberExpr+`
		 WHERE id = $1
		 RETURNING `+paymentColumns,
		id))
	if err != nil {
		return Payment{}, fmt.Errorf("assign invoice number to payment %d: %w", id, err)
	}
	return created, nil
}

// Patch carries only the fields that should change; nil means "leave as is".
type Patch struct {
	Status *Status
	Method *string
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
