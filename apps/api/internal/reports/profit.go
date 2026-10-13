package reports

import (
	"context"
	"math"
)

// Booking profit estimates what one job earns after the team's pay and the
// supplies booked against it:
//
//	profit = price (before VAT) − labor − supplies
//
// Labor uses each assigned cleaner's pay rate (Payroll → pay rates) and the
// booking's planned duration, so it is an estimate: overtime, travel and
// actual check-in hours are not included.

// workDaysPerMonth converts a monthly salary to a day rate for allocation.
const workDaysPerMonth = 26

// MaxProfitBookings caps one booking-profit request.
const MaxProfitBookings = 200

// LaborLine is one cleaner's estimated pay for the job. Rate and PayType are
// only sent to roles that may see pay rates (payroll.read).
type LaborLine struct {
	CleanerID int64   `json:"cleanerId"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	PayType   string  `json:"payType,omitempty"`
	Rate      float64 `json:"rate,omitempty"`
	Hours     float64 `json:"hours"`
	Cost      float64 `json:"cost"`
	HasRate   bool    `json:"hasRate"`
}

// BookingProfit is the estimate for one booking. Price and Profit are nil
// when the booking has no price yet.
type BookingProfit struct {
	BookingID    int64       `json:"bookingId"`
	Price        *float64    `json:"price"`
	Labor        float64     `json:"labor"`
	Supplies     float64     `json:"supplies"`
	Cost         float64     `json:"cost"`
	Profit       *float64    `json:"profit"`
	Margin       *float64    `json:"margin"`
	MissingRates int         `json:"missingRates"`
	Team         []LaborLine `json:"team"`
}

// PayRate mirrors a row of cleaner_pay_rates.
type PayRate struct {
	PayType       string
	Rate          float64
	StandardHours float64
}

// TeamMember is one cleaner assigned to a booking.
type TeamMember struct {
	CleanerID int64
	Name      string
	Role      string
}

// laborCost estimates one cleaner's pay for a job lasting hours.
func laborCost(rate PayRate, hours float64) float64 {
	std := rate.StandardHours
	if std <= 0 {
		std = 8
	}
	switch rate.PayType {
	case "hourly":
		return rate.Rate * hours
	case "daily":
		return rate.Rate * hours / std
	case "per_job":
		return rate.Rate
	case "monthly":
		return rate.Rate / workDaysPerMonth * hours / std
	}
	return 0
}

// computeProfit builds the estimate for one booking.
func computeProfit(bookingID int64, price *float64, durationMinutes int, team []TeamMember,
	rates map[int64]PayRate, supplies float64) BookingProfit {
	hours := round2(float64(durationMinutes) / 60)
	out := BookingProfit{BookingID: bookingID, Price: price, Supplies: round2(supplies), Team: []LaborLine{}}
	for _, m := range team {
		line := LaborLine{CleanerID: m.CleanerID, Name: m.Name, Role: m.Role, Hours: hours}
		if rt, ok := rates[m.CleanerID]; ok {
			line.HasRate = true
			line.PayType = rt.PayType
			line.Rate = rt.Rate
			line.Cost = round2(laborCost(rt, float64(durationMinutes)/60))
		} else {
			out.MissingRates++
		}
		out.Labor += line.Cost
		out.Team = append(out.Team, line)
	}
	out.Labor = round2(out.Labor)
	out.Cost = round2(out.Labor + out.Supplies)
	if price != nil {
		p := round2(*price - out.Cost)
		out.Profit = &p
		if *price > 0 {
			m := round2(p / *price * 100)
			out.Margin = &m
		}
	}
	return out
}

// hideRates strips per-cleaner pay details for roles without payroll.read,
// leaving only the totals.
func hideRates(p BookingProfit) BookingProfit {
	team := make([]LaborLine, len(p.Team))
	for i, l := range p.Team {
		l.PayType, l.Rate, l.Cost = "", 0, 0
		team[i] = l
	}
	p.Team = team
	return p
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// BookingProfits estimates profit for each existing booking in ids. Roles
// without payroll.read get totals only, never individual pay rates.
func (s *Service) BookingProfits(ctx context.Context, ids []int64, showRates bool) ([]BookingProfit, error) {
	if len(ids) == 0 {
		return []BookingProfit{}, nil
	}
	bookings, team, rates, supplies, err := s.repo.ProfitInputs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]BookingProfit, 0, len(bookings))
	for _, b := range bookings {
		p := computeProfit(b.ID, b.Price, b.DurationMinutes, team[b.ID], rates, supplies[b.ID])
		if !showRates {
			p = hideRates(p)
		}
		out = append(out, p)
	}
	return out, nil
}
