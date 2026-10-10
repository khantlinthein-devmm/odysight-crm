package invoices

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/internal/settings"
	"github.com/odysight/crm/pkg/pagination"
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.005 }

func TestSplitPaymentFull(t *testing.T) {
	inv := sampleInvoice() // 10,000 + 7% VAT, 3% WHT → net 10,400
	p := splitPayment(inv, ReceiptParts{}, 10400, true)
	if p.Subtotal != 10000 || p.VAT != 700 || p.WHT != 300 {
		t.Fatalf("full payment split = %+v", p)
	}
}

func TestSplitPaymentPartialsAddUp(t *testing.T) {
	inv := sampleInvoice()
	var prior ReceiptParts
	payments := []float64{3333.33, 2500, 4566.67}
	for i, amt := range payments {
		final := i == len(payments)-1
		p := splitPayment(inv, prior, amt, final)
		if !near(p.Subtotal+p.VAT-p.WHT, amt) {
			t.Fatalf("payment %d: parts %+v do not reconcile with %.2f", i, p, amt)
		}
		prior.Subtotal += p.Subtotal
		prior.VAT += p.VAT
		prior.WHT += p.WHT
	}
	if !near(prior.Subtotal, 10000) || !near(prior.VAT, 700) || !near(prior.WHT, 300) {
		t.Fatalf("receipts sum to %+v, want the invoice amounts", prior)
	}
}

func TestSplitPaymentVATInclusiveHalves(t *testing.T) {
	// 9,000 incl. VAT is 8,411.21 + 588.79, paid as 4,500 + 4,500. The
	// receipts must add up to the invoice exactly, not 8,411.22 + 588.78.
	sub, vat, total := splitVAT(9000, 7, true)
	inv := Invoice{Subtotal: sub, TaxRate: 7, TaxAmount: vat, Total: total, Currency: "THB", Status: StatusIssued}
	p1 := splitPayment(inv, ReceiptParts{}, 4500, false)
	p2 := splitPayment(inv, p1, 4500, true)
	if !near(p1.Subtotal+p2.Subtotal, 8411.21) || !near(p1.VAT+p2.VAT, 588.79) {
		t.Fatalf("receipts %+v + %+v do not add up to 8,411.21 + 588.79", p1, p2)
	}
	for i, p := range []ReceiptParts{p1, p2} {
		if !near(p.Subtotal+p.VAT, 4500) {
			t.Errorf("receipt %d parts %+v do not reconcile with 4,500", i+1, p)
		}
	}
}

func TestSplitPaymentFinalClosesVATRounding(t *testing.T) {
	// The earlier receipts do not reconcile with the cash (the subtotal is a
	// satang off), so the remainder cannot be used as is; the final VAT must
	// still bring the receipted VAT to the invoice's 588.79.
	sub, vat, total := splitVAT(9000, 7, true)
	inv := Invoice{Subtotal: sub, TaxRate: 7, TaxAmount: vat, Total: total, Currency: "THB", Status: StatusIssued}
	prior := ReceiptParts{Subtotal: 4205.63, VAT: 294.38}
	p := splitPayment(inv, prior, 4500, true)
	if !near(prior.VAT+p.VAT, 588.79) || !near(p.Subtotal+p.VAT, 4500) {
		t.Fatalf("final split %+v after %+v", p, prior)
	}
}

func TestInstalmentLabelAndBalance(t *testing.T) {
	sub, vat, total := splitVAT(9000, 7, true)
	inv := Invoice{Subtotal: sub, TaxRate: 7, TaxAmount: vat, Total: total, Currency: "THB", Status: StatusPaid, AmountPaid: 9000}
	cases := []struct {
		name               string
		paidBefore, amount float64
		label              string
		balance            float64
	}{
		{"first half", 0, 4500, "ชำระบางส่วน / Partial payment", 4500},
		{"second half settles", 4500, 4500, "ชำระครบ / Final payment", 0},
		{"single full payment", 0, 9000, "", 0},
	}
	for _, c := range cases {
		rc := Receipt{PaidBefore: c.paidBefore, Amount: c.amount}
		if got := instalmentLabel(rc, inv); got != c.label {
			t.Errorf("%s: label %q, want %q", c.name, got, c.label)
		}
		if got := rc.BalanceAfter(inv); !near(got, c.balance) {
			t.Errorf("%s: balance after %v, want %v", c.name, got, c.balance)
		}
	}
}

func TestSplitPaymentWithoutVATOrWHT(t *testing.T) {
	inv := Invoice{Subtotal: 1500, Total: 1500, Currency: "THB", Status: StatusIssued}
	p := splitPayment(inv, ReceiptParts{}, 500, false)
	if p.Subtotal != 500 || p.VAT != 0 || p.WHT != 0 {
		t.Fatalf("plain partial split = %+v", p)
	}
}

func TestNextStatusAndBalance(t *testing.T) {
	inv := sampleInvoice()
	if got := nextStatus(inv, 0); got != StatusIssued {
		t.Errorf("nothing paid → %s", got)
	}
	if got := nextStatus(inv, 100); got != StatusPartiallyPaid {
		t.Errorf("part paid → %s", got)
	}
	if got := nextStatus(inv, 10400); got != StatusPaid {
		t.Errorf("fully paid → %s", got)
	}
	inv.AmountPaid = 4000
	if got := inv.BalanceDue(); got != 6400 {
		t.Errorf("balance = %v, want 6400", got)
	}
	if err := checkPayable(inv, 6400.01); !errors.Is(err, ErrOverpayment) {
		t.Errorf("overpayment accepted: %v", err)
	}
	if err := checkPayable(inv, 6400); err != nil {
		t.Errorf("exact balance rejected: %v", err)
	}
}

func TestRenderReceiptPDF(t *testing.T) {
	inv := sampleInvoice()
	inv.AmountPaid = 5200
	rc := Receipt{ReceiptNumber: "RC-2026-0001", InvoiceNumber: inv.InvoiceNumber, BookingNumber: inv.BookingNumber,
		CustomerName: inv.CustomerName, Address: inv.Address, ServiceName: inv.ServiceName,
		CustomerTaxID: inv.CustomerTaxID, CustomerTaxBranch: inv.CustomerTaxBranch,
		Amount: 5200, Subtotal: 5000, VAT: 350, WHT: 150, TaxRate: 7, WithholdingRate: 3,
		Currency: "THB", Method: "bank_transfer", Reference: "KBank 0930", VATRegistered: true,
		Status: ReceiptValid, PaidAt: time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)}
	company := settings.Company{Name: "Smile Clean CO.,LTD.", LegalName: "บริษัท สไมล์ คลีน จำกัด",
		Address: "300 ซอยอ่อนนุช 10 แขวงสวนหลวง เขตสวนหลวง กรุงเทพมหานคร 10250", Phone: "062-2828209, 099-015-1961",
		TaxID: "0245562004320", VATRegistered: true}
	inv.SiteName = "สาขาสุขุมวิท 39"
	pay := settings.PaymentSettings{BankAccount: "ธนาคารกสิกรไทย 063-1-50243-1"}
	pdf, err := renderReceiptPDF(rc, inv, company, pay)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 5000 {
		t.Fatalf("receipt PDF too small (%d bytes)", len(pdf))
	}
	// Original and copy.
	if n := strings.Count(string(pdf), "/Type /Page\n"); n != 2 {
		t.Errorf("receipt pages = %d, want 2 (original + copy)", n)
	}
	if out := os.Getenv("RECEIPT_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, pdf, 0o644)
	}

	// The final instalment carries the payment summary and still fits on one
	// page per copy.
	inv.AmountPaid, inv.Status = 10400, StatusPaid
	rc.ReceiptNumber, rc.PaidBefore = "RC-2026-0002", 5200
	final, err := renderReceiptPDF(rc, inv, company, pay)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(final), "/Type /Page\n"); n != 2 {
		t.Errorf("final receipt pages = %d, want 2 (original + copy)", n)
	}
	if out := os.Getenv("RECEIPT_FINAL_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, final, 0o644)
	}
}

func TestRenderSummaryReceiptPDF(t *testing.T) {
	sub, vat, total := splitVAT(9000, 7, true)
	paid := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	inv := Invoice{InvoiceNumber: "INV-2026-0050", BookingNumber: "BK-2026-0200", CustomerName: "คุณสมชาย",
		Address: "1 ถนนสุขุมวิท กรุงเทพฯ", ServiceName: "ทำความสะอาดบ้าน / Home cleaning",
		Subtotal: sub, TaxRate: 7, TaxAmount: vat, Total: total, Currency: "THB",
		Status: StatusPaid, AmountPaid: 9000, IssuedAt: paid.AddDate(0, 0, -10), PaidAt: &paid}
	p1 := splitPayment(inv, ReceiptParts{}, 4500, false)
	p2 := splitPayment(inv, p1, 4500, true)
	rcs := []Receipt{
		{ReceiptNumber: "RC-2026-0015", Amount: 4500, Subtotal: p1.Subtotal, VAT: p1.VAT, PaidAt: paid.AddDate(0, 0, -5)},
		{ReceiptNumber: "RC-2026-0016", Amount: 4500, Subtotal: p2.Subtotal, VAT: p2.VAT, PaidAt: paid, PaidBefore: 4500},
	}
	company := settings.Company{Name: "Smile Clean CO.,LTD.", TaxID: "0245562004320", VATRegistered: true}
	pdf, err := renderSummaryReceiptPDF(inv, rcs, company, settings.PaymentSettings{})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(pdf), "/Type /Page\n"); n != 2 {
		t.Errorf("summary receipt pages = %d, want 2 (original + copy)", n)
	}
	if out := os.Getenv("SUMMARY_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, pdf, 0o644)
	}

	inv.Status = StatusPartiallyPaid
	if _, err := renderSummaryReceiptPDF(inv, rcs[:1], company, settings.PaymentSettings{}); !errors.Is(err, ErrInvoiceNotPaid) {
		t.Errorf("summary for a part-paid invoice: %v", err)
	}
}

// TestLedgerDB exercises the payment ledger against a real database: partial
// payments, receipts, overpayment, refund and void protection. It skips
// unless TEST_DATABASE_URL points at a migrated database.
func TestLedgerDB(t *testing.T) {
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

	var bookingID int64
	number := "BK-LEDGER-" + time.Now().Format("150405.000")
	if err := pool.QueryRow(ctx,
		`INSERT INTO bookings (booking_number, customer_name, service_type, scheduled_for, duration_minutes, address, status, price)
		 VALUES ($1, 'Ledger Test Co', 'office_cleaning', now(), 120, '1 Test Rd', 'completed', 10000) RETURNING id`,
		number).Scan(&bookingID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM receipts WHERE invoice_id IN (SELECT id FROM invoices WHERE booking_number = $1)`, number)
		_, _ = pool.Exec(ctx, `DELETE FROM payments WHERE booking_number = $1`, number)
		_, _ = pool.Exec(ctx, `DELETE FROM invoices WHERE booking_number = $1`, number)
		_, _ = pool.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, bookingID)
	})

	inv, err := repo.Create(ctx, Invoice{BookingID: bookingID, BookingNumber: number, CustomerName: "Ledger Test Co",
		ServiceType: "office_cleaning", ServiceName: "Office", Subtotal: 10000, TaxRate: 7, TaxAmount: 700, Total: 10700,
		Currency: "THB", Status: StatusIssued, WithholdingRate: 3, WithholdingAmount: 300})
	if err != nil {
		t.Fatal(err)
	}

	in := func(a float64) PaymentInput {
		return PaymentInput{Amount: a, Method: "bank_transfer", PaidAt: time.Now(), VATRegistered: true}
	}
	inv1, rc1, err := repo.RecordPayment(ctx, inv.ID, in(4000))
	if err != nil {
		t.Fatal(err)
	}
	if inv1.Status != StatusPartiallyPaid || inv1.AmountPaid != 4000 || inv1.BalanceDue() != 6400 {
		t.Fatalf("after partial: status=%s paid=%v balance=%v", inv1.Status, inv1.AmountPaid, inv1.BalanceDue())
	}
	if !near(rc1.Subtotal+rc1.VAT-rc1.WHT, 4000) || rc1.ReceiptNumber == "" {
		t.Fatalf("receipt 1 = %+v", rc1)
	}

	if _, _, err := repo.RecordPayment(ctx, inv.ID, in(6400.5)); !errors.Is(err, ErrOverpayment) {
		t.Fatalf("overpayment: %v", err)
	}
	if _, err := repo.Update(ctx, inv.ID, StatusVoid); !errors.Is(err, ErrInvoiceHasPaid) {
		t.Fatalf("void with payments: %v", err)
	}

	inv2, rc2, err := repo.RecordPayment(ctx, inv.ID, in(6400))
	if err != nil {
		t.Fatal(err)
	}
	if inv2.Status != StatusPaid || inv2.PaidAt == nil {
		t.Fatalf("after final payment: %s", inv2.Status)
	}
	if !near(rc1.Subtotal+rc2.Subtotal, 10000) || !near(rc1.VAT+rc2.VAT, 700) || !near(rc1.WHT+rc2.WHT, 300) {
		t.Fatalf("receipts do not add up: %+v %+v", rc1, rc2)
	}
	if _, _, err := repo.RecordPayment(ctx, inv.ID, in(1)); !errors.Is(err, ErrInvoiceFullyPaid) {
		t.Fatalf("payment on paid invoice: %v", err)
	}

	reopened, err := repo.RefundPayment(ctx, rc2.PaymentID)
	if err != nil {
		t.Fatal(err)
	}
	if reopened == nil || reopened.Status != StatusPartiallyPaid || reopened.AmountPaid != 4000 || reopened.PaidAt != nil {
		t.Fatalf("after refund: %+v", reopened)
	}
	cancelled, err := repo.GetReceipt(ctx, rc2.ID)
	if err != nil || cancelled.Status != ReceiptCancelled {
		t.Fatalf("refunded receipt: %v %s", err, cancelled.Status)
	}

	// A pending payment linked to the invoice issues its receipt when settled.
	var pendingID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO payments (invoice_number, invoice_id, customer_name, booking_number, amount, currency, method, status)
		 VALUES ($1, $2, 'Ledger Test Co', $3, 6400, 'THB', 'cash', 'pending') RETURNING id`,
		inv.InvoiceNumber, inv.ID, number).Scan(&pendingID); err != nil {
		t.Fatal(err)
	}
	settled, err := repo.SettlePayment(ctx, pendingID, false)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Invoice == nil || settled.Invoice.Status != StatusPaid || settled.Receipt == nil {
		t.Fatalf("settle: %+v", settled)
	}
	if !near(rc1.Subtotal+settled.Receipt.Subtotal, 10000) {
		t.Fatalf("settled receipt subtotal %v", settled.Receipt.Subtotal)
	}
	list, total, err := repo.ListReceipts(ctx, paramsAll(), inv.ID)
	if err != nil || total != 3 || len(list) != 3 {
		t.Fatalf("receipts for invoice: %d (%v)", total, err)
	}

	// Deleting the booking nulls invoices.booking_id; the invoice and its
	// receipts must still load (this was a 500 in production).
	if _, err := pool.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, bookingID); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetByID(ctx, inv.ID); err != nil || got.BookingID != 0 || got.BookingNumber != number {
		t.Fatalf("invoice after booking delete: %+v %v", got, err)
	}
	if _, _, err := repo.List(ctx, paramsAll()); err != nil {
		t.Fatalf("list invoices after booking delete: %v", err)
	}
	if _, _, err := repo.ListReceipts(ctx, paramsAll(), inv.ID); err != nil {
		t.Fatalf("list receipts after booking delete: %v", err)
	}
}

func paramsAll() pagination.Params { return pagination.Params{Limit: 50} }
