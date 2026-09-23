package invoices

import "testing"

func TestInvoiceIdempotencyKeyValidation(t *testing.T) {
	req := CreateInvoiceRequest{BookingID: 1}
	key := "contract-3:2026-09"
	req.IdempotencyKey = &key
	start, end := "2026-09-01", "2026-09-30"
	req.BillingPeriodStart = &start
	req.BillingPeriodEnd = &end
	if err := req.Validate(); err != nil {
		t.Fatalf("contract invoice with idempotency key rejected: %v", err)
	}
	blank := "  "
	req.IdempotencyKey = &blank
	if err := req.Validate(); err == nil {
		t.Fatal("blank idempotency key must be rejected")
	}
}

func TestInvoiceWithoutContractStillValid(t *testing.T) {
	req := CreateInvoiceRequest{BookingID: 4}
	if err := req.Validate(); err != nil {
		t.Fatalf("one-time invoice rejected: %v", err)
	}
}
