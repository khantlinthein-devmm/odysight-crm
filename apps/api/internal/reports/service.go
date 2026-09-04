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
