package reports

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/odysight/crm/pkg/response"
)

// Booking profit is worked out from costs the office enters on the booking
// itself (not payroll):
//
//	profit = price (before VAT) − cleaners fee − transportation − other − supplies used
//
// The cleaners fee is either one amount for the whole job team (TeamFee)
// or a fee per cleaner.
//
// Supplies come from usage movements recorded against the booking.

// MaxProfitBookings caps one booking-profit request.
const MaxProfitBookings = 200

// maxCost guards against typos such as a few extra zeros.
const maxCost = 10_000_000

// LaborLine is one team member's fee for the job.
type LaborLine struct {
	CleanerID int64   `json:"cleanerId"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	Fee       float64 `json:"fee"`
	HasFee    bool    `json:"hasFee"`
}

// BookingProfit is the estimate for one booking. Price and Profit are nil
// when the booking has no price yet.
type BookingProfit struct {
	BookingID   int64       `json:"bookingId"`
	Price       *float64    `json:"price"`
	Labor       float64     `json:"labor"`
	TeamFee     *float64    `json:"teamFee"`
	Transport   float64     `json:"transport"`
	Other       float64     `json:"other"`
	OtherNote   string      `json:"otherNote"`
	Supplies    float64     `json:"supplies"`
	Cost        float64     `json:"cost"`
	Profit      *float64    `json:"profit"`
	Margin      *float64    `json:"margin"`
	MissingFees int         `json:"missingFees"`
	Team        []LaborLine `json:"team"`
}

// TeamMember is one cleaner assigned to a booking.
type TeamMember struct {
	CleanerID int64
	Name      string
	Role      string
}

// BookingCosts are the office-entered costs of one booking.
type BookingCosts struct {
	// TeamFee, when set, is the cleaners fee for the whole team and
	// CleanerFees is ignored.
	TeamFee     *float64
	CleanerFees map[int64]float64
	Transport   float64
	Other       float64
	OtherNote   string
}

// computeProfit builds the estimate for one booking. Fees only count for
// cleaners currently on the team, so removing someone drops their fee.
func computeProfit(bookingID int64, price *float64, team []TeamMember, costs BookingCosts, supplies float64) BookingProfit {
	out := BookingProfit{
		BookingID: bookingID,
		Price:     price,
		Transport: round2(costs.Transport),
		Other:     round2(costs.Other),
		OtherNote: costs.OtherNote,
		Supplies:  round2(supplies),
		Team:      []LaborLine{},
	}
	if costs.TeamFee != nil {
		fee := round2(*costs.TeamFee)
		out.TeamFee = &fee
		out.Labor = fee
	}
	for _, m := range team {
		line := LaborLine{CleanerID: m.CleanerID, Name: m.Name, Role: m.Role}
		if out.TeamFee != nil {
			out.Team = append(out.Team, line)
			continue
		}
		if fee, ok := costs.CleanerFees[m.CleanerID]; ok {
			line.Fee, line.HasFee = round2(fee), true
		} else {
			out.MissingFees++
		}
		out.Labor += line.Fee
		out.Team = append(out.Team, line)
	}
	out.Labor = round2(out.Labor)
	out.Cost = round2(out.Labor + out.Transport + out.Other + out.Supplies)
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

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// BookingProfits estimates profit for each existing booking in ids.
func (s *Service) BookingProfits(ctx context.Context, ids []int64) ([]BookingProfit, error) {
	if len(ids) == 0 {
		return []BookingProfit{}, nil
	}
	bookings, team, costs, supplies, err := s.repo.ProfitInputs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]BookingProfit, 0, len(bookings))
	for _, b := range bookings {
		out = append(out, computeProfit(b.ID, b.Price, team[b.ID], costs[b.ID], supplies[b.ID]))
	}
	return out, nil
}

// SaveCostsRequest is the body of PUT /reports/booking-costs/{id}. Fees are
// keyed by cleaner id (JSON object keys are strings).
type SaveCostsRequest struct {
	TeamFee     *float64           `json:"teamFee"`
	CleanerFees map[string]float64 `json:"cleanerFees"`
	Transport   float64            `json:"transport"`
	Other       float64            `json:"other"`
	OtherNote   string             `json:"otherNote"`
}

func validCost(v float64) bool { return v >= 0 && v <= maxCost && !math.IsNaN(v) }

func (r SaveCostsRequest) parse() (BookingCosts, error) {
	c := BookingCosts{CleanerFees: map[int64]float64{}, Transport: r.Transport, Other: r.Other,
		OtherNote: strings.TrimSpace(r.OtherNote)}
	if r.TeamFee != nil {
		if !validCost(*r.TeamFee) {
			return c, response.NewAPIError(400, "costs must be between 0 and 10,000,000")
		}
		fee := *r.TeamFee
		c.TeamFee = &fee
	}
	if !validCost(c.Transport) || !validCost(c.Other) {
		return c, response.NewAPIError(400, "costs must be between 0 and 10,000,000")
	}
	if len([]rune(c.OtherNote)) > 200 {
		return c, response.NewAPIError(400, "otherNote must be at most 200 characters")
	}
	for k, v := range r.CleanerFees {
		id, err := strconv.ParseInt(k, 10, 64)
		if err != nil || id < 1 {
			return c, response.NewAPIError(400, "cleanerFees keys must be cleaner ids")
		}
		if !validCost(v) {
			return c, response.NewAPIError(400, "cleaner fees must be between 0 and 10,000,000")
		}
		c.CleanerFees[id] = v
	}
	return c, nil
}

// SaveBookingCosts stores the costs and returns the fresh estimate.
func (s *Service) SaveBookingCosts(ctx context.Context, bookingID, userID int64, req SaveCostsRequest) (BookingProfit, error) {
	costs, err := req.parse()
	if err != nil {
		return BookingProfit{}, err
	}
	if err := s.repo.SaveBookingCosts(ctx, bookingID, userID, costs); err != nil {
		return BookingProfit{}, err
	}
	out, err := s.BookingProfits(ctx, []int64{bookingID})
	if err != nil {
		return BookingProfit{}, err
	}
	if len(out) == 0 {
		return BookingProfit{}, response.NewAPIError(404, "booking not found")
	}
	return out[0], nil
}
