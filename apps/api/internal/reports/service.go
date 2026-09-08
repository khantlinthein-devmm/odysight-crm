package reports

import (
	"context"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSummary(ctx context.Context) (Summary, error) {
	return s.repo.LoadSummary(ctx, time.Now())
}

// GetFinancial returns the financial & tax report for the [from, to] window.
func (s *Service) GetFinancial(ctx context.Context, from, to time.Time, currency string) (FinancialReport, error) {
	return s.repo.LoadFinancial(ctx, from, to, currency)
}
