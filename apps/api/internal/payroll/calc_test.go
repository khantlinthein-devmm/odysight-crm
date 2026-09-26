package payroll

import (
	"testing"
	"time"
)

func shift(day int, inH, outH float64) Shift {
	in := time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC).Add(time.Duration(inH * float64(time.Hour)))
	out := time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC).Add(time.Duration(outH * float64(time.Hour)))
	return Shift{CheckIn: &in, CheckOut: &out}
}

func TestComputeHourlyWithOvertime(t *testing.T) {
	r := &Rate{PayType: PayHourly, Rate: 60, OTMultiplier: 1.5, StandardHours: 8}
	// 8h + 10h (2h OT) + an open day that is not paid.
	in := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	l := Compute(r, []Shift{shift(1, 9, 17), shift(2, 8, 18), {CheckIn: &in}}, 0, 30)
	if l.DaysWorked != 2 || l.OpenDays != 1 || l.RegularHours != 16 || l.OTHours != 2 {
		t.Fatalf("hours: %+v", l)
	}
	if l.BasePay != 960 || l.OTPay != 180 || l.Total != 1140 {
		t.Fatalf("pay: base %v ot %v total %v", l.BasePay, l.OTPay, l.Total)
	}
}

func TestComputeDaily(t *testing.T) {
	r := &Rate{PayType: PayDaily, Rate: 500, OTMultiplier: 1.5, StandardHours: 8}
	l := Compute(r, []Shift{shift(1, 8, 17), shift(2, 8, 16)}, 0, 7)
	// 9h day → 1h OT at (500/8)*1.5 = 93.75
	if l.BasePay != 1000 || l.OTPay != 93.75 || l.Total != 1093.75 {
		t.Fatalf("daily: %+v", l)
	}
}

func TestComputePerJobAndMonthly(t *testing.T) {
	pj := Compute(&Rate{PayType: PayPerJob, Rate: 350}, []Shift{shift(1, 8, 20)}, 4, 7)
	if pj.Total != 1400 || pj.OTPay != 0 {
		t.Fatalf("per job: %+v", pj)
	}
	m := Compute(&Rate{PayType: PayMonthly, Rate: 15000, StandardHours: 8, OTMultiplier: 1.5}, nil, 0, 15)
	if m.BasePay != 7500 {
		t.Fatalf("monthly prorate: %+v", m)
	}
}

func TestComputeWithoutRateStillCountsHours(t *testing.T) {
	l := Compute(nil, []Shift{shift(1, 8, 12)}, 2, 7)
	if l.HasRate || l.Hours != 4 || l.Total != 0 || l.Jobs != 2 {
		t.Fatalf("no rate: %+v", l)
	}
}

func TestComputeCapsRunawayShift(t *testing.T) {
	in := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	out := in.Add(30 * time.Hour)
	l := Compute(&Rate{PayType: PayHourly, Rate: 10, StandardHours: 8, OTMultiplier: 1.5}, []Shift{{CheckIn: &in, CheckOut: &out}}, 0, 7)
	if l.Hours != maxShiftHours {
		t.Fatalf("hours = %v, want capped at %d", l.Hours, maxShiftHours)
	}
}
