package invoices

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/odysight/crm/pkg/pagination"
)

var (
	ErrReceiptNotFound    = errors.New("receipt not found")
	ErrPaymentNotFound    = errors.New("payment not found")
	ErrInvoiceNotOpen     = errors.New("invoice is not open for payment")
	ErrInvoiceFullyPaid   = errors.New("invoice is already fully paid")
	ErrOverpayment        = errors.New("payment exceeds the balance due")
	ErrInvoiceHasPaid     = errors.New("invoice has payments")
	ErrInvoiceHasReceipts = errors.New("invoice has receipts")
	ErrInvoiceNotPaid     = errors.New("invoice has nothing paid")
)

const (
	ReceiptValid     = "valid"
	ReceiptCancelled = "cancelled"
)

// Receipt acknowledges one paid payment against an invoice. Amount is the
// cash received; Subtotal, VAT and WHT are the parts of the invoice that the
// payment settles (Subtotal + VAT - WHT == Amount).
type Receipt struct {
	ID              int64
	ReceiptNumber   string
	InvoiceID       int64
	PaymentID       int64
	Amount          float64
	Subtotal        float64
	VAT             float64
	WHT             float64
	TaxRate         float64
	WithholdingRate float64
	Currency        string
	Method          string
	Reference       string
	VATRegistered   bool
	Status          string
	PaidAt          time.Time
	CancelledAt     *time.Time
	CreatedAt       time.Time
	// Read from the invoice snapshot.
	InvoiceNumber     string
	BookingID         int64
	BookingNumber     string
	CustomerName      string
	CustomerEmail     string
	Address           string
	ServiceName       string
	CustomerTaxID     string
	CustomerTaxBranch string
}

// Gross is the invoice value (service + VAT) the receipt covers.
func (r Receipt) Gross() float64 { return round2(r.Subtotal + r.VAT) }

// ReceiptParts is how a payment divides into service value, VAT and WHT.
type ReceiptParts struct {
	Subtotal float64
	VAT      float64
	WHT      float64
}

// splitPayment apportions cash received against an invoice. Money received is
// net of withholding tax, so it covers received*total/net of the invoice; VAT
// is that covered amount's pro-rata share. The payment that clears the balance
// takes whatever each part has left, so the receipts always add up to the
// invoice exactly. prior is the sum of the invoice's earlier valid receipts.
func splitPayment(inv Invoice, prior ReceiptParts, received float64, final bool) ReceiptParts {
	if final {
		rest := ReceiptParts{
			Subtotal: round2(inv.Subtotal - prior.Subtotal),
			VAT:      round2(inv.TaxAmount - prior.VAT),
			WHT:      round2(inv.WithholdingAmount - prior.WHT),
		}
		// Only trust the remainder when it reconciles with the cash (it will
		// not for invoices settled by a legacy payment without a receipt).
		if math.Abs(rest.Subtotal+rest.VAT-rest.WHT-received) < 0.005 && rest.Subtotal >= 0 && rest.VAT >= 0 && rest.WHT >= 0 {
			return rest
		}
	}
	net := inv.NetPayable()
	if net <= 0 || inv.Total <= 0 {
		return ReceiptParts{Subtotal: round2(received)}
	}
	covered := received * inv.Total / net
	vat := round2(covered * inv.TaxAmount / inv.Total)
	covered = round2(covered)
	return ReceiptParts{
		Subtotal: round2(covered - vat),
		VAT:      vat,
		WHT:      round2(covered - received),
	}
}

// nextStatus is the invoice status once amountPaid has been received.
func nextStatus(inv Invoice, amountPaid float64) Status {
	switch {
	case amountPaid >= inv.NetPayable()-0.005:
		return StatusPaid
	case amountPaid > 0.005:
		return StatusPartiallyPaid
	default:
		return StatusIssued
	}
}

const receiptSelect = `SELECT r.id, r.receipt_number, r.invoice_id, r.payment_id, r.amount::float8,
		r.subtotal_part::float8, r.vat_part::float8, r.wht_part::float8, r.tax_rate::float8, r.withholding_rate::float8,
		r.currency, r.method, r.reference, r.vat_registered, r.status, r.paid_at, r.cancelled_at, r.created_at,
		i.invoice_number, COALESCE(i.booking_id, 0), i.booking_number, i.customer_name, i.customer_email, i.address,
		i.service_name, i.customer_tax_id, i.customer_tax_branch
	FROM receipts r JOIN invoices i ON i.id = r.invoice_id`

func scanReceipt(row pgx.Row) (Receipt, error) {
	var rc Receipt
	err := row.Scan(&rc.ID, &rc.ReceiptNumber, &rc.InvoiceID, &rc.PaymentID, &rc.Amount,
		&rc.Subtotal, &rc.VAT, &rc.WHT, &rc.TaxRate, &rc.WithholdingRate,
		&rc.Currency, &rc.Method, &rc.Reference, &rc.VATRegistered, &rc.Status, &rc.PaidAt, &rc.CancelledAt, &rc.CreatedAt,
		&rc.InvoiceNumber, &rc.BookingID, &rc.BookingNumber, &rc.CustomerName, &rc.CustomerEmail, &rc.Address,
		&rc.ServiceName, &rc.CustomerTaxID, &rc.CustomerTaxBranch)
	return rc, err
}

// ListReceipts pages receipts, newest first, optionally for one invoice.
func (r *Repository) ListReceipts(ctx context.Context, params pagination.Params, invoiceID int64) ([]Receipt, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		args = append(args, "%"+params.Search+"%")
		n := "$" + strconv.Itoa(len(args))
		conds = append(conds, "(r.receipt_number ILIKE "+n+" OR i.invoice_number ILIKE "+n+
			" OR i.customer_name ILIKE "+n+" OR i.booking_number ILIKE "+n+")")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "r.status = $"+strconv.Itoa(len(args)))
	}
	if invoiceID > 0 {
		args = append(args, invoiceID)
		conds = append(conds, "r.invoice_id = $"+strconv.Itoa(len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM receipts r JOIN invoices i ON i.id = r.invoice_id`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count receipts: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, receiptSelect+where+
		` ORDER BY r.created_at DESC, r.id DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query receipts: %w", err)
	}
	defer rows.Close()
	items := []Receipt{}
	for rows.Next() {
		rc, err := scanReceipt(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan receipt: %w", err)
		}
		items = append(items, rc)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetReceipt(ctx context.Context, id int64) (Receipt, error) {
	rc, err := scanReceipt(r.pool.QueryRow(ctx, receiptSelect+` WHERE r.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, ErrReceiptNotFound
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("get receipt %d: %w", id, err)
	}
	return rc, nil
}

// PaymentInput is a payment received against an invoice.
type PaymentInput struct {
	Amount        float64
	Method        string
	Reference     string
	PaidAt        time.Time
	VATRegistered bool
}

// RecordPayment stores a paid payment against the invoice and issues its
// receipt, all in one transaction so the invoice balance, the payment and
// the receipt can never disagree.
func (r *Repository) RecordPayment(ctx context.Context, invoiceID int64, in PaymentInput) (Invoice, Receipt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, Receipt{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inv, err := lockInvoice(ctx, tx, invoiceID)
	if err != nil {
		return Invoice{}, Receipt{}, err
	}
	if err := checkPayable(inv, in.Amount); err != nil {
		return Invoice{}, Receipt{}, err
	}
	var paymentID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO payments (invoice_number, invoice_id, customer_name, booking_number, amount, currency,
			method, status, reference, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'paid', $8, $9)
		 RETURNING id`,
		inv.InvoiceNumber, inv.ID, inv.CustomerName, inv.BookingNumber, in.Amount, inv.Currency,
		in.Method, in.Reference, in.PaidAt).Scan(&paymentID); err != nil {
		return Invoice{}, Receipt{}, fmt.Errorf("insert payment for invoice %d: %w", inv.ID, err)
	}
	updated, rc, err := applyPayment(ctx, tx, inv, paymentID, in)
	if err != nil {
		return Invoice{}, Receipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, Receipt{}, fmt.Errorf("commit payment for invoice %d: %w", inv.ID, err)
	}
	return updated, rc, nil
}

// OpenInvoiceIDForBooking returns the booking's invoice that can still take
// payments, or ErrNotFound.
func (r *Repository) OpenInvoiceIDForBooking(ctx context.Context, bookingNumber string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM invoices
		 WHERE booking_number = $1 AND status IN ('draft', 'issued', 'partially_paid')
		 ORDER BY id DESC LIMIT 1`, bookingNumber).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("open invoice for booking %s: %w", bookingNumber, err)
	}
	return id, nil
}

// SettledPayment is what happened when a pending payment became paid.
type SettledPayment struct {
	Invoice *Invoice
	Receipt *Receipt
}

// SettlePayment marks an existing payment paid and, when it belongs to an
// invoice that is still open, applies it and issues the receipt.
func (r *Repository) SettlePayment(ctx context.Context, paymentID int64, vatRegistered bool) (SettledPayment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SettledPayment{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status, method, reference, bookingNumber string
	var invoiceID *int64
	var amount float64
	err = tx.QueryRow(ctx,
		`SELECT status, invoice_id, amount::float8, method, reference, booking_number
		   FROM payments WHERE id = $1 FOR UPDATE`, paymentID).
		Scan(&status, &invoiceID, &amount, &method, &reference, &bookingNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return SettledPayment{}, ErrPaymentNotFound
	}
	if err != nil {
		return SettledPayment{}, fmt.Errorf("lock payment %d: %w", paymentID, err)
	}
	if status == "paid" {
		return SettledPayment{}, nil
	}
	// A payment recorded before its booking was invoiced links up now.
	if invoiceID == nil && bookingNumber != "" {
		var id int64
		var number string
		err := tx.QueryRow(ctx,
			`SELECT id, invoice_number FROM invoices
			 WHERE booking_number = $1 AND status IN ('draft', 'issued', 'partially_paid')
			 ORDER BY id DESC LIMIT 1`, bookingNumber).Scan(&id, &number)
		if err == nil {
			invoiceID = &id
			if _, err := tx.Exec(ctx, `UPDATE payments SET invoice_id = $2, invoice_number = $3 WHERE id = $1`,
				paymentID, id, number); err != nil {
				return SettledPayment{}, fmt.Errorf("link payment %d to invoice %d: %w", paymentID, id, err)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return SettledPayment{}, fmt.Errorf("find invoice for payment %d: %w", paymentID, err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'paid' WHERE id = $1`, paymentID); err != nil {
		return SettledPayment{}, fmt.Errorf("mark payment %d paid: %w", paymentID, err)
	}
	var out SettledPayment
	if invoiceID != nil {
		inv, err := lockInvoice(ctx, tx, *invoiceID)
		if err != nil {
			return SettledPayment{}, err
		}
		if inv.Status != StatusVoid {
			if err := checkPayable(inv, amount); err != nil {
				return SettledPayment{}, err
			}
			updated, rc, err := applyPayment(ctx, tx, inv, paymentID, PaymentInput{
				Amount: amount, Method: method, Reference: reference, PaidAt: time.Now(), VATRegistered: vatRegistered,
			})
			if err != nil {
				return SettledPayment{}, err
			}
			out = SettledPayment{Invoice: &updated, Receipt: &rc}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SettledPayment{}, fmt.Errorf("commit settle payment %d: %w", paymentID, err)
	}
	return out, nil
}

// RefundPayment marks a paid payment refunded, cancels its receipt and takes
// the money back off the invoice balance.
func (r *Repository) RefundPayment(ctx context.Context, paymentID int64) (*Invoice, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var invoiceID *int64
	var amount float64
	err = tx.QueryRow(ctx,
		`SELECT status, invoice_id, amount::float8 FROM payments WHERE id = $1 FOR UPDATE`, paymentID).
		Scan(&status, &invoiceID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock payment %d: %w", paymentID, err)
	}
	if status == "refunded" {
		return nil, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'refunded' WHERE id = $1`, paymentID); err != nil {
		return nil, fmt.Errorf("mark payment %d refunded: %w", paymentID, err)
	}
	var out *Invoice
	if invoiceID != nil && status == "paid" {
		inv, err := lockInvoice(ctx, tx, *invoiceID)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE receipts SET status = 'cancelled', cancelled_at = now()
			 WHERE payment_id = $1 AND status = 'valid'`, paymentID); err != nil {
			return nil, fmt.Errorf("cancel receipt for payment %d: %w", paymentID, err)
		}
		if inv.Status != StatusVoid {
			paid := math.Max(0, round2(inv.AmountPaid-amount))
			next := nextStatus(inv, paid)
			updated, err := scanInvoice(tx.QueryRow(ctx,
				`UPDATE invoices SET amount_paid = $2, status = $3,
					paid_at = CASE WHEN $3 = 'paid' THEN paid_at ELSE NULL END
				 WHERE id = $1 RETURNING `+invoiceColumns, inv.ID, paid, next))
			if err != nil {
				return nil, fmt.Errorf("reopen invoice %d: %w", inv.ID, err)
			}
			out = &updated
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refund payment %d: %w", paymentID, err)
	}
	return out, nil
}

func lockInvoice(ctx context.Context, tx pgx.Tx, id int64) (Invoice, error) {
	inv, err := scanInvoice(tx.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM invoices WHERE id = $1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, fmt.Errorf("lock invoice %d: %w", id, err)
	}
	return inv, nil
}

func checkPayable(inv Invoice, amount float64) error {
	switch {
	case inv.Status == StatusPaid:
		return ErrInvoiceFullyPaid
	case !inv.Status.Open():
		return ErrInvoiceNotOpen
	case amount > inv.BalanceDue()+0.005:
		return fmt.Errorf("%w (%.2f due)", ErrOverpayment, inv.BalanceDue())
	}
	return nil
}

// applyPayment moves the invoice balance and issues the receipt for a payment
// row that already exists. The caller holds the invoice row lock.
func applyPayment(ctx context.Context, tx pgx.Tx, inv Invoice, paymentID int64, in PaymentInput) (Invoice, Receipt, error) {
	var prior ReceiptParts
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(subtotal_part), 0)::float8, COALESCE(SUM(vat_part), 0)::float8, COALESCE(SUM(wht_part), 0)::float8
		   FROM receipts WHERE invoice_id = $1 AND status = 'valid'`, inv.ID).
		Scan(&prior.Subtotal, &prior.VAT, &prior.WHT); err != nil {
		return Invoice{}, Receipt{}, fmt.Errorf("sum receipts for invoice %d: %w", inv.ID, err)
	}
	final := in.Amount >= inv.BalanceDue()-0.005
	parts := splitPayment(inv, prior, in.Amount, final)
	paid := round2(inv.AmountPaid + in.Amount)
	next := nextStatus(inv, paid)

	updated, err := scanInvoice(tx.QueryRow(ctx,
		`UPDATE invoices SET amount_paid = $2, status = $3,
			paid_at = CASE WHEN $3 = 'paid' THEN COALESCE(paid_at, $4) ELSE paid_at END
		 WHERE id = $1 RETURNING `+invoiceColumns, inv.ID, paid, next, in.PaidAt))
	if err != nil {
		return Invoice{}, Receipt{}, fmt.Errorf("apply payment to invoice %d: %w", inv.ID, err)
	}

	var receiptID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO receipts (receipt_number, invoice_id, payment_id, amount, subtotal_part, vat_part, wht_part,
			tax_rate, withholding_rate, currency, method, reference, vat_registered, paid_at)
		 VALUES ('RC-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('receipt_seq')::text, 4, '0'),
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING id`,
		inv.ID, paymentID, in.Amount, parts.Subtotal, parts.VAT, parts.WHT,
		inv.TaxRate, inv.WithholdingRate, inv.Currency, in.Method, in.Reference, in.VATRegistered, in.PaidAt).
		Scan(&receiptID); err != nil {
		return Invoice{}, Receipt{}, fmt.Errorf("issue receipt for invoice %d: %w", inv.ID, err)
	}
	rc, err := scanReceipt(tx.QueryRow(ctx, receiptSelect+` WHERE r.id = $1`, receiptID))
	if err != nil {
		return Invoice{}, Receipt{}, fmt.Errorf("load receipt %d: %w", receiptID, err)
	}
	return updated, rc, nil
}
