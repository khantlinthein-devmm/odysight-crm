package contracts

import "testing"

func TestCreateContractValidation(t *testing.T) {
	req := CreateContractRequest{
		CustomerID: 1, Title: "Mall 12-month",
		StartDate: "2026-01-01", EndDate: "2026-12-31",
		SiteIDs: []int64{1, 2},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("valid contract rejected: %v", err)
	}
	if req.Status != "draft" {
		t.Fatalf("default status should be draft, got %q", req.Status)
	}
	if req.BillingFrequency != "monthly" {
		t.Fatalf("default billing should be monthly, got %q", req.BillingFrequency)
	}
}

func TestContractEndBeforeStartRejected(t *testing.T) {
	req := CreateContractRequest{
		CustomerID: 1, StartDate: "2026-12-31", EndDate: "2026-01-01",
	}
	if err := req.Validate(); err == nil {
		t.Fatal("end before start must be rejected")
	}
}

func TestContractFrequencies(t *testing.T) {
	for _, f := range []BillingFrequency{BillingMonthly, BillingQuarterly, BillingAnnual, BillingCustom} {
		if !f.Valid() {
			t.Errorf("frequency %q should be valid", f)
		}
	}
	if BillingFrequency("weekly").Valid() {
		t.Error("weekly billing must be rejected")
	}
}

func TestContractTransitions(t *testing.T) {
	allowed := [][2]Status{
		{StatusDraft, StatusActive},
		{StatusActive, StatusExpiring},
		{StatusActive, StatusExpired},
		{StatusExpiring, StatusRenewed},
		{StatusExpired, StatusRenewed},
		{StatusDraft, StatusCancelled},
	}
	for _, tc := range allowed {
		if !canTransition(tc[0], tc[1]) {
			t.Errorf("%s -> %s should be allowed", tc[0], tc[1])
		}
	}
	denied := [][2]Status{
		{StatusDraft, StatusExpired},
		{StatusDraft, StatusRenewed},
		{StatusExpired, StatusActive},
		{StatusCancelled, StatusDraft},
		{StatusRenewed, StatusActive},
	}
	for _, tc := range denied {
		if canTransition(tc[0], tc[1]) {
			t.Errorf("%s -> %s must be rejected", tc[0], tc[1])
		}
	}
}
