package customers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestDBRenamePropagates checks that editing a customer's name and email
// updates their bookings, invoices and payments. Skips without
// TEST_DATABASE_URL.
func TestDBRenamePropagates(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := NewRepository(pool)

	stamp := time.Now().UnixNano()
	var custID, bookingID, invoiceID, paymentID int64
	bn := fmt.Sprintf("BK-RN-%d", stamp)
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (first_name, last_name, email, phone, address, property_type, area, status)
		 VALUES ('Old', 'Name', $1, $2, '1 Rd', 'condo', 'Silom', 'active') RETURNING id`,
		fmt.Sprintf("old-%d@test.local", stamp), fmt.Sprintf("+66%09d", stamp%1e9)).Scan(&custID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO bookings (booking_number, customer_id, customer_name, customer_email, service_type, scheduled_for, duration_minutes, address, status)
		 VALUES ($1, $2, 'Old Name', 'old@x', 'deep', now(), 60, '1 Rd', 'completed') RETURNING id`, bn, custID).Scan(&bookingID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO invoices (invoice_number, booking_id, booking_number, customer_name, customer_email, service_type, service_name, subtotal, total)
		 VALUES ($1, $2, $3, 'Old Name', 'old@x', 'deep', 'Deep', 100, 100) RETURNING id`, "INV-RN-"+bn, bookingID, bn).Scan(&invoiceID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO payments (invoice_number, invoice_id, customer_name, booking_number, amount, method, status)
		 VALUES ($1, $2, 'Old Name', $3, 100, 'cash', 'paid') RETURNING id`, "INV-RN-"+bn, invoiceID, bn).Scan(&paymentID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM payments WHERE id = $1`, paymentID)
		_, _ = pool.Exec(ctx, `DELETE FROM invoices WHERE id = $1`, invoiceID)
		_, _ = pool.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, bookingID)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, custID)
	})

	first, last, email := "New", "Person", fmt.Sprintf("new-%d@test.local", stamp)
	if _, err := repo.Update(ctx, custID, Patch{FirstName: &first, LastName: &last, Email: &email}); err != nil {
		t.Fatal(err)
	}
	var bName, bEmail, iName, iEmail, pName string
	_ = pool.QueryRow(ctx, `SELECT customer_name, customer_email FROM bookings WHERE id = $1`, bookingID).Scan(&bName, &bEmail)
	_ = pool.QueryRow(ctx, `SELECT customer_name, customer_email FROM invoices WHERE id = $1`, invoiceID).Scan(&iName, &iEmail)
	_ = pool.QueryRow(ctx, `SELECT customer_name FROM payments WHERE id = $1`, paymentID).Scan(&pName)
	if bName != "New Person" || iName != "New Person" || pName != "New Person" || bEmail != email || iEmail != email {
		t.Fatalf("not propagated: booking %q/%q invoice %q/%q payment %q", bName, bEmail, iName, iEmail, pName)
	}
}
