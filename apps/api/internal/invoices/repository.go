package invoices

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var (
	ErrNotFound            = errors.New("invoice not found")
	ErrBookingNotFound     = errors.New("booking not found")
	ErrBookingNotCompleted = errors.New("booking not completed")
	ErrActiveInvoiceExists = errors.New("booking has an active invoice")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const invoiceColumns = `id, invoice_number, booking_id, booking_number, customer_name, customer_email, address,
	service_type, service_name, subtotal, tax_rate, tax_amount, total, currency, status,
	customer_tax_id, customer_tax_branch, withholding_rate::float8, withholding_amount::float8,
	contract_id, idempotency_key, billing_period_start, billing_period_end,
	issued_at, paid_at, created_at, updated_at`

func scanInvoice(row pgx.Row) (Invoice, error) {
	var inv Invoice
	err := row.Scan(&inv.ID, &inv.InvoiceNumber, &inv.BookingID, &inv.BookingNumber,
		&inv.CustomerName, &inv.CustomerEmail, &inv.Address, &inv.ServiceType, &inv.ServiceName,
		&inv.Subtotal, &inv.TaxRate, &inv.TaxAmount, &inv.Total, &inv.Currency,
		&inv.Status, &inv.CustomerTaxID, &inv.CustomerTaxBranch, &inv.WithholdingRate, &inv.WithholdingAmount,
		&inv.ContractID, &inv.IdempotencyKey, &inv.BillingPeriodStart, &inv.BillingPeriodEnd,
		&inv.IssuedAt, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt)
	return inv, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Invoice, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"invoice_number", "customer_name", "booking_number"} {
			orParts = append(orParts, col+" ILIKE $"+strconv.Itoa(start+i))
			args = append(args, like)
			_ = i
		}
		conds = append(conds, "("+strings.Join(orParts, " OR ")+")")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "status = $"+strconv.Itoa(len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM invoices `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + invoiceColumns + ` FROM invoices ` + where +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query invoices: %w", err)
	}
	defer rows.Close()

	items := []Invoice{}
	for rows.Next() {
		item, err := scanInvoice(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan invoice: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Invoice, error) {
	inv, err := scanInvoice(r.pool.QueryRow(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, fmt.Errorf("get invoice %d: %w", id, err)
	}
	return inv, nil
}

// GetActiveByBookingID returns any non-void invoice for a booking.
func (r *Repository) GetActiveByBookingID(ctx context.Context, bookingID int64) (Invoice, error) {
	inv, err := scanInvoice(r.pool.QueryRow(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE booking_id = $1 AND status <> 'void'`, bookingID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, fmt.Errorf("get active invoice for booking %d: %w", bookingID, err)
	}
	return inv, nil
}

// BookingForInvoice loads the booking fields needed to bill it.
func (r *Repository) BookingForInvoice(ctx context.Context, bookingID int64) (BookingSnapshot, error) {
	var b BookingSnapshot
	err := r.pool.QueryRow(ctx,
		`SELECT b.id, b.booking_number, b.customer_name, b.customer_email, b.address, b.service_type,
		        b.duration_minutes, b.status,
		        COALESCE(c.tax_id, ''), COALESCE(c.tax_branch, ''), COALESCE(c.withholding_rate, 0)::float8
		   FROM bookings b LEFT JOIN customers c ON c.id = b.customer_id
		  WHERE b.id = $1`, bookingID).
		Scan(&b.ID, &b.BookingNumber, &b.CustomerName, &b.CustomerEmail, &b.Address, &b.ServiceType,
			&b.DurationMinutes, &b.Status, &b.CustomerTaxID, &b.CustomerTaxBranch, &b.WithholdingRate)
	if errors.Is(err, pgx.ErrNoRows) {
		return BookingSnapshot{}, ErrBookingNotFound
	}
	if err != nil {
		return BookingSnapshot{}, fmt.Errorf("load booking %d for invoice: %w", bookingID, err)
	}
	b.Completed = b.Status == "completed"
	return b, nil
}

// GetByIdempotencyKey returns the invoice previously created for a contract
// billing key, letting retried scheduler runs return the existing row.
func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (Invoice, error) {
	inv, err := scanInvoice(r.pool.QueryRow(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE idempotency_key = $1`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, fmt.Errorf("get invoice by idempotency key: %w", err)
	}
	return inv, nil
}

// Create inserts the invoice; the number comes from the shared sequence.
func (r *Repository) Create(ctx context.Context, inv Invoice) (Invoice, error) {
	invNumberExpr := `'INV-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('invoice_seq')::text, 4, '0')`
	created, err := scanInvoice(r.pool.QueryRow(ctx,
		`INSERT INTO invoices (invoice_number, booking_id, booking_number, customer_name, customer_email, address,
			service_type, service_name, subtotal, tax_rate, tax_amount, total, currency, status,
			contract_id, idempotency_key, billing_period_start, billing_period_end,
			customer_tax_id, customer_tax_branch, withholding_rate, withholding_amount)
		 VALUES (`+invNumberExpr+`, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21)
		 RETURNING `+invoiceColumns,
		inv.BookingID, inv.BookingNumber, inv.CustomerName, inv.CustomerEmail, inv.Address,
		inv.ServiceType, inv.ServiceName, inv.Subtotal, inv.TaxRate, inv.TaxAmount,
		inv.Total, inv.Currency, inv.Status,
		inv.ContractID, inv.IdempotencyKey, inv.BillingPeriodStart, inv.BillingPeriodEnd,
		inv.CustomerTaxID, inv.CustomerTaxBranch, inv.WithholdingRate, inv.WithholdingAmount))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "uq_invoices_idempotency_key" {
				return Invoice{}, fmt.Errorf("duplicate idempotency key")
			}
			return Invoice{}, ErrActiveInvoiceExists
		}
		return Invoice{}, fmt.Errorf("create invoice: %w", err)
	}
	return created, nil
}

// Stack: status -> allowed next statuses.
var allowedTransitions = map[Status][]Status{
	StatusDraft:  {StatusIssued, StatusVoid},
	StatusIssued: {StatusPaid, StatusVoid},
	StatusPaid:   {StatusVoid},
	StatusVoid:   {},
}

// Update applies a status transition and returns the updated invoice.
func (r *Repository) Update(ctx context.Context, id int64, next Status) (Invoice, error) {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return Invoice{}, err
	}
	allowed, ok := allowedTransitions[current.Status]
	if !ok {
		return Invoice{}, fmt.Errorf("no transitions for status %q", current.Status)
	}
	if next == current.Status {
		// No-op; return current.
		return current, nil
	}
	permitted := false
	for _, s := range allowed {
		if s == next {
			permitted = true
			break
		}
	}
	if !permitted {
		return Invoice{}, fmt.Errorf("cannot change invoice %d from %q to %q", id, current.Status, next)
	}

	var paidAt any
	if next == StatusPaid {
		paidAt = time.Now()
	}
	updated, err := scanInvoice(r.pool.QueryRow(ctx,
		`UPDATE invoices SET
			status = $2,
			paid_at = COALESCE($3, paid_at)
		 WHERE id = $1
		 RETURNING `+invoiceColumns,
		id, next, paidAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, fmt.Errorf("update invoice %d: %w", id, err)
	}
	return updated, nil
}

// OverdueIssued lists issued, unpaid invoices that were never reminded and were
// issued before `before` (used to compute the overdue threshold).
func (r *Repository) OverdueIssued(ctx context.Context, before time.Time) ([]Invoice, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+invoiceColumns+` FROM invoices
		 WHERE status = 'issued' AND paid_at IS NULL AND reminder_sent_at IS NULL AND issued_at < $1
		   AND customer_email IS NOT NULL AND customer_email <> ''
		 ORDER BY issued_at ASC
		 LIMIT 100`, before)
	if err != nil {
		return nil, fmt.Errorf("query overdue invoices: %w", err)
	}
	defer rows.Close()

	items := []Invoice{}
	for rows.Next() {
		item, err := scanInvoice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan overdue invoice: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// MarkReminderSent records that an overdue reminder was emailed. Idempotent:
// only ever stamps an invoice that has not been stamped yet.
func (r *Repository) MarkReminderSent(ctx context.Context, id int64) error {
	if _, err := r.pool.Exec(ctx,
		`UPDATE invoices SET reminder_sent_at = now() WHERE id = $1 AND reminder_sent_at IS NULL`, id); err != nil {
		return fmt.Errorf("mark reminder sent for invoice %d: %w", id, err)
	}
	return nil
}

// MarkPaidForBooking settles any active invoices for a booking (payments hook).
func (r *Repository) MarkPaidForBooking(ctx context.Context, bookingNumber string) error {
	if _, err := r.pool.Exec(ctx,
		`UPDATE invoices SET status = 'paid', paid_at = COALESCE(paid_at, now())
		 WHERE booking_number = $1 AND status IN ('draft', 'issued')`,
		bookingNumber); err != nil {
		return fmt.Errorf("mark invoices paid for %s: %w", bookingNumber, err)
	}
	return nil
}

// GetByBookingNumber returns the most recent invoice for a booking.
func (r *Repository) GetByBookingNumber(ctx context.Context, bookingNumber string) (Invoice, error) {
	inv, err := scanInvoice(r.pool.QueryRow(ctx,
		`SELECT `+invoiceColumns+` FROM invoices
		 WHERE booking_number = $1 ORDER BY id DESC LIMIT 1`, bookingNumber))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, fmt.Errorf("get invoice for booking %s: %w", bookingNumber, err)
	}
	return inv, nil
}
