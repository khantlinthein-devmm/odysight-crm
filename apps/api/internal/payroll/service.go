package payroll

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/odysight/crm/pkg/response"
)

const dateFormat = "2006-01-02"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Report is payroll for a date range.
type Report struct {
	From, To string
	Lines    []Line
	Total    float64
}

// Period parses a YYYY-MM-DD range (defaults: this month to date).
func Period(fromRaw, toRaw string, now time.Time) (time.Time, time.Time, error) {
	now = now.Local()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	var err error
	if s := strings.TrimSpace(fromRaw); s != "" {
		if from, err = time.ParseInLocation(dateFormat, s, time.Local); err != nil {
			return time.Time{}, time.Time{}, response.NewAPIError(400, "from must be YYYY-MM-DD")
		}
	}
	if s := strings.TrimSpace(toRaw); s != "" {
		if to, err = time.ParseInLocation(dateFormat, s, time.Local); err != nil {
			return time.Time{}, time.Time{}, response.NewAPIError(400, "to must be YYYY-MM-DD")
		}
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, response.NewAPIError(400, "to must not be before from")
	}
	if to.Sub(from) > 366*24*time.Hour {
		return time.Time{}, time.Time{}, response.NewAPIError(400, "period must be a year or shorter")
	}
	return from, to, nil
}

func (s *Service) Report(ctx context.Context, from, to time.Time) (Report, error) {
	cleaners, err := s.repo.Cleaners(ctx)
	if err != nil {
		return Report{}, err
	}
	rates, err := s.repo.Rates(ctx)
	if err != nil {
		return Report{}, err
	}
	shifts, err := s.repo.Shifts(ctx, from.Format(dateFormat), to.Format(dateFormat))
	if err != nil {
		return Report{}, err
	}
	jobs, err := s.repo.CompletedJobs(ctx, from, to.AddDate(0, 0, 1))
	if err != nil {
		return Report{}, err
	}
	periodDays := int(to.Sub(from).Hours()/24) + 1
	rep := Report{From: from.Format(dateFormat), To: to.Format(dateFormat), Lines: []Line{}}
	for _, c := range cleaners {
		var rp *Rate
		if rt, ok := rates[c.ID]; ok {
			rp = &rt
		}
		// Inactive cleaners without any work in the period are noise.
		if c.Status == "inactive" && len(shifts[c.ID]) == 0 && jobs[c.ID] == 0 {
			continue
		}
		l := Compute(rp, shifts[c.ID], jobs[c.ID], periodDays)
		l.CleanerID, l.Name = c.ID, c.Name
		rep.Lines = append(rep.Lines, l)
		rep.Total += l.Total
	}
	rep.Total = round2(rep.Total)
	return rep, nil
}

func (s *Service) Rates(ctx context.Context) (map[int64]Rate, error) {
	return s.repo.Rates(ctx)
}

// RateRequest is the body for PATCH /payroll/rates/{cleanerId}.
type RateRequest struct {
	PayType       PayType  `json:"payType"`
	Rate          float64  `json:"rate"`
	OTMultiplier  *float64 `json:"otMultiplier"`
	StandardHours *float64 `json:"standardHours"`
}

func (s *Service) SetRate(ctx context.Context, cleanerID, userID int64, req RateRequest) (Rate, error) {
	if !req.PayType.Valid() {
		return Rate{}, response.NewAPIError(400, "payType must be hourly, daily, per_job or monthly")
	}
	if req.Rate < 0 || req.Rate > 1_000_000 {
		return Rate{}, response.NewAPIError(400, "rate must be between 0 and 1,000,000")
	}
	rt := Rate{CleanerID: cleanerID, PayType: req.PayType, Rate: req.Rate, OTMultiplier: 1.5, StandardHours: 8}
	if req.OTMultiplier != nil {
		if *req.OTMultiplier < 1 || *req.OTMultiplier > 5 {
			return Rate{}, response.NewAPIError(400, "otMultiplier must be between 1 and 5")
		}
		rt.OTMultiplier = *req.OTMultiplier
	}
	if req.StandardHours != nil {
		if *req.StandardHours <= 0 || *req.StandardHours > 24 {
			return Rate{}, response.NewAPIError(400, "standardHours must be between 0 and 24")
		}
		rt.StandardHours = *req.StandardHours
	}
	saved, err := s.repo.UpsertRate(ctx, rt, userID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return Rate{}, response.NewAPIError(404, "cleaner not found")
	}
	return saved, err
}
