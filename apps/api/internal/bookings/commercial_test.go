package bookings

import "testing"

// One-time bookings must work with no site and no contract.
func TestOneTimeBookingWithoutContract(t *testing.T) {
	req := CreateBookingRequest{
		CustomerName: "ABC Restaurant", ServiceType: "office_cleaning",
		ScheduledFor: "2026-09-20T09:00:00Z", DurationMinutes: 120, Status: "pending",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("one-time booking without contract rejected: %v", err)
	}
}

// Recurring bookings must work without a contract.
func TestRecurringBookingWithoutContract(t *testing.T) {
	req := CreateBookingRequest{
		CustomerName: "ABC Restaurant", ServiceType: "office_cleaning",
		ScheduledFor: "2026-09-22T09:00:00Z", DurationMinutes: 120, Status: "pending",
		IsRecurring: true, Recurrence: "weekly",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("recurring booking without contract rejected: %v", err)
	}
}

// Contract-based recurring bookings carry both optional links.
func TestContractBasedRecurringBooking(t *testing.T) {
	site, contract := int64(7), int64(3)
	req := CreateBookingRequest{
		CustomerName: "Mega Mall", CustomerID: &site,
		SiteID: &site, ContractID: &contract, ServiceType: "office_cleaning",
		ScheduledFor: "2026-09-22T09:00:00Z", DurationMinutes: 240, Status: "pending",
		IsRecurring: true, Recurrence: "weekly",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("contract booking rejected: %v", err)
	}
}

func TestRecurringFrequencies(t *testing.T) {
	for _, f := range []string{"weekly", "biweekly", "monthly"} {
		if !IsValidRecurrence(f) {
			t.Errorf("frequency %q should be valid", f)
		}
	}
	if IsValidRecurrence("daily") {
		t.Error("daily recurrence must be rejected")
	}
}
