// Package payroll turns attendance and completed jobs into cleaner pay for a
// date range, using each cleaner's pay rate.
package payroll

import (
	"math"
	"time"
)

// PayType is how a cleaner is paid.
type PayType string

const (
	PayHourly  PayType = "hourly"
	PayDaily   PayType = "daily"
	PayPerJob  PayType = "per_job"
	PayMonthly PayType = "monthly"
)

func (p PayType) Valid() bool {
	switch p {
	case PayHourly, PayDaily, PayPerJob, PayMonthly:
		return true
	}
	return false
}

// Rate is one cleaner's pay terms.
type Rate struct {
	CleanerID     int64
	PayType       PayType
	Rate          float64
	OTMultiplier  float64
	StandardHours float64
	UpdatedAt     time.Time
}

// Shift is one attendance day.
type Shift struct {
	CheckIn  *time.Time
	CheckOut *time.Time
}

// maxShiftHours caps a single day so a forgotten check-out the next morning
// cannot turn into a 30-hour shift.
const maxShiftHours = 16

// Line is one cleaner's pay for the period.
type Line struct {
	CleanerID    int64
	Name         string
	PayType      PayType
	Rate         float64
	HasRate      bool
	DaysWorked   int
	OpenDays     int // checked in but never checked out: not paid, flagged
	Hours        float64
	RegularHours float64
	OTHours      float64
	Jobs         int
	BasePay      float64
	OTPay        float64
	Total        float64
}

// Compute works out one cleaner's pay. periodDays is the length of the
// report range (used to prorate monthly salaries over 30-day months).
func Compute(rate *Rate, shifts []Shift, jobs, periodDays int) Line {
	l := Line{Jobs: jobs}
	std := 8.0
	mult := 1.5
	if rate != nil {
		l.HasRate, l.PayType, l.Rate = true, rate.PayType, rate.Rate
		if rate.StandardHours > 0 {
			std = rate.StandardHours
		}
		if rate.OTMultiplier >= 1 {
			mult = rate.OTMultiplier
		}
	}
	for _, s := range shifts {
		if s.CheckIn == nil {
			continue
		}
		if s.CheckOut == nil || !s.CheckOut.After(*s.CheckIn) {
			l.OpenDays++
			continue
		}
		h := math.Min(s.CheckOut.Sub(*s.CheckIn).Hours(), maxShiftHours)
		l.DaysWorked++
		l.Hours += h
		l.RegularHours += math.Min(h, std)
		l.OTHours += math.Max(0, h-std)
	}
	if rate != nil {
		switch rate.PayType {
		case PayHourly:
			l.BasePay = l.RegularHours * rate.Rate
			l.OTPay = l.OTHours * rate.Rate * mult
		case PayDaily:
			l.BasePay = float64(l.DaysWorked) * rate.Rate
			l.OTPay = l.OTHours * (rate.Rate / std) * mult
		case PayPerJob:
			l.BasePay = float64(jobs) * rate.Rate
		case PayMonthly:
			l.BasePay = rate.Rate * float64(periodDays) / 30
			l.OTPay = l.OTHours * (rate.Rate / 30 / std) * mult
		}
	}
	l.Hours, l.RegularHours, l.OTHours = round2(l.Hours), round2(l.RegularHours), round2(l.OTHours)
	l.BasePay, l.OTPay = round2(l.BasePay), round2(l.OTPay)
	l.Total = round2(l.BasePay + l.OTPay)
	return l
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
