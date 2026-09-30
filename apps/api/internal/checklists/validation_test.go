package checklists

import "testing"

func TestChecklistCreateValidation(t *testing.T) {
	req := CreateChecklistRequest{BookingID: 9, Items: []string{"Lobby swept", "Bins emptied"}}
	if err := req.Validate(); err != nil {
		t.Fatalf("adhoc checklist rejected: %v", err)
	}
	tpl := int64(2)
	req = CreateChecklistRequest{BookingID: 9, TemplateID: &tpl}
	if err := req.Validate(); err != nil {
		t.Fatalf("template checklist rejected: %v", err)
	}
	// Neither template nor items: the service picks the booking's template.
	req = CreateChecklistRequest{BookingID: 9}
	if err := req.Validate(); err != nil {
		t.Fatalf("checklist with automatic template rejected: %v", err)
	}
	req = CreateChecklistRequest{}
	if err := req.Validate(); err == nil {
		t.Fatal("checklist without a booking must be rejected")
	}
}

func TestChecklistCompletionAndConfirmation(t *testing.T) {
	if !StatusPending.Valid() || !StatusInProgress.Valid() || !StatusCompleted.Valid() {
		t.Fatal("checklist statuses must be valid")
	}
	req := ConfirmChecklistRequest{ClientSignature: "  "}
	if err := req.Validate(); err == nil {
		t.Fatal("blank signature must be rejected")
	}
	tpl := CreateTemplateRequest{Name: "Factory Deep Cleaning", Items: []string{"Production area cleaned", "Floor cleaned"}}
	if err := tpl.Validate(); err != nil {
		t.Fatalf("factory template rejected: %v", err)
	}
}
