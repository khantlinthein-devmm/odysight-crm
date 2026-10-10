package bookings

import "testing"

func strp(s string) *string { return &s }

func TestAssignedTo(t *testing.T) {
	me := CleanerIdentity{ID: 7, Name: "Mali C"}
	if !assignedTo(Booking{Cleaners: []CleanerBrief{{ID: 3}, {ID: 7}}}, me) {
		t.Error("crew member should be assigned")
	}
	if assignedTo(Booking{Cleaners: []CleanerBrief{{ID: 3}}, AssignedCleaner: "Mali C"}, me) {
		t.Error("name match must not override an explicit crew list")
	}
	if !assignedTo(Booking{AssignedCleaner: "mali c"}, me) {
		t.Error("legacy name-only assignment should match")
	}
	if assignedTo(Booking{}, CleanerIdentity{ID: 7}) {
		t.Error("empty names must never match")
	}
}

func TestInPool(t *testing.T) {
	if !inPool(Booking{Status: StatusPending}) {
		t.Error("pending unassigned booking is in the pool")
	}
	if inPool(Booking{Status: StatusPending, Cleaners: []CleanerBrief{{ID: 1}}}) {
		t.Error("assigned booking is not in the pool")
	}
	if inPool(Booking{Status: StatusConfirmed}) {
		t.Error("confirmed booking is not in the pool")
	}
}

func TestCheckCleanerUpdate(t *testing.T) {
	me := CleanerIdentity{ID: 7, Name: "Mali C"}
	mine := Booking{Cleaners: []CleanerBrief{{ID: 7}}}
	cases := []struct {
		name string
		b    Booking
		req  UpdateBookingRequest
		ok   bool
	}{
		{"complete own job", mine, UpdateBookingRequest{Status: strp("completed")}, true},
		{"start own job with notes", mine, UpdateBookingRequest{Status: strp("in_progress"), Notes: strp("gate code 12")}, true},
		{"someone else's job", Booking{Cleaners: []CleanerBrief{{ID: 8}}}, UpdateBookingRequest{Status: strp("completed")}, false},
		{"cancel own job", mine, UpdateBookingRequest{Status: strp("cancelled")}, false},
		{"reassign crew", mine, UpdateBookingRequest{CleanerIDs: []int64{9}}, false},
		{"move schedule", mine, UpdateBookingRequest{ScheduledFor: strp("2026-10-01T09:00:00Z")}, false},
	}
	for _, tc := range cases {
		err := checkCleanerUpdate(tc.b, me, tc.req)
		if (err == nil) != tc.ok {
			t.Errorf("%s: err=%v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}
