package quotes

import "testing"

func item(name string, qty, price float64) QuoteItemInput {
	return QuoteItemInput{ServiceName: name, Quantity: qty, UnitPrice: price}
}

func TestQuoteTotalsFromPriceCalculatorInputs(t *testing.T) {
	// Quote persists what the price calculator computes: no duplicated logic,
	// just quantity x unit price plus tax.
	sub, total := totals([]QuoteItemInput{item("Deep clean", 2, 1500), item("Windows", 1, 800)}, 7)
	if sub != 3800 {
		t.Fatalf("subtotal = %v, want 3800", sub)
	}
	if total != 4066 {
		t.Fatalf("total = %v, want 4066", total)
	}
}

func TestQuoteRequiresItemsNotContract(t *testing.T) {
	req := CreateQuoteRequest{CustomerID: 5, Items: []QuoteItemInput{item("Clean", 1, 100)}}
	if err := req.Validate(); err != nil {
		t.Fatalf("quote without contract rejected: %v", err)
	}
	req = CreateQuoteRequest{CustomerID: 5}
	if err := req.Validate(); err == nil {
		t.Fatal("quote without items must be rejected")
	}
}

func TestQuoteStatusEnum(t *testing.T) {
	for _, s := range []Status{StatusDraft, StatusSent, StatusAccepted, StatusRejected, StatusExpired} {
		if !s.Valid() {
			t.Errorf("status %q should be valid", s)
		}
	}
}
