package supplies

import (
	"testing"
	"time"
)

func TestRangeDefaultsAndInclusiveEnd(t *testing.T) {
	now := time.Date(2026, 9, 26, 15, 0, 0, 0, time.Local)
	from, to, err := Range("", "", now)
	if err != nil || from.Day() != 1 || to.Day() != 27 {
		t.Fatalf("default range = %v..%v err=%v", from, to, err)
	}
	if _, _, err := Range("2026-09-10", "2026-09-01", now); err == nil {
		t.Fatal("reversed range should fail")
	}
}

func TestRound2(t *testing.T) {
	if round2(12.345) != 12.35 || round2(-3.333) != -3.33 {
		t.Fatalf("round2: %v %v", round2(12.345), round2(-3.333))
	}
}

func TestInsufficientStockMessage(t *testing.T) {
	if got := (&InsufficientStock{Available: 2.5}).Error(); got != "only 2.5 in stock; record a purchase or adjustment first" {
		t.Fatal(got)
	}
}
