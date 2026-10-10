package complaints

import (
	"testing"
	"time"
)

func TestDueAtBySeverity(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	for sev, h := range map[string]int{"high": 24, "medium": 48, "low": 72, "bogus": 48} {
		if got := DueAt(sev, t0); !got.Equal(t0.Add(time.Duration(h) * time.Hour)) {
			t.Errorf("DueAt(%s) = %v", sev, got)
		}
	}
}

func TestOverdue(t *testing.T) {
	due := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	after := due.Add(time.Minute)
	if !(Complaint{Status: StatusOpen, DueAt: due}).Overdue(after) {
		t.Error("open past due should be overdue")
	}
	if (Complaint{Status: StatusResolved, DueAt: due}).Overdue(after) {
		t.Error("resolved is never overdue")
	}
	if (Complaint{Status: StatusInProgress, DueAt: due}).Overdue(due.Add(-time.Minute)) {
		t.Error("not yet due")
	}
}
