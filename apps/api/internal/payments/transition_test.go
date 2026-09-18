package payments

import "testing"

func TestAllowedTransition(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusPending, StatusPaid, true},
		{StatusPending, StatusFailed, true},
		{StatusPending, StatusRefunded, false},
		{StatusFailed, StatusPending, true},
		{StatusFailed, StatusPaid, true},
		{StatusFailed, StatusRefunded, false},
		{StatusPaid, StatusRefunded, true},
		{StatusPaid, StatusPending, false},
		{StatusPaid, StatusFailed, false},
		{StatusPaid, StatusPaid, false}, // same-status short-circuits before this check
		{StatusRefunded, StatusPaid, false},
		{StatusRefunded, StatusPending, false},
	}
	for _, c := range cases {
		if got := allowedTransition(c.from, c.to); got != c.want {
			t.Errorf("allowedTransition(%q -> %q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}
